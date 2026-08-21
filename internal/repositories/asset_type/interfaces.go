package assettype

import "github.com/mfaisal-Ash/inventory-backend/internal/model"

type Repository interface {
	List() ([]model.AssetType, error)
	FindByID(id uint) (*model.AssetType, error)
	FindByKode(kode string) (*model.AssetType, error)
	Create(t *model.AssetType) error
	Update(t *model.AssetType) error
	Delete(id uint) error

	// CountAssetsUsing menghitung berapa baris di tabel assets yang masih
	// memakai kode jenis aset ini — dipakai untuk mencegah penghapusan
	// jenis aset yang masih dipakai.
	CountAssetsUsing(kode string) (int64, error)
}
