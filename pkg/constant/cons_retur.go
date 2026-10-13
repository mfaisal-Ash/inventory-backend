package constant

// Retur penggantian: modem/alat di klien rusak -> dicek teknisi -> kalau perlu
// diganti, unit baru berstatus Normal (BarangSerial tersedia) keluar ke klien
// dan unit lama WAJIB ditarik kembali ke gudang sebagai barang rusak.
const (
	StatusReturDilaporkan     = "dilaporkan"
	StatusReturPerluGanti     = "perlu_ganti"
	StatusReturTidakPerlu     = "tidak_perlu_ganti"
	StatusReturSelesai        = "selesai"
	HasilReturPerluGanti      = "perlu_ganti"
	HasilReturTidakPerluGanti = "tidak_perlu_ganti"
)

const (
	ErrReturTidakDitemukan       = "data retur tidak ditemukan"
	ErrReturStatusTidakSesuai    = "status retur tidak sesuai untuk proses ini"
	ErrReturPenggantiBukanNormal = "unit pengganti harus berstatus Normal (tersedia) dan berada di gudang retur ini"
	ErrReturSerialLamaDobel      = "unit ini sudah punya retur yang masih berjalan"
)
