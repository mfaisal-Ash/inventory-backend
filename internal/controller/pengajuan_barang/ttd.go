package pengajuan_barang

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	barangRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const (
	maxTTDBytes  = 300 * 1024 // gambar tanda tangan maksimal 300 KB
	maxTTDPixels = 4_000_000  // batas lebar x tinggi, mencegah "decompression bomb"
	minTTDInkPx  = 30         // minimal piksel "tinta" supaya kanvas kosong ditolak
	pngDataURL   = "data:image/png;base64,"
	maxLowStock  = 500
)

// decodeTTD mengubah kiriman frontend (data URL "data:image/png;base64,..."
// atau base64 mentah) menjadi byte PNG, dengan validasi ketat: ukuran, isi
// benar-benar PNG, dimensi wajar, dan ada coretan (bukan kanvas kosong).
func decodeTTD(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, pngDataURL)
	if raw == "" {
		return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		if data, err = base64.RawStdEncoding.DecodeString(raw); err != nil {
			return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
		}
	}
	if len(data) == 0 || len(data) > maxTTDBytes || !utils.ImageMatchesType(data, "image/png") {
		return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxTTDPixels {
		return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
	}
	ink := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y && ink < minTTDInkPx; y++ {
		for x := b.Min.X; x < b.Max.X && ink < minTTDInkPx; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			// Piksel dihitung "tinta" kalau tidak transparan dan bukan putih.
			if a > 0x4000 && (r < 0xE000 || g < 0xE000 || bl < 0xE000) {
				ink++
			}
		}
	}
	if ink < minTTDInkPx {
		return nil, errors.New(constant.ErrPengajuanTTDTidakValid)
	}
	return data, nil
}

// buildTTD menyusun catatan tanda tangan. Nama default-nya nama lengkap akun
// kalau pengguna tidak mengetik nama sendiri.
func (h *Controller) buildTTD(userID uint, peran, nama, jabatan string, data []byte, ip string) *model.PengajuanTandaTangan {
	if strings.TrimSpace(nama) == "" {
		if u, err := h.usersRepo.FindByID(userID); err == nil && u != nil {
			nama = u.FullName
			if nama == "" {
				nama = u.Username
			}
		}
	}
	return &model.PengajuanTandaTangan{
		Peran: peran, UserID: userID, Nama: nama, Jabatan: jabatan,
		Data: data, IPAddress: ip, SignedAt: time.Now(),
	}
}

// ServeTTD: GET /pengajuan-barang/:id/ttd/:peran — gambar tanda tangan
// (peran: pencatat | penyetuju). Hanya untuk pengguna yang boleh melihat
// pengajuan; tidak pernah ikut di JSON daftar/detail supaya payload ringan.
func (h *Controller) ServeTTD(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	peran := c.Params("peran")
	if peran != model.PeranTTDPencatat && peran != model.PeranTTDPenyetuju {
		return utils.Fail(c, fiber.StatusBadRequest, "peran tanda tangan harus pencatat atau penyetuju", nil)
	}
	t, err := h.repo.FindTandaTangan(id, peran)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrPengajuanTTDTidakDitemukan, nil)
	}
	c.Set(fiber.HeaderContentType, "image/png")
	c.Set(fiber.HeaderCacheControl, "private, max-age=300")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(t.Data)
}

// StokMenipisRow: satu barang yang stoknya sudah di/bawah batas minimum.
type StokMenipisRow struct {
	BarangID    uint   `json:"barang_id"`
	KodeBarang  string `json:"kode_barang"`
	Nama        string `json:"nama"`
	Merek       string `json:"merek"`
	Tipe        string `json:"tipe"`
	Satuan      string `json:"satuan"`
	NamaGudang  string `json:"nama_gudang"`
	Stok        int    `json:"stok"`
	StokMinimum int    `json:"stok_minimum"`
	// SaranQty: jumlah yang disarankan diajukan, yaitu mengisi kembali sampai 2x
	// stok minimum (minimal 1). Hanya saran — pengaju bebas mengubahnya.
	SaranQty int `json:"saran_qty"`
	// PengajuanAktif: nomor pengajuan "masuk" yang masih menunggu persetujuan
	// untuk barang ini (kosong kalau belum ada) — supaya tidak diajukan dua kali.
	PengajuanAktif string `json:"pengajuan_aktif"`
}

func saranQty(stok, min int) int {
	n := min*2 - stok
	if n < 1 {
		n = 1
	}
	return n
}

// StokMenipis: GET /pengajuan-barang/stok-menipis — pintu masuk alur restock
// dari halaman Ringkasan Stok: daftar barang aktif & sudah disetujui yang stoknya
// <= stok minimum, lengkap dengan saran jumlah dan status pengajuan yang
// sedang berjalan. Pengajuannya sendiri dibuat lewat POST /pengajuan-barang
// (jenis "masuk") dengan tanda tangan digital pengaju.
func (h *Controller) StokMenipis(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	if p.Limit > maxLowStock {
		p.Limit = maxLowStock
	}
	list, total, err := h.barangRepo.List(p, barangRepoPkg.Filter{
		StokMenipis:      true,
		OnlyActive:       true,
		ApprovalStatuses: []string{constant.ApprovalDisetujui},
	})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar stok menipis", nil)
	}
	ids := make([]uint, 0, len(list))
	for _, b := range list {
		ids = append(ids, b.ID)
	}
	aktif, err := h.repo.OpenMasukByBarang(ids)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memeriksa pengajuan yang sedang berjalan", nil)
	}
	rows := make([]StokMenipisRow, 0, len(list))
	for _, b := range list {
		satuan := ""
		if b.Satuan != nil {
			satuan = b.Satuan.Nama
		}
		rows = append(rows, StokMenipisRow{
			BarangID: b.ID, KodeBarang: b.KodeBarang, Nama: b.Nama, Merek: b.Merek, Tipe: b.Tipe,
			Satuan: satuan, NamaGudang: b.NamaGudang, Stok: b.Stok, StokMinimum: b.StokMinimum,
			SaranQty: saranQty(b.Stok, b.StokMinimum), PengajuanAktif: aktif[b.ID],
		})
	}
	return utils.OKWithMeta(c, "daftar stok menipis berhasil diambil", rows, utils.BuildPaginationMeta(p, total))
}

// alurKeterangan: tambahan pesan sukses supaya pengaju tahu langkah berikutnya.
func alurKeterangan(p *model.PengajuanBarang) string {
	if p.Jenis != constant.JenisPengajuanMasuk {
		return ""
	}
	return " — sudah ditandatangani pengaju; menunggu tanda tangan & persetujuan pihak kedua, lalu dokumen barang masuk otomatis dibuat"
}
