package inspeksi_stok

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
)

type Repository interface {
	CreateInspection(*model.InspeksiStok) error
	GetBalance(barangID, gudangID uint) (*model.BalanceStok, error)
	SaveBalance(*model.BalanceStok) error
}
