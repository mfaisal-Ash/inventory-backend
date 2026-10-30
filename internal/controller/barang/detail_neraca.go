package barang

// NeracaBarang ...
type NeracaBarang struct {
	PerGudang     []NeracaGudangRow `json:"per_gudang"`
	StokAwal      int               `json:"stok_awal"`
	Masuk         int               `json:"masuk"`
	Keluar        int               `json:"keluar"`
	Penyesuaian   int               `json:"penyesuaian"`
	StokAkhir     int               `json:"stok_akhir"`
	StokMaster    int               `json:"stok_master"`
	SelisihMaster int               `json:"selisih_master"`
}

// NeracaGudangRow contains the inventory balance for one warehouse.
type NeracaGudangRow struct {
	GudangID    uint   `json:"gudang_id"`
	NamaGudang  string `json:"nama_gudang"`
	StokAwal    int    `json:"stok_awal"`
	Masuk       int    `json:"masuk"`
	Keluar      int    `json:"keluar"`
	Penyesuaian int    `json:"penyesuaian"`
	StokAkhir   int    `json:"stok_akhir"`
}
