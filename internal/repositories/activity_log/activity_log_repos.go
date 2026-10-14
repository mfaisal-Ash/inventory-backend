package activity_log

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
)

type Filter struct {
	UserID  uint
	Action  string
	Module  string
	Search  string
	Success *bool
	Dari    *time.Time // awal hari (inklusif)
	Sampai  *time.Time // awal hari terakhir (inklusif, otomatis +1 hari)
	Page    int
	Limit   int
}

type Repository interface {
	Create(l *model.ActivityLog) error
	List(f Filter) ([]model.ActivityLog, int64, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) Repository { return &repository{db: db} }

// Create menyimpan satu baris log. Kalau UserID terisi tapi username belum,
// username & nama lengkap diambil dari tabel users supaya riwayat tetap
// terbaca walau akunnya nanti diganti namanya.
func (r *repository) Create(l *model.ActivityLog) error {
	if l.UserID != nil && l.Username == "" {
		var u model.User
		if err := r.db.Select("id", "username", "full_name").First(&u, *l.UserID).Error; err == nil {
			l.Username = u.Username
			l.FullName = u.FullName
		}
	}
	return r.db.Create(l).Error
}

func (r *repository) List(f Filter) ([]model.ActivityLog, int64, error) {
	q := r.db.Model(&model.ActivityLog{})

	if f.UserID != 0 {
		q = q.Where("user_id = ?", f.UserID)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.Module != "" {
		q = q.Where("module = ?", f.Module)
	}
	if f.Success != nil {
		q = q.Where("success = ?", *f.Success)
	}
	if f.Dari != nil {
		q = q.Where("created_at >= ?", *f.Dari)
	}
	if f.Sampai != nil {
		q = q.Where("created_at < ?", f.Sampai.AddDate(0, 0, 1))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where(
			"description ILIKE ? OR username ILIKE ? OR full_name ILIKE ? OR module_label ILIKE ? OR entity_id ILIKE ?",
			like, like, like, like, like,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, limit := f.Page, f.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	var rows []model.ActivityLog
	err := q.Order("created_at DESC, id DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&rows).Error
	return rows, total, err
}
