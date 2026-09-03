package asset_history

import (
	"time"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
)

type Repository interface {
	Log(h *model.AssetHistory) error

	ListByAsset(assetID uint, limit int) ([]model.AssetHistory, error)

	// ListByAssetRange: dipakai fitur tracking Bulanan — ambil SEMUA
	// kejadian riwayat aset ini dalam 1 rentang tanggal (biasanya 1 bulan
	// penuh), tanpa batas jumlah baris seperti ListByAsset, karena
	// rentangnya sudah dipersempit oleh caller.
	ListByAssetRange(assetID uint, dari, sampai time.Time) ([]model.AssetHistory, error)

	// ListRange: dipakai laporan Tracking Aset — ambil kejadian riwayat
	// LINTAS SEMUA aset (bukan 1 aset saja) dalam rentang tanggal opsional
	// (dari/sampai nil berarti tidak dibatasi ke arah itu), dibatasi limit
	// supaya tidak menyedot seluruh tabel kalau rentangnya besar.
	ListRange(dari, sampai *time.Time, limit int) ([]model.AssetHistory, error)
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Log(h *model.AssetHistory) error {
	return r.db.Create(h).Error
}

func (r *repository) ListByAsset(assetID uint, limit int) ([]model.AssetHistory, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []model.AssetHistory
	err := r.db.Where("asset_id = ?", assetID).
		Order("created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *repository) ListByAssetRange(assetID uint, dari, sampai time.Time) ([]model.AssetHistory, error) {
	var rows []model.AssetHistory
	err := r.db.Where("asset_id = ? AND created_at >= ? AND created_at < ?", assetID, dari, sampai).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) ListRange(dari, sampai *time.Time, limit int) ([]model.AssetHistory, error) {
	if limit <= 0 || limit > 20000 {
		limit = 20000
	}
	q := r.db.Model(&model.AssetHistory{})
	if dari != nil {
		q = q.Where("created_at >= ?", *dari)
	}
	if sampai != nil {
		q = q.Where("created_at <= ?", *sampai)
	}
	var rows []model.AssetHistory
	err := q.Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}
