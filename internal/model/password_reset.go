package model

import "time"

const (
	PasswordResetPending  = "pending"
	PasswordResetApproved = "approved"
	PasswordResetRejected = "rejected"
)

type PasswordResetRequest struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"not null;index"`
	User       *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Status     string     `json:"status" gorm:"size:20;not null;index"`
	Reason     string     `json:"reason" gorm:"size:500"`
	ReviewedBy *uint      `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote string     `json:"review_note,omitempty" gorm:"size:500"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
