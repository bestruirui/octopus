package bodycache

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// 默认最大请求体大小：256MB
	DefaultBodyMaxMB = 256
	// 默认内存缓存阈值：16MB
	DefaultMemoryThresholdMB = 16
	// 默认临时目录：./cache
	DefaultTmpDir = "./cache"
	// 默认启动清理阈值：24小时
	DefaultTmpCleanupHours = 24

	// 临时文件名前缀：用于清理
	TmpFilePrefix = "octopus-images-"
)

const (
	envBodyMaxMB             = "OCTOPUS_IMAGES_BODY_MAX_MB"
	envMemoryThresholdMB     = "OCTOPUS_IMAGES_BODY_MEMORY_THRESHOLD_MB"
	envTmpDir                = "OCTOPUS_IMAGES_BODY_TMP_DIR"
	envTmpCleanupHours       = "OCTOPUS_IMAGES_BODY_TMP_CLEANUP_HOURS"
	bytesPerMB         int64 = 1024 * 1024
)

// BodyTooLargeError 表示读取请求体时超过最大限制。
// 上层可以通过 errors.As 判断该错误，用于返回 413 等逻辑。
type BodyTooLargeError struct {
	MaxBytes    int64
	ActualBytes int64
}

func (e *BodyTooLargeError) Error() string {
	return fmt.Sprintf("images body too large: max=%d actual=%d", e.MaxBytes, e.ActualBytes)
}

// BodyCache 是一个可重放的请求体缓存对象：小体积缓存内存，大体积落盘临时文件。
// 该对象需要 Close 释放资源并删除临时文件（若存在）。
type BodyCache struct {
	mu     sync.Mutex
	closed bool

	size int64

	// 仅当 isFile=false 时使用
	mem []byte

	// 仅当 isFile=true 时使用
	tmpPath string
}

// New 从 r 读取完整 body 并缓存（内存或临时文件）。
// - 最大限制默认 256MB，可由环境变量 OCTOPUS_IMAGES_BODY_MAX_MB 覆盖
// - 内存阈值默认 16MB，可由环境变量 OCTOPUS_IMAGES_BODY_MEMORY_THRESHOLD_MB 覆盖
// - 临时目录默认 ./cache，可由环境变量 OCTOPUS_IMAGES_BODY_TMP_DIR 覆盖
func New(r io.ReadCloser) (*BodyCache, error) {
	if r == nil {
		return nil, errors.New("nil body reader")
	}
	defer r.Close()

	maxBytes := BodyMaxBytesFromEnv()
	thresholdBytes := MemoryThresholdBytesFromEnv()
	tmpDir := TmpDirFromEnv()

	// 使用 maxBytes+1 的限制读取，用于准确判断是否超限。
	limited := io.LimitReader(r, maxBytes+1)

	sw := &spillWriter{
		thresholdBytes: thresholdBytes,
		tmpDir:         tmpDir,
		prefix:         TmpFilePrefix,
	}

	n, err := io.Copy(sw, limited)
	if err != nil {
		// 出错时确保清理已创建的临时文件
		_ = sw.Close()
		return nil, err
	}

	if n > maxBytes {
		_ = sw.Close()
		return nil, &BodyTooLargeError{
			MaxBytes:    maxBytes,
			ActualBytes: n,
		}
	}

	bc := &BodyCache{
		size:    n,
		mem:     sw.Bytes(),
		tmpPath: sw.TmpPath(),
	}
	// spillWriter.Close 只负责关闭文件句柄，不删除文件；删除由 BodyCache.Close 负责。
	if cerr := sw.Close(); cerr != nil {
		_ = bc.Close()
		return nil, cerr
	}
	return bc, nil
}

// NewReader 每次调用返回一个新的 reader，用于多次重试重放请求体。
func (b *BodyCache) NewReader() (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errors.New("body cache closed")
	}

	if b.tmpPath != "" {
		f, err := os.Open(b.tmpPath)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	// 内存模式：每次新建 bytes.Reader
	return io.NopCloser(bytes.NewReader(b.mem)), nil
}

// Size 返回缓存的 body 大小（字节）。
func (b *BodyCache) Size() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

// IsFile 表示是否落盘到临时文件。
func (b *BodyCache) IsFile() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tmpPath != ""
}

