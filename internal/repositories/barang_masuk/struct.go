package barang_masuk

import (
	"gorm.io/gorm"

	poRepo "github.com/inventory-backend/internal/repositories/po"
)

type repository struct {
	db     *gorm.DB
	poRepo poRepo.Repository
}

func New(db *gorm.DB, poRepo poRepo.Repository) Repository {
	return &repository{db: db, poRepo: poRepo}
}
