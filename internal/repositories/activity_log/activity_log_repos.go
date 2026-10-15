package activity_log

import (
	"log"
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

// RetentionDays: riwayat yang lebih tua dari ini dihapus otomatis.
const RetentionDays = 7

type Repository interface {
	Create(l *model.ActivityLog) error
	DeleteOlderThan(t time.Time) (int64, error)
	List(f Filter) ([]model.ActivityLog, int64, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) Repository { return &repository{db: db} }

// Create menyimpan satu baris log. Kalau UserID terisi tapi username belum,
// username & nama lengkap diambil dari tabel users supaya riwayat tetap
// terbaca walau akunnya nanti diganti namanya.
func (r *repository) Create(l *model.ActivityLog) error {
	r.fillUser(l)
	return r.db.Create(l).Error
}

// fillUser melengkapi nama & role user. Untuk percobaan login gagal (user belum
// teridentifikasi) akun dicari lewat username yang diketik; bila akunnya ada,
// role-nya ikut ditampilkan.
func (r *repository) fillUser(l *model.ActivityLog) {
	var u model.User
	switch {
	case l.UserID != nil && l.Username == "":
		if err := r.db.Select("id", "username", "full_name", "role_id").First(&u, *l.UserID).Error; err != nil {
			return
		}
	case l.UserID == nil && l.Username != "":
		if err := r.db.Select("id", "username", "full_name", "role_id").Where("username = ?", l.Username).First(&u).Error; err != nil {
			return
		}
		id := u.ID
		l.UserID = &id
	default:
		if l.UserID != nil && l.RoleName == "" {
			if err := r.db.Select("id", "username", "full_name", "role_id").First(&u, *l.UserID).Error; err != nil {
				return
			}
		} else {
			return
		}
	}
	l.Username = u.Username
	l.FullName = u.FullName
	if l.RoleName == "" {
		var role model.Role
		if err := r.db.Select("id", "name").First(&role, u.RoleID).Error; err == nil {
			l.RoleName = role.Name
		}
	}
}

func (r *repository) DeleteOlderThan(t time.Time) (int64, error) {
	res := r.db.Where("created_at < ?", t).Delete(&model.ActivityLog{})
	return res.RowsAffected, res.Error
}

// StartPurger menghapus riwayat yang lebih tua dari RetentionDays: sekali saat
// start, lalu setiap `every`. Berjalan di goroutine sampai proses berhenti.
func StartPurger(repo Repository, every time.Duration) {
	purge := func() {
		cutoff := time.Now().AddDate(0, 0, -RetentionDays)
		if n, err := repo.DeleteOlderThan(cutoff); err != nil {
			log.Printf("activity-log: gagal menghapus riwayat lama: %v", err)
		} else if n > 0 {
			log.Printf("activity-log: %d riwayat lebih dari %d hari dihapus", n, RetentionDays)
		}
	}
	go func() {
		purge()
		t := time.NewTicker(every)
		defer t.Stop()
		for range t.C {
			purge()
		}
	}()
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
