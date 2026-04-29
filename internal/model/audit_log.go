package model

type AuditLog struct {
	ID         int64  `json:"id" gorm:"primaryKey"`
	UserID     int    `json:"user_id" gorm:"index"`
	Username   string `json:"username" gorm:"index"`
	Action     string `json:"action" gorm:"index"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Target     string `json:"target"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}
