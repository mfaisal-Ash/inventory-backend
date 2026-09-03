package task

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.AssignedTo != 0 {
		q = q.Where("assigned_to = ?", f.AssignedTo)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.Task, int64, error) {
	var list []model.Task
	var total int64

	q := applyFilter(r.db.Model(&model.Task{}), f)
	if p.Search != "" {
		q = q.Where("title ILIKE ? OR description ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).Preload("Assignee").Preload("Assigner").Order("due_date asc")).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.Task, error) {
	var t model.Task
	if err := r.db.Preload("Assignee").Preload("Assigner").First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) Create(t *model.Task) error {
	return r.db.Create(t).Error
}

// Update: Omit(clause.Associations) — cegah Save() menimpa balik
// assigned_to dengan ID dari relasi Assignee yang ter-preload di FindByID
// (bug sama seperti repositories/barang; lihat komentar di sana). Tanpa
// ini, memindahkan tugas ke orang lain lewat Ubah Tugas kelihatan berhasil
// tapi diam-diam balik ke assignee lama di database.
func (r *repository) Update(t *model.Task) error {
	return r.db.Omit(clause.Associations).Save(t).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&model.Task{}, id).Error
}

func (r *repository) CountByStatus(assignedTo uint, status string) (int64, error) {
	var count int64
	q := r.db.Model(&model.Task{}).Where("status = ?", status)
	if assignedTo != 0 {
		q = q.Where("assigned_to = ?", assignedTo)
	}
	err := q.Count(&count).Error
	return count, err
}

func (r *repository) CountOverdue(assignedTo uint) (int64, error) {
	var count int64
	q := r.db.Model(&model.Task{}).Where("status != 'selesai' AND due_date < ?", time.Now())
	if assignedTo != 0 {
		q = q.Where("assigned_to = ?", assignedTo)
	}
	err := q.Count(&count).Error
	return count, err
}