// TmpPath 返回临时文件路径（仅在 IsFile=true 时有效）。
func (b *BodyCache) TmpPath() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tmpPath
}

// Close 释放资源并删除临时文件（若存在）。可重复调用。
func (b *BodyCache) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true

	tmpPath := b.tmpPath
	b.tmpPath = ""
	b.mem = nil
	b.size = 0
	b.mu.Unlock()

	if tmpPath != "" {
		// 删除失败时把错误返回给上层；上层可记录日志但不阻断流程。
		if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// CleanupOldTmpFiles 删除 dir 目录下：文件名匹配 prefix 且 mtime 早于 olderThan 的文件。
func CleanupOldTmpFiles(dir string, prefix string, olderThan time.Duration) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("empty dir")
	}
	if olderThan <= 0 {
		return errors.New("olderThan must be positive")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		// 目录不存在时不视为错误（首次启动可能尚未产生缓存文件）
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	deadline := time.Now().Add(-olderThan)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			// 读取 info 失败时跳过，不阻断整体清理
			continue
		}
		if info.ModTime().Before(deadline) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}

// BodyMaxBytesFromEnv 返回最大请求体大小（字节）。
func BodyMaxBytesFromEnv() int64 {
	mb := envInt(envBodyMaxMB, DefaultBodyMaxMB)
	if mb <= 0 {
		mb = DefaultBodyMaxMB
	}
	return int64(mb) * bytesPerMB
}

// MemoryThresholdBytesFromEnv 返回内存阈值（字节）。
func MemoryThresholdBytesFromEnv() int64 {
	mb := envInt(envMemoryThresholdMB, DefaultMemoryThresholdMB)
	if mb <= 0 {
		mb = DefaultMemoryThresholdMB
	}
	return int64(mb) * bytesPerMB
}

// TmpDirFromEnv 返回临时目录路径。
func TmpDirFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(envTmpDir)); v != "" {
		return v
	}
	return DefaultTmpDir
}

// TmpCleanupOlderThanFromEnv 返回启动清理阈值时长。
func TmpCleanupOlderThanFromEnv() time.Duration {
	h := envInt(envTmpCleanupHours, DefaultTmpCleanupHours)
	if h <= 0 {
		h = DefaultTmpCleanupHours
	}
	return time.Duration(h) * time.Hour
}

func envInt(name string, def int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}

// spillWriter 在超过阈值时将数据从内存溢写到临时文件。
type spillWriter struct {
	thresholdBytes int64
	tmpDir         string
	prefix         string

	size int64

	buf bytes.Buffer

	f       *os.File
	tmpPath string
}

func (w *spillWriter) Write(p []byte) (int, error) {
	// 先判断是否需要落盘
	if w.f == nil && w.size+int64(len(p)) > w.thresholdBytes {
		if err := w.ensureFile(); err != nil {
			return 0, err
		}
	}

	var n int
	var err error
	if w.f != nil {
		n, err = w.f.Write(p)
	} else {
		n, err = w.buf.Write(p)
	}
	w.size += int64(n)
	return n, err
}

func (w *spillWriter) ensureFile() error {
	dir := w.tmpDir
	if strings.TrimSpace(dir) == "" {
		dir = DefaultTmpDir
	}

	// 确保临时目录存在
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	f, err := os.CreateTemp(dir, w.prefix+"*")
	if err != nil {
		return err
	}
	w.f = f
	w.tmpPath = f.Name()

	// 把 buffer 内容写入文件
	if w.buf.Len() > 0 {
		if _, err := w.f.Write(w.buf.Bytes()); err != nil {
			_ = w.f.Close()
			_ = os.Remove(w.tmpPath)
			w.f = nil
			w.tmpPath = ""
			return err
		}
		w.buf.Reset()
	}
	return nil
}

// Bytes 返回内存缓存的数据（仅在未落盘时有效）。
func (w *spillWriter) Bytes() []byte {
	if w.f != nil {
		return nil
	}
	// 复制一份，避免外部修改内部 buffer
	out := make([]byte, w.buf.Len())
	copy(out, w.buf.Bytes())
	return out
}

// TmpPath 返回临时文件路径（仅在已落盘时有效）。
func (w *spillWriter) TmpPath() string {
	return w.tmpPath
}

// Close 关闭文件句柄（不删除文件）。
func (w *spillWriter) Close() error {
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}