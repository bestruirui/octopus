package model

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	UserRoleAdmin  = "admin"
	UserRoleEditor = "editor"
	UserRoleViewer = "viewer"
)

type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"unique" json:"username"`
	Password string `gorm:"not null" json:"-"`
	Role     string `gorm:"default:'admin'" json:"role"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Expire   int    `json:"expire"`
}

type UserChangePassword struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UserChangeUsername struct {
	NewUsername string `json:"new_username"`
}

type UserBootstrapCreate struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UserLoginResponse struct {
	Token    string `json:"token"`
	ExpireAt string `json:"expire_at"`
}

// TableName explicitly returns "-" to prevent GORM from treating these DTOs as database tables.
func (UserLogin) TableName() string            { return "-" }
func (UserChangePassword) TableName() string   { return "-" }
func (UserChangeUsername) TableName() string   { return "-" }
func (UserBootstrapCreate) TableName() string  { return "-" }
func (UserCreateRequest) TableName() string    { return "-" }
func (UserLoginResponse) TableName() string    { return "-" }

func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
}
