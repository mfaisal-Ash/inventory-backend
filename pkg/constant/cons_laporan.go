package constant

const (
	LaporanStokBarang   = "Stok Barang"
	LaporanBarangMasuk  = "Barng Masuk"
	LaporanBarangKeluar = "Barang Keluar"
	LaporanPO           = "Purchase Order"
	LaporanStokOpname   = "Stock Opname"

	LaporanBarangRetur     = "Barang Retur"
	LaporanBarangRusak     = "Barang Rusak"
	LaporanFifoFefo        = "FIFO FEFO"
	LaporanPengajuanBarang = "Pengajuan Barang"
	LaporanTrackingAset    = "Tracking Aset"
)

const (
	FormatExcel = "Excel"
	FormatPDF   = "PDF"
	FormatWord  = "Docs"
)

const (
	ErrLaporanTipeTidakDidukung   = "Jenis Laporan tidak di dukung."
	ErrLaporanFormatTidakDidukung = "Format ekspor laporan tidak di dukung (Gunakan excel, pdf, atau docs)."
)
