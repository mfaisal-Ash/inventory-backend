package inspeksi_stok

import (
	"errors"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"gorm.io/gorm"
)

type repository struct { db *gorm.DB }

func New(db *gorm.DB) Repository { return &repository{db: db} }

func (r *repository) CreateInspection(row *model.InspeksiStok) error {
	if row.DiperiksaQty < 0 || row.NormalQty < 0 || row.RusakQty < 0 || row.BelumDiperiksaQty < 0 {
		return errors.New("kuantitas inspeksi tidak boleh negatif")
	}
	if row.NormalQty+row.RusakQty+row.BelumDiperiksaQty != row.DiperiksaQty {
		return errors.New("normal, rusak, dan belum diperiksa harus sama dengan jumlah diperiksa")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil { return err }
		var balance model.BalanceStok
		err := tx.Where("barang_id = ? AND gudang_id = ?", row.BarangID, row.GudangID).First(&balance).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			balance = model.BalanceStok{BarangID: row.BarangID, GudangID: row.GudangID}
		} else if err != nil { return err }
		balance.NormalQty = row.NormalQty
		balance.RusakQty = row.RusakQty
		balance.BelumDiperiksaQty = row.BelumDiperiksaQty
		return tx.Save(&balance).Error
	})
}

func (r *repository) GetBalance(barangID, gudangID uint) (*model.BalanceStok, error) {
	var balance model.BalanceStok
	if err := r.db.Where("barang_id = ? AND gudang_id = ?", barangID, gudangID).First(&balance).Error; err != nil { return nil, err }
	return &balance, nil
}

func (r *repository) SaveBalance(balance *model.BalanceStok) error { return r.db.Save(balance).Error }
