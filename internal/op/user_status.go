package op

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/db"
	"gorm.io/gorm"
)

var ErrUserNotInitialized = errors.New("user not initialized")

func allowInitBypass() bool {
	value := strings.TrimSpace(os.Getenv(strings.ToUpper(conf.APP_NAME) + "_ALLOW_UNINITIALIZED_STARTUP"))
	if value == "" {
		return false
	}
	return strings.EqualFold(value, "1") || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

func UserReady() bool {
	return userCache.ID != 0
}

func UserBootstrapStatus() (bool, string, error) {
	if UserReady() {
		return true, "", nil
	}

	var count int64
	if err := db.GetDB().Model(&userCache).Count(&count).Error; err != nil {
		if errors.Is(err, gorm.ErrInvalidDB) {
			return false, "database not initialized", err
		}
		return false, "failed to inspect user initialization state", fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return true, "", nil
	}
	return false, "initial admin account is not set up yet", nil
}
