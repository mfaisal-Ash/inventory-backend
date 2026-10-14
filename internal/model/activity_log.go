package model

import "time"

// ActivityLog mencatat apa pun yang dilakukan user di aplikasi: tambah/ubah/
// hapus data, login/logout, import, export, cetak, dan seterusnya. Baris
// log tidak pernah diubah setelah dibuat (append-only).
type ActivityLog struct {
	ID uint `json:"id" gorm:"primaryKey"`

	UserID   *uint  `json:"user_id" gorm:"index"`
	Username string `json:"username" gorm:"size:50;index"`
	FullName string `json:"full_name" gorm:"size:100"`
	RoleName string `json:"role_name" gorm:"size:50"`

	// Action: create, update, delete, approve, reject, complete, cancel,
	// protect, login, logout, import, export, print, other.
	Action      string `json:"action" gorm:"size:20;not null;index"`
	Module      string `json:"module" gorm:"size:60;index"`
	ModuleLabel string `json:"module_label" gorm:"size:100"`
	EntityID    string `json:"entity_id" gorm:"size:40"`
	Description string `json:"description" gorm:"size:500"`

	Method     string `json:"method" gorm:"size:10"`
	Path       string `json:"path" gorm:"size:255"`
	StatusCode int    `json:"status_code"`
	Success    bool   `json:"success" gorm:"index"`

	IPAddress string `json:"ip_address" gorm:"size:64"`
	UserAgent string `json:"user_agent" gorm:"size:255"`

	// Detail: JSON ringkas (body yang sudah disamarkan, pesan error, dll).
	Detail string `json:"detail" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"index"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
