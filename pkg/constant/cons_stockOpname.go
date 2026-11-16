package constant

const (
	StatusSODraft               = "draft"
	StatusSOSedangDihitung      = "sedang_dihitung"
	StatusSOMenungguPemeriksaan = "menunggu_pemeriksaan"
	StatusSODitolak             = "ditolak"
	StatusSOSelesai             = "selesai"
	StatusSODibatalkan          = "dibatalkan"
)

const (
	ErrSOTidakDitemukan = "dokumen stock opname tidak ditemukan"
	ErrSOBukanDraft     = "dokumen stock opname masih berupa draft"
)
