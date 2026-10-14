// Package neraca menyusun neraca stok (saldo awal, mutasi, saldo akhir) per
// barang per gudang untuk satu periode, dari seluruh kejadian yang mengubah
// stok: barang masuk, barang keluar, pemakaian kabel (meter), penggantian
// retur, dan penyesuaian stock opname.
//
// Saldo dijangkarkan ke stok sistem saat ini (barang_stok_gudang), lalu
// dihitung mundur: saldo akhir = stok sistem - mutasi setelah periode, saldo
// awal = saldo akhir - mutasi dalam periode. Dengan begitu neraca tetap
// cocok dengan stok yang tampil di aplikasi walau ada stok awal tanpa
// dokumen. Selisih antara stok sistem dan jumlah seluruh dokumen dilaporkan
// terpisah sebagai "selisih tak terlacak" supaya tidak tersembunyi.
package neraca

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

type Filter struct {
	Dari     time.Time // awal hari
	Sampai   time.Time // awal hari terakhir periode (inklusif)
	GudangID uint
	BarangID uint
	Search   string
}

type Row struct {
	BarangID   uint   `json:"barang_id"`
	KodeBarang string `json:"kode_barang"`
	NamaBarang string `json:"nama_barang"`
	Merek      string `json:"merek"`
	Satuan     string `json:"satuan"`
	GudangID   uint   `json:"gudang_id"`
	NamaGudang string `json:"nama_gudang"`

	SaldoAwal  int `json:"saldo_awal"`
	Masuk      int `json:"masuk"`
	Keluar     int `json:"keluar"`
	PakaiKabel int `json:"pakai_kabel"`
	ReturGanti int `json:"retur_ganti"`
	Opname     int `json:"opname"`
	SaldoAkhir int `json:"saldo_akhir"`

	// StokSistem = stok barang_stok_gudang saat ini; SelisihTakTerlacak =
	// StokSistem dikurangi total seluruh mutasi berdokumen sejak awal.
	StokSistem         int `json:"stok_sistem"`
	SelisihTakTerlacak int `json:"selisih_tak_terlacak"`
}

type Total struct {
	SaldoAwal  int `json:"saldo_awal"`
	Masuk      int `json:"masuk"`
	Keluar     int `json:"keluar"`
	PakaiKabel int `json:"pakai_kabel"`
	ReturGanti int `json:"retur_ganti"`
	Opname     int `json:"opname"`
	SaldoAkhir int `json:"saldo_akhir"`
}

type ReturInfo struct {
	SelesaiPeriode int64 `json:"selesai_periode"`
	MenungguGanti  int64 `json:"menunggu_ganti"`
	MenungguCek    int64 `json:"menunggu_cek"`
}

type Result struct {
	Dari   string    `json:"dari"`
	Sampai string    `json:"sampai"`
	Rows   []Row     `json:"rows"`
	Total  Total     `json:"total"`
	Retur  ReturInfo `json:"retur"`
	// TakTerlacak = jumlah baris yang punya selisih tak terlacak (untuk
	// peringatan di layar).
	TakTerlacak int `json:"tak_terlacak"`
}

