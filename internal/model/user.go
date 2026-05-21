package model

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          uint   `gorm:"primaryKey"`
	Username    string `gorm:"unique"`
	Password    string `gorm:"not null"`
	TOTPEnabled bool   `gorm:"default:false"`
	TOTPSecret  string `gorm:"default:''"`
}

type UserLogin struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Expire   int    `json:"expire"`
	TOTPCode string `json:"totp_code"`
}

type UserTOTP struct {
	Code string `json:"code"`
}

type UserChangePassword struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UserChangeUsername struct {
	NewUsername string `json:"new_username"`
}

type UserLoginResponse struct {
	Token    string `json:"token"`
	ExpireAt string `json:"expire_at"`
}

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
