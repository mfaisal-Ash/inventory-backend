package gudang

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Status pengecekan fisik satu barang di satu gudang.
const (
	CekBelum        = "belum"           // belum pernah dicek
	CekSesuai       = "sesuai"          // fisik = sistem pada pengecekan terakhir
	CekBeda         = "beda"            // fisik != sistem pada pengecekan terakhir
	CekPerluUlang   = "perlu_cek_ulang" // stok sistem berubah sejak pengecekan terakhir
	maxItemCekFisik = 500
)

type RincianBarang struct {
	BarangID   uint   `json:"barang_id"`
	KodeBarang string `json:"kode_barang"`
	Nama       string `json:"nama"`
	Merek      string `json:"merek"`
	Tipe       string `json:"tipe"`
	Satuan     string `json:"satuan"`

	StokSistem int `json:"stok_sistem"`

	StatusCek   string     `json:"status_cek"`
	JumlahFisik *int       `json:"jumlah_fisik"`
	Selisih     *int       `json:"selisih"` // fisik - sistem pada pengecekan terakhir
	Catatan     string     `json:"catatan"`
	DicekOleh   string     `json:"dicek_oleh"`
	DicekPada   *time.Time `json:"dicek_pada"`
}

type RincianRingkasan struct {
	JumlahJenisBarang int `json:"jumlah_jenis_barang"`
	TotalStokSistem   int `json:"total_stok_sistem"`
	Belum             int `json:"belum"`
	Sesuai            int `json:"sesuai"`
	Beda              int `json:"beda"`
	PerluCekUlang     int `json:"perlu_cek_ulang"`
}

type RincianAset struct {
	JenisAset string `json:"jenis_aset"`
	Jumlah    int64  `json:"jumlah"`
}

type RincianGudangResponse struct {
	Gudang   *model.Gudang    `json:"gudang"`
	Ringkas  RincianRingkasan `json:"ringkasan"`
	Barang   []RincianBarang  `json:"barang"`
	Aset     []RincianAset    `json:"aset"`
	TotalAst int64            `json:"total_aset"`
}

type CekFisikItem struct {
	BarangID    uint   `json:"barang_id"`
	Sesuai      bool   `json:"sesuai"`
	JumlahFisik *int   `json:"jumlah_fisik"`
	Catatan     string `json:"catatan"`
}

type CekFisikRequest struct {
	Items []CekFisikItem `json:"items"`
}

type CekFisikResponse struct {
	Tersimpan int `json:"tersimpan"`
	Sesuai    int `json:"sesuai"`
	Beda      int `json:"beda"`
}

type RiwayatCekFisikRow struct {
	ID           uint      `json:"id"`
	GudangID     uint      `json:"gudang_id"`
	GudangNama   string    `json:"gudang_nama"`
	BarangID     uint      `json:"barang_id"`
	KodeBarang   string    `json:"kode_barang"`
	NamaBarang   string    `json:"nama_barang"`
	JumlahSistem int       `json:"jumlah_sistem"`
	JumlahFisik  int       `json:"jumlah_fisik"`
	Selisih      int       `json:"selisih"`
	Sesuai       bool      `json:"sesuai"`
	Catatan      string    `json:"catatan"`
	NamaCek      string    `json:"nama_cek"`
	DicekPada    time.Time `json:"dicek_pada"`
}

type rincianStokRow struct {
	BarangID   uint
	KodeBarang string
	Nama       string
	Merek      string
	Tipe       string
	Satuan     string
	StokSistem int
}

// RincianGudang: isi gudang secara rinci — tiap barang beserta stok sistem
// (Kelola Barang + mutasi) dan hasil pengecekan fisik terakhir — plus jumlah
// aset yang ditempatkan di gudang ini.
func (h *Controller) RincianGudang(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gudang tidak valid", nil)
	}
	g, err := h.repo.FindGudangByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "gudang tidak ditemukan", nil)
	}
	roleName, _ := c.Locals(constant.CtxRoleName).(string)
	maskProtectedOne(roleName, g)

	var stok []rincianStokRow
	if err := h.db.Raw(`
		SELECT b.id AS barang_id, b.kode_barang, b.nama, b.merek, b.tipe,
		       COALESCE(s.singkatan, '') AS satuan, bsg.stok AS stok_sistem
		FROM barang_stok_gudang bsg
		JOIN barang b ON b.id = bsg.barang_id AND b.deleted_at IS NULL
		LEFT JOIN satuan s ON s.id = b.satuan_id
		WHERE bsg.gudang_id = ? AND bsg.stok > 0
		ORDER BY b.nama ASC, b.kode_barang ASC`, id).Scan(&stok).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil isi gudang", nil)
	}

	var cek []model.CekFisikGudang
	if err := h.db.Raw(`
		SELECT DISTINCT ON (barang_id) *
		FROM cek_fisik_gudang
		WHERE gudang_id = ?
		ORDER BY barang_id, dicek_pada DESC, id DESC`, id).Scan(&cek).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil hasil pengecekan", nil)
	}
	terakhir := make(map[uint]model.CekFisikGudang, len(cek))
	for _, r := range cek {
		terakhir[r.BarangID] = r
	}

	res := RincianGudangResponse{Gudang: g, Barang: make([]RincianBarang, 0, len(stok)), Aset: []RincianAset{}}
	for _, s := range stok {
		row := RincianBarang{
			BarangID: s.BarangID, KodeBarang: s.KodeBarang, Nama: s.Nama, Merek: s.Merek, Tipe: s.Tipe,
			Satuan: s.Satuan, StokSistem: s.StokSistem, StatusCek: CekBelum,
		}
		if last, ok := terakhir[s.BarangID]; ok {
			fisik := last.JumlahFisik
			selisih := last.JumlahFisik - last.JumlahSistem
			dicek := last.DicekPada
			row.JumlahFisik = &fisik
			row.Selisih = &selisih
			row.Catatan = last.Catatan
			row.DicekOleh = last.NamaCek
			row.DicekPada = &dicek
			switch {
			case last.JumlahSistem != s.StokSistem:
				row.StatusCek = CekPerluUlang
			case last.Sesuai:
				row.StatusCek = CekSesuai
			default:
				row.StatusCek = CekBeda
			}
		}
		switch row.StatusCek {
		case CekSesuai:
			res.Ringkas.Sesuai++
		case CekBeda:
			res.Ringkas.Beda++
		case CekPerluUlang:
			res.Ringkas.PerluCekUlang++
		default:
			res.Ringkas.Belum++
		}
		res.Ringkas.JumlahJenisBarang++
		res.Ringkas.TotalStokSistem += s.StokSistem
		res.Barang = append(res.Barang, row)
	}

	if err := h.db.Raw(`
		SELECT jenis_aset, COUNT(*) AS jumlah
		FROM assets
		WHERE gudang_id = ? AND deleted_at IS NULL
		GROUP BY jenis_aset
		ORDER BY jenis_aset`, id).Scan(&res.Aset).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil aset gudang", nil)
	}
	for _, a := range res.Aset {
		res.TotalAst += a.Jumlah
	}
	return utils.OK(c, "rincian gudang berhasil diambil", res)
}

