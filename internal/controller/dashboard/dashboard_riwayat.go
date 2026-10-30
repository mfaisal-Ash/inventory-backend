package dashboard

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Riwayat terpadu: barang masuk, barang keluar, barang rusak, retur, dan
// pengadaan (pengajuan barang masuk ke gudang) dalam satu linimasa. Tiap sumber
// diringkas menjadi SATU baris per dokumen dengan kolom yang sama.
//
// Fragmen SQL di bawah semuanya tetap (bukan dari input pengguna); input hanya
// dipakai sebagai parameter ber-nama, jadi aman dari injeksi.

const (
	jenisMasuk     = "masuk"
	jenisKeluar    = "keluar"
	jenisRusak     = "rusak"
	jenisRetur     = "retur"
	jenisPengadaan = "pengadaan"
)

var riwayatSQL = map[string]string{
	jenisMasuk: `
SELECT 'masuk' AS jenis, bm.id::bigint AS id, bm.nomor_penerimaan AS nomor,
	COALESCE(bm.completed_at, bm.tanggal) AS tanggal,
	bm.gudang_id::bigint AS gudang_id, COALESCE(g.nama, '') AS nama_gudang,
	COALESCE((SELECT string_agg(b.nama, ', ' ORDER BY b.nama) FROM barang_masuk_items i JOIN barang b ON b.id = i.barang_id WHERE i.barang_masuk_id = bm.id), '') AS barang,
	COALESCE((SELECT SUM(i.qty) FROM barang_masuk_items i WHERE i.barang_masuk_id = bm.id), 0)::bigint AS qty,
	bm.status AS status, COALESCE(bm.catatan, '') AS keterangan
FROM barang_masuk bm LEFT JOIN gudangs g ON g.id = bm.gudang_id`,

	jenisKeluar: `
SELECT 'keluar', bk.id::bigint, bk.nomor_pengeluaran,
	COALESCE(bk.completed_at, bk.tanggal),
	bk.gudang_id::bigint, COALESCE(g.nama, ''),
	COALESCE((SELECT string_agg(b.nama, ', ' ORDER BY b.nama) FROM barang_keluar_items i JOIN barang b ON b.id = i.barang_id WHERE i.barang_keluar_id = bk.id), ''),
	COALESCE((SELECT SUM(i.qty) FROM barang_keluar_items i WHERE i.barang_keluar_id = bk.id), 0)::bigint,
	bk.status, CONCAT_WS(' — ', NULLIF(bk.penerima, ''), NULLIF(bk.keperluan, ''))
FROM barang_keluar bk LEFT JOIN gudangs g ON g.id = bk.gudang_id`,

	// barang_rusak tidak menyimpan gudang; gudang = gudang asal barangnya
	// (barang.nama_gudang). Laporan rusak tanpa barang terhubung tampil tanpa gudang.
	jenisRusak: `
SELECT 'rusak', br.id::bigint, br.label_barang, br.created_at,
	(SELECT g2.id FROM gudangs g2 WHERE g2.nama = b.nama_gudang AND g2.deleted_at IS NULL ORDER BY g2.id LIMIT 1)::bigint,
	COALESCE(b.nama_gudang, ''),
	br.nama_barang, 1::bigint, br.status, COALESCE(br.keterangan, '')
FROM barang_rusak br LEFT JOIN barang b ON b.id = br.barang_id
WHERE br.deleted_at IS NULL`,

	jenisRetur: `
SELECT 'retur', rp.id::bigint, rp.nomor_retur, rp.created_at,
	rp.gudang_id::bigint, COALESCE(g.nama, ''),
	b.nama || ' (SN ' || rp.serial_lama || ')', 1::bigint, rp.status, rp.klien
FROM retur_penggantian rp
JOIN barang b ON b.id = rp.barang_id
LEFT JOIN gudangs g ON g.id = rp.gudang_id
WHERE rp.deleted_at IS NULL`,

	// Pengadaan = pengajuan barang masuk ke gudang (alur restock). Dokumen
	// barang masuk hasil persetujuan tercatat terpisah di jenis "masuk".
	jenisPengadaan: `
SELECT 'pengadaan', p.id::bigint, p.nomor_pengajuan, p.tanggal,
	p.gudang_id::bigint, COALESCE(g.nama, ''),
	COALESCE((SELECT string_agg(b.nama, ', ' ORDER BY b.nama) FROM pengajuan_barang_items i JOIN barang b ON b.id = i.barang_id WHERE i.pengajuan_id = p.id), ''),
	COALESCE((SELECT SUM(i.qty) FROM pengajuan_barang_items i WHERE i.pengajuan_id = p.id), 0)::bigint,
	p.status, COALESCE(p.keperluan, '')
FROM pengajuan_barang p LEFT JOIN gudangs g ON g.id = p.gudang_id
WHERE p.jenis = '` + constant.JenisPengajuanMasuk + `'`,
}

const riwayatKolom = "jenis, id, nomor, tanggal, gudang_id, nama_gudang, barang, qty, status, keterangan"

var riwayatUrutan = []string{jenisMasuk, jenisKeluar, jenisRusak, jenisRetur, jenisPengadaan}

