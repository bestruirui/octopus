package op

import (
	"context"
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

func validateUserRole(role string) error {
	if role != model.UserRoleAdmin && role != model.UserRoleEditor && role != model.UserRoleViewer {
		return fmt.Errorf("invalid role: %s", role)
	}
	return nil
}

func validateManagedUserCredentials(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}
	if utf8.RuneCountInString(password) < minInitialAdminPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", minInitialAdminPasswordLength)
	}
	return nil
}

func refreshUserCacheFromDB() {
	var user model.User
	result := db.GetDB().Order("id ASC").First(&user)
	if result.Error == nil {
		userCache = user
		return
	}
	userCache = model.User{}
}

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
	if err := validateManagedUserCredentials(username, password); err != nil {
		if strings.Contains(err.Error(), "at least") {
			return fmt.Errorf("initial admin %s", err.Error())
		}
		return err
	}
	username = strings.TrimSpace(username)

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

func UserCreate(req model.UserCreateRequest, ctx context.Context) error {
	req.Username = strings.TrimSpace(req.Username)
	if err := validateManagedUserCredentials(req.Username, req.Password); err != nil {
		return err
	}
	if err := validateUserRole(req.Role); err != nil {
		return err
	}

	var count int64
	if err := db.GetDB().WithContext(ctx).Model(&model.User{}).
		Where("username = ?", req.Username).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to inspect existing users: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("username already exists")
	}

	user := model.User{
		Username: req.Username,
		Password: req.Password,
		Role:     req.Role,
	}
	if err := user.HashPassword(); err != nil {
		return err
	}
	if err := db.GetDB().WithContext(ctx).Create(&user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func UserChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := UserGetByID(userID, context.Background())
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if err := user.ComparePassword(oldPassword); err != nil {
		return fmt.Errorf("incorrect old password: %w", err)
	}

	user.Password = newPassword
	if err := user.HashPassword(); err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := db.GetDB().Model(&user).Update("password", user.Password).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if userCache.ID == user.ID {
		userCache.Password = user.Password
	}

	return nil
}

func UserChangeUsername(userID uint, newUsername string) error {
	newUsername = strings.TrimSpace(newUsername)
	if newUsername == "" {
		return fmt.Errorf("username is required")
	}

	user, err := UserGetByID(userID, context.Background())
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.Username == newUsername {
		return fmt.Errorf("new username is the same as the old username")
	}

	var count int64
	if err := db.GetDB().Model(&model.User{}).
		Where("username = ? AND id <> ?", newUsername, user.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("failed to inspect existing users: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("username already exists")
	}

	user.Username = newUsername
	if err := db.GetDB().Model(&user).Update("username", user.Username).Error; err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}
	if userCache.ID == user.ID {
		userCache.Username = user.Username
	}
	return nil
}

func UserVerify(username, password string) (model.User, error) {
	if !UserReady() {
		return model.User{}, ErrUserNotInitialized
	}
	user, err := UserGetByUsername(strings.TrimSpace(username), context.Background())
	if err != nil {
		return model.User{}, fmt.Errorf("incorrect username")
	}
	if err := user.ComparePassword(password); err != nil {
		return model.User{}, fmt.Errorf("incorrect password")
	}
	return user, nil
}

func UserGet() model.User {
	return userCache
}

func UserGetByID(id uint, ctx context.Context) (model.User, error) {
	var user model.User
	if err := db.GetDB().WithContext(ctx).First(&user, id).Error; err != nil {
		return model.User{}, err
	}
	return user, nil
}

func UserGetByUsername(username string, ctx context.Context) (model.User, error) {
	var user model.User
	if err := db.GetDB().WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return model.User{}, err
	}
	return user, nil
}

func UserList(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := db.GetDB().WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func UserUpdateRole(id uint, role string, ctx context.Context) error {
	if err := validateUserRole(role); err != nil {
		return err
	}
	res := db.GetDB().WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("role", role)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	// Refresh cache if the updated user is the current one
	if userCache.ID == id {
		userCache.Role = role
	}
	return nil
}

func UserDelete(id uint, currentUserID uint, ctx context.Context) error {
	if currentUserID != 0 && id == currentUserID {
		return fmt.Errorf("cannot delete the active user")
	}
	res := db.GetDB().WithContext(ctx).Delete(&model.User{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	if userCache.ID == id {
		refreshUserCacheFromDB()
	}
	return nil
}