// SimpanCekFisik menyimpan hasil checklist. Barang yang dicentang "sesuai"
// dicatat fisik = stok sistem saat ini; yang tidak dicentang wajib mengisi
// jumlah fisik. Stok TIDAK diubah (koreksi lewat Stock Opname).
func (h *Controller) SimpanCekFisik(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gudang tidak valid", nil)
	}
	if _, err := h.repo.FindGudangByID(id); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "gudang tidak ditemukan", nil)
	}
	var req CekFisikRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "payload tidak valid", nil)
	}
	if len(req.Items) == 0 {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "tidak ada barang yang dicek", nil)
	}
	if len(req.Items) > maxItemCekFisik {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "terlalu banyak barang dalam satu kali simpan (maksimal 500)", nil)
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	var nama string
	h.db.Raw(`SELECT COALESCE(NULLIF(full_name, ''), username) FROM users WHERE id = ?`, userID).Scan(&nama)

	var stokRows []struct {
		BarangID uint
		Stok     int
	}
	if err := h.db.Raw(`SELECT barang_id, stok FROM barang_stok_gudang WHERE gudang_id = ?`, id).Scan(&stokRows).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membaca stok gudang", nil)
	}
	stokSistem := make(map[uint]int, len(stokRows))
	for _, r := range stokRows {
		stokSistem[r.BarangID] = r.Stok
	}

	now := time.Now()
	rows := make([]model.CekFisikGudang, 0, len(req.Items))
	seen := make(map[uint]bool, len(req.Items))
	var out CekFisikResponse
	for _, it := range req.Items {
		sistem, ada := stokSistem[it.BarangID]
		if !ada {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "barang id "+strconv.FormatUint(uint64(it.BarangID), 10)+" tidak ada di gudang ini", nil)
		}
		if seen[it.BarangID] {
			continue
		}
		seen[it.BarangID] = true

		fisik := sistem
		if !it.Sesuai {
			if it.JumlahFisik == nil || *it.JumlahFisik < 0 {
				return utils.Fail(c, fiber.StatusUnprocessableEntity, "jumlah fisik wajib diisi (minimal 0) untuk barang yang tidak dicentang sesuai", nil)
			}
			fisik = *it.JumlahFisik
		}
		catatan := strings.TrimSpace(it.Catatan)
		if len(catatan) > 255 {
			catatan = catatan[:255]
		}
		sesuai := fisik == sistem
		if sesuai {
			out.Sesuai++
		} else {
			out.Beda++
		}
		rows = append(rows, model.CekFisikGudang{
			GudangID: id, BarangID: it.BarangID, JumlahSistem: sistem, JumlahFisik: fisik, Sesuai: sesuai,
			Catatan: catatan, DicekOleh: userID, NamaCek: nama, DicekPada: now,
		})
	}
	if err := h.db.Create(&rows).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan hasil pengecekan", nil)
	}
	out.Tersimpan = len(rows)
	return utils.OK(c, "hasil pengecekan fisik berhasil disimpan", out)
}

// RiwayatCekFisik: riwayat pengecekan fisik terbaru (semua gudang, atau satu
// gudang lewat ?gudang_id=). Dipakai tab Pengecekan Barang.
func (h *Controller) RiwayatCekFisik(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 200)
	if limit < 1 || limit > 1000 {
		limit = 200
	}
	gudangID := c.QueryInt("gudang_id", 0)

	var rows []RiwayatCekFisikRow
	q := h.db.Table("cek_fisik_gudang AS c").
		Select(`c.id, c.gudang_id, g.nama AS gudang_nama, c.barang_id, b.kode_barang, b.nama AS nama_barang,
			c.jumlah_sistem, c.jumlah_fisik, (c.jumlah_fisik - c.jumlah_sistem) AS selisih, c.sesuai,
			c.catatan, c.nama_cek, c.dicek_pada`).
		Joins("JOIN gudangs g ON g.id = c.gudang_id").
		Joins("JOIN barang b ON b.id = c.barang_id").
		Order("c.dicek_pada DESC, c.id DESC").
		Limit(limit)
	if gudangID > 0 {
		q = q.Where("c.gudang_id = ?", gudangID)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat pengecekan", nil)
	}
	if rows == nil {
		rows = []RiwayatCekFisikRow{}
	}
	return utils.OK(c, "riwayat pengecekan fisik berhasil diambil", rows)
}
