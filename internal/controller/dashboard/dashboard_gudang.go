package dashboard

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	neracaRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const dayLayout = "2006-01-02"

func today() time.Time {
	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
}

func queryUint(c *fiber.Ctx, name string) uint {
	v, _ := strconv.ParseUint(c.Query(name, "0"), 10, 64)
	return uint(v)
}

// queryDay membaca ?name=YYYY-MM-DD. ok=false + pesan kalau formatnya salah;
// kosong -> (zero, true).
func queryDay(c *fiber.Ctx, name string) (t time.Time, set bool, errMsg string) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return time.Time{}, false, ""
	}
	t, err := time.ParseInLocation(dayLayout, raw, time.Local)
	if err != nil {
		return time.Time{}, false, "format " + name + " harus YYYY-MM-DD"
	}
	return t, true, ""
}

// StokGrafik: GET /dashboard/stok-grafik?granularity=daily|weekly|monthly|yearly
//
//	&dari=YYYY-MM-DD&sampai=YYYY-MM-DD&gudang_id=
//
// Grafik stok per gudang (satu garis per gudang + total). Tanpa dari/sampai
// dipakai rentang bawaan: 30 hari / 12 minggu / 12 bulan / 5 tahun terakhir.
func (h *Controller) StokGrafik(c *fiber.Ctx) error {
	g, ok := neracaRepo.ParseGranularity(c.Query("granularity"))
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "granularity harus daily, weekly, monthly, atau yearly", nil)
	}
	dari, hasDari, msg := queryDay(c, "dari")
	if msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	}
	sampai, hasSampai, msg := queryDay(c, "sampai")
	if msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	}
	defDari, defSampai := neracaRepo.DefaultRange(today(), g)
	if !hasSampai {
		sampai = defSampai
	}
	if !hasDari {
		if hasSampai {
			defDari, _ = neracaRepo.DefaultRange(sampai, g)
		}
		dari = defDari
	}
	if sampai.Before(dari) {
		return utils.Fail(c, fiber.StatusBadRequest, "tanggal sampai tidak boleh sebelum tanggal dari", nil)
	}
	if n := len(neracaRepo.BuildBuckets(dari, sampai, g)); n > neracaRepo.MaxBuckets {
		return utils.Fail(c, fiber.StatusBadRequest,
			fmt.Sprintf("rentang terlalu panjang untuk grafik %s (maksimal %d titik) — persempit rentang atau pilih granularitas lebih besar", g, neracaRepo.MaxBuckets), nil)
	}
	res, err := neracaRepo.Series(h.db, neracaRepo.SeriesFilter{
		Dari: dari, Sampai: sampai, Granularity: g, GudangID: queryUint(c, "gudang_id"),
	})
	if err != nil {
		log.Printf("dashboard/gudang: %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyusun grafik stok", nil)
	}
	return utils.OK(c, "grafik stok per gudang berhasil disusun", res)
}

// ── Tab "Stok Barang": seluruh stok per gudang, hasil akhir = neraca ─────────

type StokGudangRow struct {
	neracaRepo.Row
	HargaBeli  int64 `json:"harga_beli"`
	NilaiAkhir int64 `json:"nilai_akhir"` // saldo akhir x harga beli
}

type StokGudangGroup struct {
	GudangID     uint             `json:"gudang_id"`
	NamaGudang   string           `json:"nama_gudang"`
	JumlahBarang int              `json:"jumlah_barang"`
	Total        neracaRepo.Total `json:"total"`
	NilaiAkhir   int64            `json:"nilai_akhir"`
	Rows         []StokGudangRow  `json:"rows"`
}

type StokGudangResponse struct {
	Dari        string               `json:"dari"`
	Sampai      string               `json:"sampai"`
	Gudang      []StokGudangGroup    `json:"gudang"`
	GrandTotal  neracaRepo.Total     `json:"grand_total"`
	NilaiAkhir  int64                `json:"nilai_akhir"`
	Retur       neracaRepo.ReturInfo `json:"retur"`
	TakTerlacak int                  `json:"tak_terlacak"`
	// SelisihTotal: total selisih tak terlacak (stok sistem - mutasi berdokumen).
	SelisihTotal int `json:"selisih_total"`
}

