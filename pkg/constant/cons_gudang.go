package constant

const (
	QueryNamaILike    = "nama ILIKE ?"
	QueryGudangIDEq   = "gudang_id"
	QueryKodeRakILIKE = "koda_rak ILIKE ?"
	QueryStatusEq     = "status = ?"
)

// Tipe gudang: "pusat" (1 per sistem) atau "cabang".
const (
	TipeGudangPusat  = "pusat"
	TipeGudangCabang = "cabang"
)
