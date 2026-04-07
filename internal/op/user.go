package op

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/log"
	"gorm.io/gorm"
)

var userCache model.User

const minInitialAdminPasswordLength = 12

func UserInit() error {
	if err := userBootstrapFromEnv(); err != nil {
		return err
	}

	result := db.GetDB().First(&userCache)
	if result.Error == nil {
		return nil
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	log.Warnf("initial admin account is not set up yet; waiting for bootstrap from web UI")
	userCache = model.User{}
	return nil
}

func userBootstrapFromEnv() error {
	username := strings.TrimSpace(os.Getenv("OCTOPUS_INITIAL_ADMIN_USERNAME"))
	password := os.Getenv("OCTOPUS_INITIAL_ADMIN_PASSWORD")

	if username == "" && password == "" {
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("both OCTOPUS_INITIAL_ADMIN_USERNAME and OCTOPUS_INITIAL_ADMIN_PASSWORD must be set together")
	}

	if err := deleteLegacyAdminUser(username); err != nil {
		return err
	}

	if UserReady() && userCache.Username == username {
		log.Infof("initial admin account already matches environment variable OCTOPUS_INITIAL_ADMIN_USERNAME=%s", username)
		return nil
	}

	if err := UserBootstrapCreate(username, password); err != nil {
		return fmt.Errorf("bootstrap admin from env: %w", err)
	}
	log.Infof("initial admin account created from environment variable OCTOPUS_INITIAL_ADMIN_USERNAME=%s", username)
	return nil
}

func deleteLegacyAdminUser(targetUsername string) error {
	if targetUsername == "admin" {
		return nil
	}

	result := db.GetDB().Where("username = ?", "admin").Delete(&model.User{})
	if result.Error != nil {
		return fmt.Errorf("delete legacy admin user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil
	}

	if userCache.Username == "admin" {
		userCache = model.User{}
	}

	log.Warnf("deleted legacy admin user because OCTOPUS_INITIAL_ADMIN_USERNAME=%s is configured", targetUsername)
	return nil
}

func UserBootstrapCreate(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if utf8.RuneCountInString(password) < minInitialAdminPasswordLength {
		return fmt.Errorf("initial admin password must be at least %d characters long", minInitialAdminPasswordLength)
	}

	var count int64
	if err := db.GetDB().Model(&model.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to inspect user state: %w", err)
	}
	if count > 0 || UserReady() {
		return fmt.Errorf("initial admin account is already set up")
	}

	user := model.User{
		Username: username,
		Password: password,
	}
	if err := user.HashPassword(); err != nil {
		return err
	}
	if err := db.GetDB().Create(&user).Error; err != nil {
		return err
	}
	userCache = user
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
