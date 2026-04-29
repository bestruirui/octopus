package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/auth"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
	"github.com/lingyuins/octopus/internal/task"
)

var (
	maxDBImportBytes               int64 = 64 << 20
	maxDBImportMultipartExtraBytes int64 = 1 << 20
)

func init() {
	router.NewGroupRouter("/api/v1/setting").
		Use(middleware.Auth()).
		Use(middleware.RequirePermission(auth.PermSettingsRead)).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(getSettingList),
		).
		AddRoute(
			router.NewRoute("/set", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermSettingsWrite)).
				Use(middleware.RequireJSON()).
				Handle(setSetting),
		).
		AddRoute(
			router.NewRoute("/export", http.MethodGet).
				Use(middleware.RequirePermission(auth.PermSettingsWrite)).
				Handle(exportDB),
		).
		AddRoute(
			router.NewRoute("/import", http.MethodPost).
				Use(middleware.RequirePermission(auth.PermSettingsWrite)).
				Handle(importDB),
		)
}

func getSettingList(c *gin.Context) {
	settings, err := op.SettingList(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, settings)
}

func setSetting(c *gin.Context) {
	var setting model.Setting
	if err := c.ShouldBindJSON(&setting); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := setting.Validate(); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := op.SettingSetString(setting.Key, setting.Value); err != nil {
		resp.InternalError(c)
		return
	}
	if shouldRefreshSemanticCacheRuntime(setting.Key) {
		if err := op.RefreshSemanticCacheRuntime(); err != nil {
			resp.InternalError(c)
			return
		}
	}
	switch setting.Key {
	case model.SettingKeyStatsSaveInterval:
		minutes, err := strconv.Atoi(setting.Value)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		interval := time.Duration(minutes) * time.Minute
		task.Update(task.TaskStatsSave, interval)
		task.Update(task.TaskRuntimeState, interval)
	case model.SettingKeyModelInfoUpdateInterval:
		hours, err := strconv.Atoi(setting.Value)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		task.Update(string(setting.Key), time.Duration(hours)*time.Hour)
	case model.SettingKeySyncLLMInterval:
		hours, err := strconv.Atoi(setting.Value)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		task.Update(string(setting.Key), time.Duration(hours)*time.Hour)
	}
	resp.Success(c, setting)
}

func shouldRefreshSemanticCacheRuntime(key model.SettingKey) bool {
	switch key {
	case model.SettingKeySemanticCacheEnabled,
		model.SettingKeySemanticCacheTTL,
		model.SettingKeySemanticCacheThreshold,
		model.SettingKeySemanticCacheMaxEntries,
		model.SettingKeySemanticCacheEmbeddingBaseURL,
		model.SettingKeySemanticCacheEmbeddingAPIKey,
		model.SettingKeySemanticCacheEmbeddingModel,
		model.SettingKeySemanticCacheEmbeddingTimeoutSeconds:
		return true
	default:
		return false
	}
}

func exportDB(c *gin.Context) {
	includeLogs, _ := strconv.ParseBool(c.DefaultQuery("include_logs", "false"))
	includeStats, _ := strconv.ParseBool(c.DefaultQuery("include_stats", "false"))

	dump, err := op.DBExportAll(c.Request.Context(), includeLogs, includeStats)
	if err != nil {
		resp.InternalError(c)
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=\"octopus-export-"+time.Now().Format("20060102150405")+".json\"")
	c.JSON(http.StatusOK, dump)
}

func importDB(c *gin.Context) {
	var dump model.DBDump
	defer cleanupDBImportMultipartForm(c)

	if err := readDBDump(c, &dump); err != nil {
		status := http.StatusBadRequest
		if isDBImportTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
		}
		resp.Error(c, status, err.Error())
		return
	}

	result, err := op.DBImportIncremental(c.Request.Context(), &dump)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := op.InitCache(); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	if err := op.RefreshSemanticCacheRuntime(); err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	resp.Success(c, result)
}

func decodeDBDump(body []byte, dump *model.DBDump) error {
	return decodeDBDumpReader(bytes.NewReader(body), dump)
}

func readDBDump(c *gin.Context, dump *model.DBDump) error {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		limitDBImportRequestBody(c)
		fh, err := c.FormFile("file")
		if err != nil {
			return normalizeDBImportMultipartError(err)
		}
		if fh.Size > 0 && fh.Size > maxDBImportBytes {
			return newDBImportTooLargeError()
		}

		f, err := fh.Open()
		if err != nil {
			return err
		}
		defer f.Close()

		return decodeDBDumpReader(f, dump)
	}

	return decodeDBDumpReader(c.Request.Body, dump)
}

func cleanupDBImportMultipartForm(c *gin.Context) {
	if c == nil || c.Request == nil || c.Request.MultipartForm == nil {
		return
	}
	_ = c.Request.MultipartForm.RemoveAll()
}

func limitDBImportRequestBody(c *gin.Context) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxDBImportBytes+maxDBImportMultipartExtraBytes)
}

func normalizeDBImportMultipartError(err error) error {
	if err == nil {
		return nil
	}
	if isHTTPMaxBytesError(err) {
		return newDBImportTooLargeError()
	}
	if errors.Is(err, http.ErrMissingFile) {
		return fmt.Errorf("missing upload file field 'file'")
	}
	return err
}

func decodeDBDumpReader(r io.Reader, dump *model.DBDump) error {
	limitedReader := &io.LimitedReader{R: r, N: maxDBImportBytes + 1}
	if dump == nil {
		var empty struct{}
		if err := json.NewDecoder(limitedReader).Decode(&empty); err != nil {
			if limitedReader.N <= 0 {
				return newDBImportTooLargeError()
			}
			return err
		}
		if limitedReader.N <= 0 {
			return newDBImportTooLargeError()
		}
		return nil
	}

	var envelope struct {
		model.DBDump
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(limitedReader).Decode(&envelope); err != nil {
		if limitedReader.N <= 0 {
			return newDBImportTooLargeError()
		}
		return err
	}
	if limitedReader.N <= 0 {
		return newDBImportTooLargeError()
	}

	*dump = envelope.DBDump

	if isEmptyDBDump(*dump) && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, dump); err != nil {
			return err
		}
	}

	return nil
}

func isEmptyDBDump(dump model.DBDump) bool {
	return dump.Version == 0 &&
		len(dump.Channels) == 0 &&
		len(dump.ChannelKeys) == 0 &&
		len(dump.Groups) == 0 &&
		len(dump.GroupItems) == 0 &&
		len(dump.Settings) == 0 &&
		len(dump.APIKeys) == 0 &&
		len(dump.LLMInfos) == 0 &&
		len(dump.RelayLogs) == 0 &&
		len(dump.StatsDaily) == 0 &&
		len(dump.StatsHourly) == 0 &&
		len(dump.StatsTotal) == 0 &&
		len(dump.StatsChannel) == 0 &&
		len(dump.StatsModel) == 0 &&
		len(dump.StatsAPIKey) == 0
}

func isDBImportTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "backup file exceeds")
}

func isHTTPMaxBytesError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func newDBImportTooLargeError() error {
	return fmt.Errorf("backup file exceeds %s import limit; retry without logs/stats or use a database-level backup for larger datasets", formatDBImportLimit(maxDBImportBytes))
}

func formatDBImportLimit(limit int64) string {
	switch {
	case limit%(1<<20) == 0:
		return fmt.Sprintf("%d MiB", limit>>20)
	case limit%(1<<10) == 0:
		return fmt.Sprintf("%d KiB", limit>>10)
	default:
		return fmt.Sprintf("%d bytes", limit)
	}
}
