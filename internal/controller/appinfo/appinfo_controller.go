package appinfo

import (
	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// CurrentVersion diisi otomatis saat build dari tag GitHub lewat
// -ldflags "-X <modul>/internal/controller/appinfo.CurrentVersion=vX.Y.Z"
// (lihat workflows/backend-release.yml). Nilai di bawah hanya cadangan untuk
// `go run` / build lokal tanpa ldflags. Harus `var`, bukan `const`, karena
// linker Go hanya bisa menimpa variabel string.
var CurrentVersion = "v1.3.7"

type VersionChanges struct {
	New []string `json:"new,omitempty"`
	Fix []string `json:"fix,omitempty"`
}

type VersionEntry struct {
	Version string         `json:"version"`
	Date    string         `json:"date"`
	Changes VersionChanges `json:"changes"`
}

var changelogData = []VersionEntry{
	{
		Version: "v1.3.4",
		Date:    "2 November 2026",
		Changes: VersionChanges{
			New: []string{
				"Tanggal catat barang bisa dipilih (data susulan bertanggal mundur) dan kode barang berformat KATEGORI-PRODUK-DDMMYYYY-NN",
				"Neraca & Analisis Data menghitung stok awal dan keluar sebelum tercatat sesuai tanggal catat, plus pilihan periode cepat",
				"Detail Gudang dengan pengecekan fisik (checklist sesuai/beda dibanding data sistem)",
				"Tracking Aset mode Sederhana berupa tabel dengan detail rinci",
				"Tab Umum di Aset Gudang untuk aset perusahaan non-jaringan (televisi, printer, dll)",
				"Pengecekan Barang dipindah ke menu Manajemen dengan tab Pengecekan Barang, Barang Retur, dan Barang Rusak",
			},
			Fix: []string{
				"Isian Sisa di Tambah Barang tidak lagi diabaikan; selisih Kuantitas dan Sisa tersimpan sebagai keluar sebelum tercatat",
				"Versi aplikasi disamakan dengan tag rilis terbaru",
			},
		},
	},
	{
		Version: "v1.3.0",
		Date:    "11 Agustus 2026",
		Changes: VersionChanges{
			New: []string{
				"Titik koordinat gudang di peta Pickup & Dropoff",
				"Kerjasama Kurir di data Supplier (menggantikan Email) — Total Order & Rating dihitung otomatis dari data pengiriman",
				"Resi pengiriman didesain ulang: barcode besar, SKU berawalan WRSD-, Order ID, berat barang, kontak pengirim lengkap",
				"Cetak (Print) tersedia di seluruh tabel & laporan untuk Super Admin",
				"Widget Traffic Pengiriman & Rekap Data di dashboard Super Admin",
			},
			Fix: []string{
				"Matrix perizinan sekarang benar-benar mengendalikan tombol Tambah/Ubah/Cetak di setiap modul (sebelumnya sebagian tombol staff-only walau izin sudah diaktifkan)",
				"Kapasitas Terpakai & Rak Penuh di Manajemen Gudang dihitung otomatis dari data rak",
				"Peta pelacakan pengiriman tidak lagi error saat lokasi belum tersedia",
			},
		},
	},
	{
		Version: "v1.2.1",
		Date:    "10 Agustus 2026",
		Changes: VersionChanges{
			Fix: []string{
				"Perbaikan format tanggal di Barang Masuk/Keluar, Pengiriman, Purchase Order, Stock Opname",
				"Bypass izin penuh untuk Super Admin di seluruh modul",
			},
		},
	},
	{
		Version: "v1.2.0",
		Date:    "10 Agustus 2026",
		Changes: VersionChanges{
			New: []string{"tambah Menu", "integrasi ke backend"},
			Fix: []string{"Perbaikan menu dashboard"},
		},
	},
	{
		Version: "v1.1.0",
		Date:    "1 Agustus 2026",
		Changes: VersionChanges{
			New: []string{"Auth register,login", "lupa password", "Dashboard"},
		},
	},
}

type VersionResponse struct {
	Version     string `json:"version"`
	AppName     string `json:"app_name"`
	Description string `json:"description"`
	Developer   string `json:"developer"`
}

func (h *Controller) Version(c *fiber.Ctx) error {
	return utils.OK(c, "versi aplikasi berhasil diambil", VersionResponse{
		Version:     CurrentVersion,
		AppName:     "WMS - RSD",
		Description: "WMS-RSD merupakan pelayanan gudang serta inventaris produk dalam perusahaan — mengelola stok, barang masuk/keluar, aset gudang (tiang/ODC/ONT/ODP/OLT/transportasi) beserta tracking-nya, hingga laporan operasional dalam satu sistem.",
		Developer:   "Tim Internal RSD",
	})
}

func (h *Controller) Changelog(c *fiber.Ctx) error {
	return utils.OK(c, "riwayat pembaruan berhasil diambil", changelogData)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/app")
	g.Get("/version", h.Version)
	g.Get("/changelog", h.Changelog)

	g.Get("/check-update", middleware.JWTAuth(h.jwtSvc), h.CheckUpdate)
	g.Get("/update-status", middleware.JWTAuth(h.jwtSvc), h.UpdateStatus)
	g.Post("/update", middleware.JWTAuth(h.jwtSvc), middleware.RequireRole(constant.RoleSuperAdmin), h.TriggerUpdate)
}