func addTotal(t *neracaRepo.Total, r neracaRepo.Row) {
	t.SaldoAwal += r.SaldoAwal
	t.Masuk += r.Masuk
	t.Keluar += r.Keluar
	t.PakaiKabel += r.PakaiKabel
	t.ReturGanti += r.ReturGanti
	t.Opname += r.Opname
	t.StokAwalCatat += r.StokAwalCatat
	t.KeluarAwal += r.KeluarAwal
	t.SaldoAkhir += r.SaldoAkhir
}

// groupNeracaPerGudang mengelompokkan baris neraca per gudang (urut nama
// gudang) lengkap dengan total & nilai tiap gudang. Murni, tanpa database.
func groupNeracaPerGudang(rows []neracaRepo.Row, harga map[uint]int64) ([]StokGudangGroup, neracaRepo.Total, int64) {
	idx := map[uint]int{}
	var groups []StokGudangGroup
	var grand neracaRepo.Total
	var grandNilai int64
	for _, r := range rows {
		i, ok := idx[r.GudangID]
		if !ok {
			i = len(groups)
			idx[r.GudangID] = i
			groups = append(groups, StokGudangGroup{GudangID: r.GudangID, NamaGudang: r.NamaGudang, Rows: []StokGudangRow{}})
		}
		nilai := int64(r.SaldoAkhir) * harga[r.BarangID]
		gr := &groups[i]
		gr.Rows = append(gr.Rows, StokGudangRow{Row: r, HargaBeli: harga[r.BarangID], NilaiAkhir: nilai})
		gr.JumlahBarang++
		gr.NilaiAkhir += nilai
		addTotal(&gr.Total, r)
		addTotal(&grand, r)
		grandNilai += nilai
	}
	sort.SliceStable(groups, func(a, b int) bool { return groups[a].NamaGudang < groups[b].NamaGudang })
	return groups, grand, grandNilai
}

// StokGudang: GET /dashboard/stok-gudang?dari=&sampai=&gudang_id=&search=
// Tanpa dari/sampai: awal bulan berjalan s.d. hari ini. Angka akhirnya adalah
// saldo akhir neraca (saldo awal + masuk - keluar - kabel - retur ± opname),
// dikelompokkan per gudang, lengkap dengan nilai persediaan (saldo x harga beli).
func (h *Controller) StokGudang(c *fiber.Ctx) error {
	now := today()
	dari := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	sampai := now
	if t, set, msg := queryDay(c, "dari"); msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	} else if set {
		dari = t
	}
	if t, set, msg := queryDay(c, "sampai"); msg != "" {
		return utils.Fail(c, fiber.StatusBadRequest, msg, nil)
	} else if set {
		sampai = t
	}
	if sampai.Before(dari) {
		return utils.Fail(c, fiber.StatusBadRequest, "tanggal sampai tidak boleh sebelum tanggal dari", nil)
	}

	res, err := neracaRepo.New(h.db).Hitung(neracaRepo.Filter{
		Dari: dari, Sampai: sampai, GudangID: queryUint(c, "gudang_id"), Search: c.Query("search", ""),
	})
	if err != nil {
		log.Printf("dashboard/gudang: %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyusun stok per gudang", nil)
	}

	ids := make([]uint, 0, len(res.Rows))
	seen := map[uint]bool{}
	for _, r := range res.Rows {
		if !seen[r.BarangID] {
			seen[r.BarangID] = true
			ids = append(ids, r.BarangID)
		}
	}
	harga := map[uint]int64{}
	if len(ids) > 0 {
		var hb []struct {
			ID        uint
			HargaBeli int64
		}
		if err := h.db.Table("barang").Select("id, harga_beli").Where("id IN ?", ids).Scan(&hb).Error; err != nil {
			log.Printf("dashboard/gudang: %v", err)
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil harga barang", nil)
		}
		for _, x := range hb {
			harga[x.ID] = x.HargaBeli
		}
	}

	groups, grand, nilai := groupNeracaPerGudang(res.Rows, harga)
	return utils.OK(c, "stok barang per gudang berhasil disusun", StokGudangResponse{
		Dari: res.Dari, Sampai: res.Sampai, Gudang: groups, GrandTotal: grand, NilaiAkhir: nilai,
		Retur: res.Retur, TakTerlacak: res.TakTerlacak, SelisihTotal: res.SelisihTotal,
	})
}

// ── Tab "Asset barang": aset per gudang ──────────────────────────────────────

