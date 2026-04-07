package op

import (
	"errors"
	"fmt"

	"github.com/lingyuins/octopus/internal/db"
	"gorm.io/gorm"
)

var ErrUserNotInitialized = errors.New("user not initialized")

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
