package accesspolicy

import "github.com/mfaisal-Ash/inventory-backend/internal/model"

type Repository interface {
	// Get mengembalikan (nil, nil) kalau kebijakan belum pernah disimpan.
	Get() (*model.AccessPolicy, error)
	// SaveAccessPolicy menyimpan daftar jaringan gudang (dipakai karyawan, dan admin
	// bila adminGudangOnly = true).
	SaveAccessPolicy(networks string, adminGudangOnly bool, updatedBy uint) (*model.AccessPolicy, error)
}
