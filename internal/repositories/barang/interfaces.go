package barang

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	KategoriID  uint
	SatuanID    uint
	StokMenipis bool
	OnlyActive  bool

	// Merek/Tipe: filter tambahan supaya pengguna bisa mempersempit daftar
	// barang berdasarkan merek/tipe (dipakai bareng search nama/SKU yang
	// sudah ada) — exact match, bukan ILIKE, karena dropdown filter di
	// frontend diisi dari nilai merek/tipe yang benar-benar ada.
	Merek string
	Tipe  string

	ApprovalStatuses []string

	OrSubmittedBy uint
	// OrDelegatedTo: kalau diisi, item yang didelegasikan_ke user ini juga
	// ikut muncul di daftar walau approval_status-nya belum masuk
	// ApprovalStatuses (mis. masih "menunggu") — supaya admin yang didelegasikan
	// benar-benar bisa MELIHAT & memproses item yang ditugaskan kepadanya.
	OrDelegatedTo uint

	// OnlyDelegatedTo: dipakai tab/filter "Didelegasikan ke Saya" — kalau
	// diisi, HANYA item yang didelegasikan ke user ini yang ditampilkan
	// (menggantikan pelebaran approval-status di atas).
	OnlyDelegatedTo uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.Barang, int64, error)
	FindByID(id uint) (*model.Barang, error)
	FindByKode(kode string) (*model.Barang, error)
	Create(b *model.Barang) error
	Update(b *model.Barang) error
	Delete(id uint) error

	AdjustStok(id uint, delta int) (*model.Barang, error)

	SetStokGudangAwal(barangID, gudangID uint, stok int) error

	// UpdateWithStokKoreksi: simpan perubahan field non-stok SEKALIGUS
	// mengoreksi stok — delta diterapkan ke BarangStokGudang milik
	// stokGudangID, lalu Barang.Stok di-derive ulang dari
	// SUM(BarangStokGudang) supaya total tidak pernah menyimpang dari
	// rincian per-gudangnya (real-time-safe, lihat SyncBarangStokTotalTx).
	UpdateWithStokKoreksi(b *model.Barang, stokGudangID uint, delta int) error

	NextSKUNumber(prefix string) (int, error)

	CountAll() (int64, error)
	CountStokMenipis() (int64, error)
	SumNilaiInventaris() (int64, error)
}
