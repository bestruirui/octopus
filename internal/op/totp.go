package op

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

// TOTP 相关常量
const (
	totpTimeStep     = 30 // 时间步长（秒）
	totpCodeLength   = 6  // 验证码长度
	totpSecretLength = 20 // 密钥长度（字节）
	totpSkew         = 1  // 允许的时间偏移（步数）
)

// GenerateTOTPSecret 生成 TOTP 密钥（base32 编码）
func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, totpSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	// base32 编码，去掉填充符
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return encoded, nil
}

// GenerateTOTPUri 生成 otpauth:// URI，用于生成二维码
func GenerateTOTPUri(secret, username, appName string) string {
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", appName)
	params.Set("algorithm", "SHA1")
	params.Set("digits", fmt.Sprintf("%d", totpCodeLength))
	params.Set("period", fmt.Sprintf("%d", totpTimeStep))

	return fmt.Sprintf("otpauth://totp/%s:%s?%s",
		url.PathEscape(appName),
		url.PathEscape(username),
		params.Encode(),
	)
}

// totpCode 生成指定时间戳的 TOTP 验证码
func totpCode(secret []byte, timestamp int64) string {
	// 计算时间计数器
	counter := timestamp / totpTimeStep

	// 将计数器转为 8 字节大端序
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))

	// HMAC-SHA1
	mac := hmac.New(sha1.New, secret)
	mac.Write(counterBytes)
	hash := mac.Sum(nil)

	// 动态截断（RFC 4226）
	offset := hash[len(hash)-1] & 0x0f
	truncated := int32((((int32(hash[offset]) & 0x7f) << 24) |
		((int32(hash[offset+1]) & 0xff) << 16) |
		((int32(hash[offset+2]) & 0xff) << 8) |
		(int32(hash[offset+3]) & 0xff))) % int32(math.Pow10(totpCodeLength))

	return fmt.Sprintf("%0*d", totpCodeLength, truncated)
}

// ValidateTOTP 验证 TOTP 验证码（允许 ±1 步的时钟偏差）
func ValidateTOTP(secret string, code string) bool {
	// 移除可能的空格和短横线
	code = strings.TrimSpace(code)
	if len(code) != totpCodeLength {
		return false
	}

	// 解码 base32 密钥
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		// 尝试带填充的解码
		paddedSecret := secret
		if padLen := len(secret) % 8; padLen > 0 {
			paddedSecret += strings.Repeat("=", 8-padLen)
		}
		secretBytes, err = base32.StdEncoding.DecodeString(strings.ToUpper(paddedSecret))
		if err != nil {
			return false
		}
	}

	now := time.Now().Unix()

	// 检查当前时间及前后 totpSkew 个时间步
	for i := -totpSkew; i <= totpSkew; i++ {
		t := now + int64(i*totpTimeStep)
		expectedCode := totpCode(secretBytes, t)
		if expectedCode == code {
			return true
		}
	}

	return false
}

// TOTPEnable 启用 TOTP
func TOTPEnable() error {
	user := UserGet()
	user.TOTPEnabled = true
	if err := db.GetDB().Model(&model.User{}).Where("id = ?", user.ID).Update("totp_enabled", true).Error; err != nil {
		return fmt.Errorf("failed to enable TOTP: %w", err)
	}
	userCache.TOTPEnabled = true
	return nil
}

// TOTPDisable 禁用 TOTP
func TOTPDisable() error {
	user := UserGet()
	user.TOTPEnabled = false
	user.TOTPSecret = ""
	if err := db.GetDB().Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"totp_enabled": false,
		"totp_secret":  "",
	}).Error; err != nil {
		return fmt.Errorf("failed to disable TOTP: %w", err)
	}
	userCache.TOTPEnabled = false
	userCache.TOTPSecret = ""
	return nil
}

// TOTPSetSecret 设置 TOTP 密钥
func TOTPSetSecret(secret string) error {
	user := UserGet()
	user.TOTPSecret = secret
	if err := db.GetDB().Model(&model.User{}).Where("id = ?", user.ID).Update("totp_secret", secret).Error; err != nil {
		return fmt.Errorf("failed to set TOTP secret: %w", err)
	}
	userCache.TOTPSecret = secret
	return nil
}

// TOTPIsEnabled 检查 TOTP 是否已启用
func TOTPIsEnabled() bool {
	user := UserGet()
	return user.TOTPEnabled && user.TOTPSecret != ""
}

// TOTPGetSecret 获取 TOTP 密钥
func TOTPGetSecret() string {
	user := UserGet()
	return user.TOTPSecret
}
