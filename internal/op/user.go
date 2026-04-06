package op

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	"gorm.io/gorm"
)

var userCache model.User

const minInitialAdminPasswordLength = 12

func UserInit() error {
	result := db.GetDB().First(&userCache)
	if result.Error == nil {
		return nil
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	if allowInitBypass() {
		log.Warnf("initial admin credentials are not configured; allowing uninitialized startup because %s_ALLOW_UNINITIALIZED_STARTUP is enabled", strings.ToUpper(conf.APP_NAME))
		userCache = model.User{}
		return nil
	}

	username := strings.TrimSpace(os.Getenv(strings.ToUpper(conf.APP_NAME) + "_INITIAL_ADMIN_USERNAME"))
	password := os.Getenv(strings.ToUpper(conf.APP_NAME) + "_INITIAL_ADMIN_PASSWORD")
	if username == "" || password == "" {
		return fmt.Errorf("initial admin credentials are required; set %s_INITIAL_ADMIN_USERNAME and %s_INITIAL_ADMIN_PASSWORD", strings.ToUpper(conf.APP_NAME), strings.ToUpper(conf.APP_NAME))
	}
	if utf8.RuneCountInString(password) < minInitialAdminPasswordLength {
		return fmt.Errorf("initial admin password must be at least %d characters long", minInitialAdminPasswordLength)
	}

	userCache.Username = username
	userCache.Password = password
	if err := userCache.HashPassword(); err != nil {
		return err
	}
	if err := db.GetDB().Create(&userCache).Error; err != nil {
		return err
	}
	log.Infof("initial admin user created: %s", userCache.Username)
	return nil
}

func UserChangePassword(oldPassword, newPassword string) error {
	if err := userCache.ComparePassword(oldPassword); err != nil {
		return fmt.Errorf("incorrect old password: %w", err)
	}

	userCache.Password = newPassword
	if err := userCache.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := db.GetDB().Model(&userCache).Update("password", userCache.Password).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func UserChangeUsername(newUsername string) error {
	if userCache.Username == newUsername {
		return fmt.Errorf("new username is the same as the old username")
	}
	userCache.Username = newUsername
	if err := db.GetDB().Model(&userCache).Update("username", userCache.Username).Error; err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	return nil
}

func UserVerify(username, password string) error {
	if !UserReady() {
		return ErrUserNotInitialized
	}
	if username != userCache.Username {
		return fmt.Errorf("incorrect username")
	}
	if err := userCache.ComparePassword(password); err != nil {
		return fmt.Errorf("incorrect password")
	}
	return nil
}

func UserGet() model.User {
	return userCache
}
