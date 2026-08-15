package constant

// Status siklus pemeriksaan barang rusak:
//   pengecekan -> status awal saat dilaporkan, menunggu diperiksa admin.
//   diperbaiki -> barang selesai diperbaiki & kembali dipakai/masuk stok.
//   retur      -> dikembalikan ke supplier (dipakai juga oleh Laporan
//                 "Barang Retur", lihat internal/controller/laporan).
//   dibuang    -> barang tidak bisa diperbaiki, dikeluarkan permanen.
const (
	StatusBarangRusakPengecekan = "pengecekan"
	StatusBarangRusakDiperbaiki = "diperbaiki"
	StatusRetur                 = "retur"
	StatusBarangRusakDibuang    = "dibuang"
)

const (
	ErrBarangRusakTidakDitemukan = "data barang rusak tidak ditemukan"
	ErrBarangRusakStatusInvalid  = "status barang rusak tidak valid"
)
