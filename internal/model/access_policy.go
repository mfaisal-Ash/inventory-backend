package model

import "time"

// AccessPolicy menyimpan kebijakan akses yang bisa diubah super admin lewat
// aplikasi (singleton, ID selalu 1). Saat ini isinya daftar jaringan gudang
// yang boleh dipakai akun karyawan. Kalau barisnya belum ada, backend memakai
// nilai KARYAWAN_ALLOWED_NETWORKS dari environment.
type AccessPolicy struct {
	ID uint `json:"id" gorm:"primaryKey"`
	// KaryawanNetworks: daftar CIDR/IP dipisah koma. Kosong = tanpa pembatasan.
	KaryawanNetworks string `json:"karyawan_networks" gorm:"type:text"`
	// AdminGudangOnly: true = akun admin ikut dibatasi ke daftar jaringan di atas.
	// Super admin tidak pernah dibatasi.
	AdminGudangOnly bool      `json:"admin_gudang_only" gorm:"not null;default:false"`
	UpdatedBy       *uint     `json:"updated_by"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (AccessPolicy) TableName() string { return "access_policy" }
