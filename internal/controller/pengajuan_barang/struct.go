package pengajuan_barang

import (
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	gudangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/gudang"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notifikasi"
	pengajuanRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	pengajuanTemplateRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_template"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	repo         pengajuanRepo.Repository
	barangRepo   barangRepo.Repository
	gudangRepo   gudangRepo.Repository
	usersRepo    usersRepo.Repository
	roleRepo     role.Repository
	jwtSvc       *utils.JWTService
	notifRepo    notificationRepo.Repository
	templateRepo pengajuanTemplateRepo.Repository
}

func New(
	repo pengajuanRepo.Repository,
	barangRepo barangRepo.Repository,
	gudangRepo gudangRepo.Repository,
	usersRepo usersRepo.Repository,
	roleRepo role.Repository,
	jwtSvc *utils.JWTService,
	notifRepo notificationRepo.Repository,
	templateRepo pengajuanTemplateRepo.Repository,
) *Controller {
	return &Controller{
		repo:         repo,
		barangRepo:   barangRepo,
		gudangRepo:   gudangRepo,
		usersRepo:    usersRepo,
		roleRepo:     roleRepo,
		jwtSvc:       jwtSvc,
		notifRepo:    notifRepo,
		templateRepo: templateRepo,
	}
}

type ItemRequest struct {
	BarangID uint   `json:"barang_id" validate:"omitempty"`
	Kode     string `json:"kode" validate:"max=100"`
	Qty      int    `json:"qty" validate:"required,min=1"`
	// Field manual hanya dipakai untuk jenis "pengajuan" ketika barang belum
	// ada di master Kelola Barang.
	Nama          string   `json:"nama" validate:"max=150"`
	Merek         string   `json:"merek" validate:"max=100"`
	Tipe          string   `json:"tipe" validate:"max=100"`
	IsSerialized  bool     `json:"is_serialized"`
	SerialNumbers []string `json:"serial_numbers" validate:"max=500"`
	// SatuanID: opsional, merujuk ke master satuan (mis. roll, meter).
	SatuanID *uint `json:"satuan_id" validate:"omitempty"`
}

type PengajuanRequest struct {
	// Jenis: "masuk" | "keluar" | "rusak" | "pengajuan" | "template" — opsional, default
	// "keluar" agar kompatibel dengan klien lama yang belum mengirim field
	// ini. "template" menggantikan jenis "umum" (pengajuan ke atasan) yang
	// dulu ada di sini — sekarang berbasis formulir yang diunggah admin,
	// lihat TemplateID.
	Jenis     string `json:"jenis" validate:"omitempty,oneof=masuk keluar rusak pengajuan template"`
	GudangID  uint   `json:"gudang_id" validate:"required"`
	Tanggal   string `json:"tanggal" validate:"required"`
	Keperluan string `json:"keperluan" validate:"required,max=255"`

	// Perihal: catatan/subjek singkat opsional untuk jenis apa pun.
	Perihal string `json:"perihal" validate:"omitempty,max=150"`

	// TemplateID: wajib diisi hanya untuk jenis "template" — merujuk ke
	// formulir yang dipilih dari daftar PengajuanTemplate yang aktif.
	// Divalidasi manual di controller karena wajib-tidaknya tergantung
	// Jenis, bukan lewat tag di sini.
	TemplateID *uint `json:"template_id" validate:"omitempty"`

	// Nama & jabatan orang yang akan tanda tangan di kolom "Bagian
	// Pencatatan/Gudang" pada dokumen cetak — opsional, boleh dikosongkan
	// lalu diisi tulis tangan di atas kertas.
	NamaPencatat    string `json:"nama_pencatat" validate:"max=150"`
	JabatanPencatat string `json:"jabatan_pencatat" validate:"max=100"`

	// Items: wajib diisi minimal satu untuk jenis "masuk"/"keluar"/"rusak",
	// tapi tidak dipakai sama sekali untuk jenis "template" (tidak terkait
	// barang) — karena itu tag validate di sini hanya memvalidasi isi tiap
	// elemen (dive) kalau ada, sedangkan wajib-tidaknya jumlah elemen
	// divalidasi manual di controller sesuai Jenis.
	Items []ItemRequest `json:"items" validate:"omitempty,dive"`

	// TTDPencatat: tanda tangan digital pengaju (pihak gudang) — data URL PNG
	// ("data:image/png;base64,...") dari kanvas tanda tangan. WAJIB untuk jenis
	// "masuk" (alur restock dua pihak), opsional untuk jenis lain. Setiap kali
	// pengajuan diedit, tanda tangan harus dikirim ulang karena isinya berubah.
	TTDPencatat string `json:"ttd_pencatat" validate:"omitempty,max=450000"`
}

type ProsesRequest struct {
	// Nama & jabatan orang yang memproses (menyetujui/menolak) — muncul di
	// kolom "Bagian General Affairs (GA)" pada dokumen cetak.
	NamaGA    string `json:"nama_ga" validate:"max=150"`
	JabatanGA string `json:"jabatan_ga" validate:"max=100"`
	Catatan   string `json:"catatan" validate:"max=255"`

	// TTD: tanda tangan digital penyetuju (pihak kedua), format sama dengan
	// PengajuanRequest.TTDPencatat. Wajib untuk jenis "masuk".
	TTD string `json:"ttd" validate:"omitempty,max=450000"`
}

type TolakRequest struct {
	NamaGA    string `json:"nama_ga" validate:"max=150"`
	JabatanGA string `json:"jabatan_ga" validate:"max=100"`
	Catatan   string `json:"catatan" validate:"required,min=3,max=255"`
}

type SummaryResponse struct {
	TotalDiajukan  int64 `json:"total_diajukan"`
	TotalDisetujui int64 `json:"total_disetujui"`
	TotalDitolak   int64 `json:"total_ditolak"`
}
