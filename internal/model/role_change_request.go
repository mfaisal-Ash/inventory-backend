package model

import "time"

type RoleChangeRequest struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	UserID        uint       `json:"user_id" gorm:"not null;index"`
	CurrentRole   string     `json:"current_role" gorm:"size:30;not null"`
	RequestedRole string     `json:"requested_role" gorm:"size:30;not null"`
	Reason        string     `json:"reason" gorm:"size:1000;not null"`
	Status        string     `json:"status" gorm:"size:20;not null;default:pending;index"`
	DecisionNote  string     `json:"decision_note" gorm:"size:1000"`
	DecidedBy     *uint      `json:"decided_by,omitempty"`
	DecidedAt     *time.Time `json:"decided_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	User          *User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
}