type Repository interface {
	Hitung(f Filter) (*Result, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) Repository { return &repository{db: db} }

const maxRows = 5000

// Semua kejadian stok dalam satu CTE. delta bertanda (masuk +, keluar -).
const evCTE = `
WITH ev AS (
	SELECT bmi.barang_id, bm.gudang_id, COALESCE(bm.completed_at, bm.tanggal) AS ts, 'masuk' AS kind, bmi.qty AS delta
	FROM barang_masuk_items bmi JOIN barang_masuk bm ON bm.id = bmi.barang_masuk_id
	WHERE bm.status = @bmSelesai
	UNION ALL
	SELECT bki.barang_id, bk.gudang_id, COALESCE(bk.completed_at, bk.tanggal), 'keluar', -bki.qty
	FROM barang_keluar_items bki JOIN barang_keluar bk ON bk.id = bki.barang_keluar_id
	WHERE bk.status = @bkSelesai
	UNION ALL
	SELECT kg.barang_id, kg.gudang_id, kp.created_at, 'kabel', -kp.jumlah
	FROM kabel_pemakaian kp JOIN kabel_gulungan kg ON kg.id = kp.gulungan_id
	UNION ALL
	SELECT kg.barang_id, kg.gudang_id, kp.dibatalkan_pada, 'kabel', kp.jumlah
	FROM kabel_pemakaian kp JOIN kabel_gulungan kg ON kg.id = kp.gulungan_id
	WHERE kp.dibatalkan = true AND kp.dibatalkan_pada IS NOT NULL
	UNION ALL
	SELECT rp.barang_id, rp.gudang_id, rp.diganti_pada, 'retur', -1
	FROM retur_penggantian rp
	WHERE rp.status = @rtSelesai AND rp.diganti_pada IS NOT NULL AND rp.deleted_at IS NULL
	UNION ALL
	SELECT soi.barang_id, so.gudang_id, COALESCE(so.completed_at, so.tanggal), 'opname', soi.selisih
	FROM stock_opname_items soi JOIN stock_opname so ON so.id = soi.stock_opname_id
	WHERE so.status = @soSelesai
)`

const eventsCTE = evCTE + `,
agg AS (
	SELECT barang_id, gudang_id,
		COALESCE(SUM(delta) FILTER (WHERE kind = 'masuk'  AND ts >= @dari AND ts < @sampai), 0) AS masuk,
		COALESCE(-SUM(delta) FILTER (WHERE kind = 'keluar' AND ts >= @dari AND ts < @sampai), 0) AS keluar,
		COALESCE(-SUM(delta) FILTER (WHERE kind = 'kabel'  AND ts >= @dari AND ts < @sampai), 0) AS pakai_kabel,
		COALESCE(-SUM(delta) FILTER (WHERE kind = 'retur'  AND ts >= @dari AND ts < @sampai), 0) AS retur_ganti,
		COALESCE(SUM(delta) FILTER (WHERE kind = 'opname' AND ts >= @dari AND ts < @sampai), 0) AS opname,
		COALESCE(SUM(delta) FILTER (WHERE ts >= @dari AND ts < @sampai), 0) AS delta_periode,
		COALESCE(SUM(delta) FILTER (WHERE ts >= @sampai), 0) AS delta_sesudah,
		COALESCE(SUM(delta), 0) AS delta_total
	FROM ev
	GROUP BY barang_id, gudang_id
),
pairs AS (
	SELECT barang_id, gudang_id FROM barang_stok_gudang
	UNION
	SELECT barang_id, gudang_id FROM agg
)
SELECT p.barang_id, b.kode_barang, b.nama AS nama_barang, b.merek, COALESCE(s.singkatan, '') AS satuan,
	p.gudang_id, g.nama AS nama_gudang,
	COALESCE(a.masuk, 0) AS masuk, COALESCE(a.keluar, 0) AS keluar,
	COALESCE(a.pakai_kabel, 0) AS pakai_kabel, COALESCE(a.retur_ganti, 0) AS retur_ganti,
	COALESCE(a.opname, 0) AS opname,
	COALESCE(a.delta_periode, 0) AS delta_periode,
	COALESCE(a.delta_sesudah, 0) AS delta_sesudah,
	COALESCE(a.delta_total, 0) AS delta_total,
	COALESCE(bsg.stok, 0) AS stok_sistem
FROM pairs p
JOIN barang b ON b.id = p.barang_id AND b.deleted_at IS NULL
JOIN gudangs g ON g.id = p.gudang_id
LEFT JOIN satuan s ON s.id = b.satuan_id
LEFT JOIN agg a ON a.barang_id = p.barang_id AND a.gudang_id = p.gudang_id
LEFT JOIN barang_stok_gudang bsg ON bsg.barang_id = p.barang_id AND bsg.gudang_id = p.gudang_id
WHERE (@gudangID = 0 OR p.gudang_id = @gudangID)
  AND (@barangID = 0 OR p.barang_id = @barangID)
  AND (@search = '' OR b.nama ILIKE @like OR b.kode_barang ILIKE @like OR b.merek ILIKE @like)
ORDER BY b.nama, g.nama
LIMIT @maxRows`

// StokRow: posisi stok satu barang di satu gudang pada tanggal tertentu.
type StokRow struct {
	BarangID   uint   `json:"barang_id"`
	KodeBarang string `json:"kode_barang"`
	NamaBarang string `json:"nama_barang"`
	Merek      string `json:"merek"`
	Tipe       string `json:"tipe"`
	GudangID   uint   `json:"gudang_id"`
	NamaGudang string `json:"nama_gudang"`
	Stok       int    `json:"stok"`
}

// stokPerTanggalSQL: stok pada akhir hari `sampai` = stok sistem saat ini
// dikurangi seluruh mutasi yang terjadi SETELAH hari itu (cara hitung yang
// sama dengan saldo akhir di neraca stok).
const stokPerTanggalSQL = evCTE + `,
agg AS (
	SELECT barang_id, gudang_id,
		COALESCE(SUM(delta) FILTER (WHERE ts >= @sampai), 0) AS delta_sesudah
	FROM ev
	GROUP BY barang_id, gudang_id
),
pairs AS (
	SELECT barang_id, gudang_id FROM barang_stok_gudang
	UNION
	SELECT barang_id, gudang_id FROM agg
)
SELECT p.barang_id, b.kode_barang, b.nama AS nama_barang, b.merek, b.tipe,
	p.gudang_id, g.nama AS nama_gudang,
	(COALESCE(bsg.stok, 0) - COALESCE(a.delta_sesudah, 0)) AS stok
FROM pairs p
JOIN barang b ON b.id = p.barang_id
JOIN gudangs g ON g.id = p.gudang_id
LEFT JOIN agg a ON a.barang_id = p.barang_id AND a.gudang_id = p.gudang_id
LEFT JOIN barang_stok_gudang bsg ON bsg.barang_id = p.barang_id AND bsg.gudang_id = p.gudang_id
WHERE (COALESCE(bsg.stok, 0) - COALESCE(a.delta_sesudah, 0)) > 0
ORDER BY b.nama, g.nama`

// StokPerTanggal menghitung posisi stok per barang per gudang pada akhir
// hari `sampai` (jam diabaikan). Dipakai pilihan periode Bulan/Tahun di
// halaman Stok Barang; mode Real-time tetap memakai barang_stok_gudang.
func StokPerTanggal(db *gorm.DB, sampai time.Time) ([]StokRow, error) {
	akhir := time.Date(sampai.Year(), sampai.Month(), sampai.Day(), 0, 0, 0, 0, sampai.Location()).AddDate(0, 0, 1)
	var rows []StokRow
	err := db.Raw(stokPerTanggalSQL, map[string]interface{}{
		"bmSelesai": constant.StatusBMSelesai,
		"bkSelesai": constant.StatusBKSelesai,
		"rtSelesai": constant.StatusReturSelesai,
		"soSelesai": constant.StatusSOSelesai,
		"sampai":    akhir,
	}).Scan(&rows).Error
	return rows, err
}

type rawRow struct {
	Row
	DeltaPeriode int
	DeltaSesudah int
	DeltaTotal   int
}

func (r *repository) Hitung(f Filter) (*Result, error) {
	dari := f.Dari
	sampaiExcl := f.Sampai.AddDate(0, 0, 1)
	search := strings.TrimSpace(f.Search)

	var raw []rawRow
	if err := r.db.Raw(eventsCTE, map[string]interface{}{
		"bmSelesai": constant.StatusBMSelesai,
		"bkSelesai": constant.StatusBKSelesai,
		"rtSelesai": constant.StatusReturSelesai,
		"soSelesai": constant.StatusSOSelesai,
		"dari":      dari,
		"sampai":    sampaiExcl,
		"gudangID":  int64(f.GudangID),
		"barangID":  int64(f.BarangID),
		"search":    search,
		"like":      "%" + search + "%",
		"maxRows":   maxRows,
	}).Scan(&raw).Error; err != nil {
		return nil, err
	}

	res := &Result{
		Dari:   dari.Format("2006-01-02"),
		Sampai: f.Sampai.Format("2006-01-02"),
		Rows:   make([]Row, 0, len(raw)),
	}
	for _, x := range raw {
		row := x.Row
		row.StokSistem = x.StokSistem
		row.SaldoAkhir = x.StokSistem - x.DeltaSesudah
		row.SaldoAwal = row.SaldoAkhir - x.DeltaPeriode
		row.SelisihTakTerlacak = x.StokSistem - x.DeltaTotal

		// Baris yang sama sekali tidak bergerak dan kosong tidak perlu tampil.
		if row.SaldoAwal == 0 && row.SaldoAkhir == 0 && row.Masuk == 0 && row.Keluar == 0 &&
			row.PakaiKabel == 0 && row.ReturGanti == 0 && row.Opname == 0 {
			continue
		}
		if row.SelisihTakTerlacak != 0 {
			res.TakTerlacak++
		}
		res.Total.SaldoAwal += row.SaldoAwal
		res.Total.Masuk += row.Masuk
		res.Total.Keluar += row.Keluar
		res.Total.PakaiKabel += row.PakaiKabel
		res.Total.ReturGanti += row.ReturGanti
		res.Total.Opname += row.Opname
		res.Total.SaldoAkhir += row.SaldoAkhir
		res.Rows = append(res.Rows, row)
	}

	rq := r.db.Table("retur_penggantian").Where("deleted_at IS NULL")
	if f.GudangID != 0 {
		rq = rq.Where("gudang_id = ?", f.GudangID)
	}
	if f.BarangID != 0 {
		rq = rq.Where("barang_id = ?", f.BarangID)
	}
	var info struct {
		SelesaiPeriode int64
		MenungguGanti  int64
		MenungguCek    int64
	}
	if err := rq.Select(`
		COUNT(*) FILTER (WHERE status = ? AND diganti_pada >= ? AND diganti_pada < ?) AS selesai_periode,
		COUNT(*) FILTER (WHERE status = ?) AS menunggu_ganti,
		COUNT(*) FILTER (WHERE status = ?) AS menunggu_cek`,
		constant.StatusReturSelesai, dari, sampaiExcl,
		constant.StatusReturPerluGanti, constant.StatusReturDilaporkan).Scan(&info).Error; err != nil {
		return nil, err
	}
	res.Retur = ReturInfo{SelesaiPeriode: info.SelesaiPeriode, MenungguGanti: info.MenungguGanti, MenungguCek: info.MenungguCek}
	return res, nil
}
