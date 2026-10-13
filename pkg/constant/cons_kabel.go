package constant

// Kabel: stok kabel dihitung dalam METER (barang berstatuan meter). Satu
// catatan = satu gulungan fisik; setiap pemakaian dicatat sendiri dan
// mengurangi stok gudang dalam transaksi yang sama. Sisa gulungan tidak
// disimpan, selalu dihitung dari panjang awal dikurangi pemakaian aktif.
const (
	StatusKabelAktif = "aktif"
	StatusKabelHabis = "habis"
)

const (
	ErrKabelTidakDitemukan    = "gulungan kabel tidak ditemukan"
	ErrKabelBukanMeter        = "barang ini bukan kabel: satuannya harus Meter (m)"
	ErrKabelMelebihiStok      = "total sisa gulungan di gudang ini tidak boleh melebihi stok gudang"
	ErrKabelSisaKurang        = "pemakaian melebihi sisa gulungan"
	ErrKabelStokKurang        = "stok kabel di gudang tidak cukup untuk pemakaian ini"
	ErrKabelPemakaianTidakAda = "catatan pemakaian tidak ditemukan"
	ErrKabelSudahDibatalkan   = "pemakaian ini sudah dibatalkan"
	ErrKabelMasihAdaPemakaian = "gulungan sudah punya pemakaian, batalkan dulu pemakaiannya sebelum menghapus"
)