type RiwayatRow struct {
	Jenis      string    `json:"jenis"`
	ID         uint      `json:"id"`
	Nomor      string    `json:"nomor"`
	Tanggal    time.Time `json:"tanggal"`
	GudangID   *uint     `json:"gudang_id"`
	NamaGudang string    `json:"nama_gudang"`
	Barang     string    `json:"barang"`
	Qty        int64     `json:"qty"`
	Status     string    `json:"status"`
	Keterangan string    `json:"keterangan"`
}

type RiwayatRingkasan struct {
	Jenis    string `json:"jenis"`
	Jumlah   int64  `json:"jumlah"`
	TotalQty int64  `json:"total_qty"`
}

type RiwayatResponse struct {
	Items     []RiwayatRow       `json:"items"`
	Ringkasan []RiwayatRingkasan `json:"ringkasan"`
}

// parseJenisRiwayat: "semua"/kosong -> semua jenis; atau daftar dipisah koma.
func parseJenisRiwayat(raw string) ([]string, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || raw == "semua" || raw == "all" {
		return riwayatUrutan, true
	}
	want := map[string]bool{}
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if _, ok := riwayatSQL[p]; !ok {
			return nil, false
		}
		want[p] = true
	}
	var out []string
	for _, j := range riwayatUrutan {
		if want[j] {
			out = append(out, j)
		}
	}
	return out, len(out) > 0
}

// buildRiwayatQuery menyusun subquery UNION ALL dan kondisi filternya.
func buildRiwayatQuery(jenis []string) string {
	parts := make([]string, 0, len(jenis))
	for _, j := range jenis {
		parts = append(parts, "("+riwayatSQL[j]+")")
	}
	return strings.Join(parts, "\nUNION ALL\n")
}

// Riwayat: GET /dashboard/riwayat?jenis=semua|masuk,keluar,rusak,retur,pengadaan
//
//	&dari=&sampai=&gudang_id=&status=&search=&page=&limit=
//
// Linimasa terbaru dulu. data.ringkasan = jumlah dokumen & total qty per jenis
// (mengikuti filter, tanpa paginasi).
func (h *Controller) Riwayat(c *fiber.Ctx) error {
	jenis, ok := parseJenisRiwayat(c.Query("jenis"))
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "jenis harus semua atau gabungan: masuk, keluar, rusak, retur, pengadaan", nil)
	}
	args := map[string]interface{}{}
	var where []string
	if t, set, msg := queryDay(c, "dari"); msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	} else if set {
		where = append(where, "t.tanggal >= @dari")
		args["dari"] = t
	}
	if t, set, msg := queryDay(c, "sampai"); msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	} else if set {
		where = append(where, "t.tanggal < @sampaiExcl")
		args["sampaiExcl"] = t.AddDate(0, 0, 1)
	}
	if gid := queryUint(c, "gudang_id"); gid != 0 {
		where = append(where, "t.gudang_id = @gudangID")
		args["gudangID"] = int64(gid)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		where = append(where, "LOWER(t.status) = LOWER(@status)")
		args["status"] = s
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		where = append(where, "(t.nomor ILIKE @like OR t.barang ILIKE @like OR t.keterangan ILIKE @like)")
		args["like"] = "%" + s + "%"
	}
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	// Nama kolom diberikan di sini (bukan di tiap fragmen) supaya tidak
	// bergantung pada fragmen mana yang kebetulan pertama dalam UNION.
	union := "(" + buildRiwayatQuery(jenis) + ") AS t(" + riwayatKolom + ")"

	var ringkasan []RiwayatRingkasan
	// Tanpa filter, map argumen kosong tidak boleh dikirim ke GORM (dianggap 1 argumen posisi).
	var ringkasanArgs []interface{}
	if len(args) > 0 {
		ringkasanArgs = append(ringkasanArgs, args)
	}
	if err := h.db.Raw("SELECT t.jenis AS jenis, COUNT(*) AS jumlah, COALESCE(SUM(t.qty), 0) AS total_qty FROM "+union+cond+" GROUP BY t.jenis ORDER BY t.jenis", ringkasanArgs...).
		Scan(&ringkasan).Error; err != nil {
		log.Printf("dashboard/riwayat (ringkasan): %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat", nil)
	}
	var total int64
	for _, r := range ringkasan {
		total += r.Jumlah
	}

	p := utils.PaginationFromContext(c)
	args["limit"] = p.Limit
	args["offset"] = (p.Page - 1) * p.Limit
	var items []RiwayatRow
	if err := h.db.Raw("SELECT t.* FROM "+union+cond+" ORDER BY t.tanggal DESC, t.id DESC LIMIT @limit OFFSET @offset", args).
		Scan(&items).Error; err != nil {
		log.Printf("dashboard/riwayat (items): %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat", nil)
	}
	if items == nil {
		items = []RiwayatRow{}
	}
	if ringkasan == nil {
		ringkasan = []RiwayatRingkasan{}
	}
	return utils.OKWithMeta(c, "riwayat gudang berhasil diambil",
		RiwayatResponse{Items: items, Ringkasan: ringkasan}, utils.BuildPaginationMeta(p, total))
}