type AsetItem struct {
	ID        uint   `json:"id"`
	Nama      string `json:"nama"`
	JenisAset string `json:"jenis_aset"`
	LabelRSD  string `json:"label_rsd"`
	Status    string `json:"status"`
	Merek     string `json:"merek"`
	Tipe      string `json:"tipe"`
	NilaiAset int64  `json:"nilai_aset"`
}

type AsetGudangGroup struct {
	GudangID   uint                  `json:"gudang_id"`
	NamaGudang string                `json:"nama_gudang"`
	TotalAset  int64                 `json:"total_aset"`
	NilaiAset  int64                 `json:"nilai_aset"`
	PerStatus  []KategoriComposition `json:"per_status"`
	PerJenis   []KategoriComposition `json:"per_jenis"`
	Items      []AsetItem            `json:"items,omitempty"`
}

type AsetGudangResponse struct {
	TotalAset int64             `json:"total_aset"`
	NilaiAset int64             `json:"nilai_aset"`
	Gudang    []AsetGudangGroup `json:"gudang"`
}

const maxAsetItemsPerGudang = 500

func toComposition(m map[string]int64) []KategoriComposition {
	out := make([]KategoriComposition, 0, len(m))
	for k, v := range m {
		out = append(out, KategoriComposition{Label: k, Value: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// AsetGudang: GET /dashboard/aset-gudang?gudang_id=&status=&jenis=&search=&detail=true
// Stok aset per gudang: jumlah, nilai, sebaran status & jenis. detail=true
// menyertakan daftar aset (maks 500 per gudang).
func (h *Controller) AsetGudang(c *fiber.Ctx) error {
	q := h.db.Model(&model.Asset{}).
		Select("assets.id, assets.nama, assets.jenis_aset, assets.label_rsd, assets.status, assets.merek, assets.tipe, assets.nilai_aset, assets.gudang_id")
	if gid := queryUint(c, "gudang_id"); gid != 0 {
		q = q.Where("assets.gudang_id = ?", gid)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("LOWER(assets.status) = LOWER(?)", s)
	}
	if j := strings.TrimSpace(c.Query("jenis")); j != "" {
		q = q.Where("LOWER(assets.jenis_aset) = LOWER(?)", j)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("assets.nama ILIKE ? OR assets.label_rsd ILIKE ? OR assets.merek ILIKE ?", like, like, like)
	}
	var rows []struct {
		AsetItem
		GudangID uint
	}
	if err := q.Order("assets.nama").Scan(&rows).Error; err != nil {
		log.Printf("dashboard/gudang: %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil aset per gudang", nil)
	}

	names := map[uint]string{}
	var gs []struct {
		ID   uint
		Nama string
	}
	if err := h.db.Table("gudangs").Select("id, nama").Scan(&gs).Error; err != nil {
		log.Printf("dashboard/gudang: %v", err)
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar gudang", nil)
	}
	for _, g := range gs {
		names[g.ID] = g.Nama
	}

	detail := c.QueryBool("detail", false)
	type acc struct {
		g      AsetGudangGroup
		status map[string]int64
		jenis  map[string]int64
	}
	byG := map[uint]*acc{}
	resp := AsetGudangResponse{Gudang: []AsetGudangGroup{}}
	for _, r := range rows {
		a, ok := byG[r.GudangID]
		if !ok {
			a = &acc{g: AsetGudangGroup{GudangID: r.GudangID, NamaGudang: names[r.GudangID]}, status: map[string]int64{}, jenis: map[string]int64{}}
			byG[r.GudangID] = a
		}
		a.g.TotalAset++
		a.g.NilaiAset += r.NilaiAset
		a.status[r.Status]++
		a.jenis[r.JenisAset]++
		resp.TotalAset++
		resp.NilaiAset += r.NilaiAset
		if detail && len(a.g.Items) < maxAsetItemsPerGudang {
			a.g.Items = append(a.g.Items, r.AsetItem)
		}
	}
	for _, a := range byG {
		a.g.PerStatus = toComposition(a.status)
		a.g.PerJenis = toComposition(a.jenis)
		resp.Gudang = append(resp.Gudang, a.g)
	}
	sort.Slice(resp.Gudang, func(i, j int) bool { return resp.Gudang[i].NamaGudang < resp.Gudang[j].NamaGudang })
	return utils.OK(c, "aset per gudang berhasil diambil", resp)
}
