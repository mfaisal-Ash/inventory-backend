package constant

const (
	StatusPengajuanDiajukan  = "diajukan"
	StatusPengajuanDisetujui = "disetujui"
	StatusPengajuanDitolak   = "ditolak"
)

// Jenis pengajuan barang — menentukan dokumen apa yang otomatis dibuat saat
// pengajuan disetujui: barang keluar (default, kompatibel dengan data lama),
// barang masuk (draft), barang rusak (satu baris per unit per item), atau
// template (pengajuan bebas berbasis formulir yang diunggah admin — lihat
// model.PengajuanTemplate — tidak terkait barang, tidak ada dokumen otomatis
// yang dibuat saat disetujui). "template" menggantikan jenis "umum"
// (pengajuan ke atasan) yang dulu ada di sini.
const (
	JenisPengajuanMasuk    = "masuk"
	JenisPengajuanKeluar   = "keluar"
	JenisPengajuanRusak    = "rusak"
	JenisPengajuanTemplate = "template"
	// JenisPengajuanBarang: pengajuan permintaan barang (mengajukan barang yang
	// dibutuhkan). Wajib punya daftar barang, tetapi saat disetujui TIDAK
	// membuat dokumen otomatis dan TIDAK mengubah stok — hanya status yang berubah.
	JenisPengajuanBarang = "pengajuan"
)

const (
	ErrPengajuanTidakDitemukan         = "pengajuan barang tidak ditemukan"
	ErrPengajuanBukanDiajukan          = "pengajuan ini sudah diproses (disetujui/ditolak) sehingga tidak bisa diubah, dihapus, disetujui, atau ditolak lagi"
	ErrPengajuanStokTidakCukup         = "stok barang tidak mencukupi untuk menyetujui pengajuan ini"
	ErrPengajuanJenisTidakValid        = "jenis pengajuan tidak valid (gunakan masuk, keluar, rusak, pengajuan, atau template)"
	ErrPengajuanTemplateWajib          = "template formulir wajib dipilih untuk jenis pengajuan ini"
	ErrPengajuanTemplateTidakDitemukan = "template formulir tidak ditemukan atau sudah tidak aktif"

	ErrPengajuanTTDPencatatWajib        = "tanda tangan digital pengaju (pihak gudang) wajib diisi untuk pengajuan barang masuk"
	ErrPengajuanTTDPenyetujuWajib       = "tanda tangan digital penyetuju wajib diisi untuk menyetujui pengajuan barang masuk"
	ErrPengajuanTTDTidakValid           = "tanda tangan digital tidak valid (kirim gambar PNG hasil coretan, maksimal 300 KB)"
	ErrPengajuanTTDPencatatHilang       = "pengajuan ini belum ditandatangani pihak pengaju, atau isinya sudah berubah setelah ditandatangani — minta pengaju menandatangani ulang"
	ErrPengajuanPenyetujuSamaDgnPengaju = "penyetuju harus orang yang berbeda dari pengaju (dua pihak harus menyetujui)"
	ErrPengajuanTTDTidakDitemukan       = "tanda tangan tidak ditemukan"
)

// PengajuanWajibTTD: jenis pengajuan yang wajib melewati alur dua pihak dengan
// tanda tangan digital (pengaju -> penyetuju) sebelum barang masuk dibuat.
// Jenis lain tetap boleh tanda tangan (opsional), tapi tidak diwajibkan supaya
// alur lama tidak berubah.
func PengajuanWajibTTD(jenis string) bool { return jenis == JenisPengajuanMasuk }
