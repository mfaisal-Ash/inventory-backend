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
)

const (
	ErrPengajuanTidakDitemukan         = "pengajuan barang tidak ditemukan"
	ErrPengajuanBukanDiajukan          = "pengajuan ini sudah diproses (disetujui/ditolak) sehingga tidak bisa diubah, dihapus, disetujui, atau ditolak lagi"
	ErrPengajuanStokTidakCukup         = "stok barang tidak mencukupi untuk menyetujui pengajuan ini"
	ErrPengajuanJenisTidakValid        = "jenis pengajuan tidak valid (gunakan masuk, keluar, rusak, atau template)"
	ErrPengajuanTemplateWajib          = "template formulir wajib dipilih untuk jenis pengajuan ini"
	ErrPengajuanTemplateTidakDitemukan = "template formulir tidak ditemukan atau sudah tidak aktif"
)
