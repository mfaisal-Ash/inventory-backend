package neraca

import (
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

// Deret waktu stok per gudang (harian / mingguan / bulanan / tahunan).
//
// Cara hitungnya sama dengan saldo akhir di neraca: dijangkarkan ke stok
// sistem saat ini (barang_stok_gudang) lalu dihitung mundur. Saldo di akhir
// sebuah periode = stok sekarang - seluruh mutasi yang terjadi SETELAH
// periode itu. Jadi grafiknya selalu berujung tepat di angka stok yang tampil
// di aplikasi, dan konsisten dengan neraca.
//
// Catatan satuan: kabel dihitung dalam meter dan barang lain dalam unit,
// persis seperti di neraca — total gudang adalah penjumlahan angka stok.

type Granularity string

const (
	Harian   Granularity = "daily"
	Mingguan Granularity = "weekly"
	Bulanan  Granularity = "monthly"
	Tahunan  Granularity = "yearly"

	// MaxBuckets membatasi jumlah titik per grafik supaya query dan payload
	// tetap ringan (mis. harian maksimal ~13 bulan).
	MaxBuckets = 400
)

// ParseGranularity menerima istilah Inggris maupun Indonesia. Kosong = harian.
func ParseGranularity(s string) (Granularity, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "daily", "harian", "day":
		return Harian, true
	case "weekly", "mingguan", "week":
		return Mingguan, true
	case "monthly", "bulanan", "month":
		return Bulanan, true
	case "yearly", "tahunan", "year":
		return Tahunan, true
	}
	return "", false
}

// BucketStart: awal periode yang memuat t (minggu mulai hari Senin).
func BucketStart(t time.Time, g Granularity) time.Time {
	y, m, d := t.Date()
	loc := t.Location()
	switch g {
	case Mingguan:
		wd := (int(t.Weekday()) + 6) % 7 // Senin = 0
		return time.Date(y, m, d-wd, 0, 0, 0, 0, loc)
	case Bulanan:
		return time.Date(y, m, 1, 0, 0, 0, 0, loc)
	case Tahunan:
		return time.Date(y, 1, 1, 0, 0, 0, 0, loc)
	}
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

// NextBucket: awal periode berikutnya (= batas akhir eksklusif periode ini).
func NextBucket(start time.Time, g Granularity) time.Time {
	switch g {
	case Mingguan:
		return start.AddDate(0, 0, 7)
	case Bulanan:
		return start.AddDate(0, 1, 0)
	case Tahunan:
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 0, 1)
}

// BuildBuckets: awal tiap periode dari periode yang memuat dari s.d. periode
// yang memuat sampai.
func BuildBuckets(dari, sampai time.Time, g Granularity) []time.Time {
	last := BucketStart(sampai, g)
	var out []time.Time
	for b := BucketStart(dari, g); !b.After(last); b = NextBucket(b, g) {
		out = append(out, b)
		if len(out) > MaxBuckets {
			break
		}
	}
	return out
}

// DefaultRange: rentang bawaan per granularitas — 30 hari, 12 minggu,
// 12 bulan, 5 tahun terakhir (periode berjalan ikut dihitung).
func DefaultRange(today time.Time, g Granularity) (dari, sampai time.Time) {
	today = BucketStart(today, Harian)
	switch g {
	case Mingguan:
		return BucketStart(today, Mingguan).AddDate(0, 0, -7*11), today
	case Bulanan:
		return BucketStart(today, Bulanan).AddDate(0, -11, 0), today
	case Tahunan:
		return BucketStart(today, Tahunan).AddDate(-4, 0, 0), today
	}
	return today.AddDate(0, 0, -29), today
}

type Period struct {
	Label   string `json:"label"`
	Mulai   string `json:"mulai"`
	Selesai string `json:"selesai"`
}

func periodOf(start time.Time, g Granularity) Period {
	const layout = "2006-01-02"
	label := start.Format(layout)
	switch g {
	case Bulanan:
		label = start.Format("2006-01")
	case Tahunan:
		label = start.Format("2006")
	}
	return Period{Label: label, Mulai: start.Format(layout), Selesai: NextBucket(start, g).AddDate(0, 0, -1).Format(layout)}
}

type SeriesGudang struct {
	GudangID   uint   `json:"gudang_id"`
	NamaGudang string `json:"nama_gudang"`
	// Saldo[i] = stok di akhir periode ke-i (sejajar dengan Periode).
	Saldo []int `json:"saldo"`
}

type SeriesResult struct {
	Granularity string         `json:"granularity"`
	Dari        string         `json:"dari"`
	Sampai      string         `json:"sampai"`
	Periode     []Period       `json:"periode"`
	Series      []SeriesGudang `json:"series"`
	Total       []int          `json:"total"`
}

type SeriesFilter struct {
	Dari, Sampai time.Time
	Granularity  Granularity
	GudangID     uint
}

// DeltaDay: total mutasi bertanda (masuk +, keluar -) satu gudang pada satu hari.
type DeltaDay struct {
	GudangID uint
	Day      time.Time
	Delta    int
}

// HitungSaldoSeries adalah inti perhitungan (tanpa database): saldo akhir tiap
// periode = stok sekarang - mutasi yang jatuh setelah periode itu.
func HitungSaldoSeries(buckets []time.Time, g Granularity, current map[uint]int, ev []DeltaDay) map[uint][]int {
	n := len(buckets)
	contrib := map[uint][]int{}
	ensure := func(id uint) []int {
		if c, ok := contrib[id]; ok {
			return c
		}
		c := make([]int, n+1)
		contrib[id] = c
		return c
	}
	for id := range current {
		ensure(id)
	}
	if n == 0 {
		return map[uint][]int{}
	}
	for _, e := range ev {
		if e.Day.Before(buckets[0]) {
			continue // mutasi sebelum rentang tidak memengaruhi saldo akhir periode mana pun
		}
		idx := sort.Search(n, func(i int) bool { return NextBucket(buckets[i], g).After(e.Day) })
		ensure(e.GudangID)[idx] += e.Delta // idx == n: setelah periode terakhir
	}
	out := make(map[uint][]int, len(contrib))
	for id, c := range contrib {
		saldo := make([]int, n)
		after := c[n]
		for i := n - 1; i >= 0; i-- {
			saldo[i] = current[id] - after
			after += c[i]
		}
		out[id] = saldo
	}
	return out
}

// utcOffsetSeconds: selisih zona waktu server terhadap UTC. Dipakai untuk
// menentukan "hari" sebuah kejadian menurut waktu setempat (WIB tidak
// punya DST, jadi offset tetap cukup).
func utcOffsetSeconds() int {
	_, off := time.Now().In(time.Local).Zone()
	return off
}

const seriesEventsSQL = evCTE + `
SELECT ev.gudang_id,
	((ev.ts AT TIME ZONE 'UTC') + INTERVAL '1 second' * @offset)::date AS day,
	SUM(ev.delta) AS delta
FROM ev
JOIN barang b ON b.id = ev.barang_id AND b.deleted_at IS NULL
WHERE ev.ts >= @fromTs AND (@gudangID = 0 OR ev.gudang_id = @gudangID)
GROUP BY 1, 2`

func Series(db *gorm.DB, f SeriesFilter) (*SeriesResult, error) {
	buckets := BuildBuckets(f.Dari, f.Sampai, f.Granularity)

	var cur []struct {
		GudangID uint
		Stok     int
	}
	if err := db.Raw(`
		SELECT bsg.gudang_id, COALESCE(SUM(bsg.stok), 0) AS stok
		FROM barang_stok_gudang bsg
		JOIN barang b ON b.id = bsg.barang_id AND b.deleted_at IS NULL
		WHERE (@gudangID = 0 OR bsg.gudang_id = @gudangID)
		GROUP BY bsg.gudang_id`, map[string]interface{}{"gudangID": int64(f.GudangID)}).Scan(&cur).Error; err != nil {
		return nil, err
	}
	current := make(map[uint]int, len(cur))
	for _, c := range cur {
		current[c.GudangID] = c.Stok
	}

	var raw []struct {
		GudangID uint
		Day      time.Time
		Delta    int
	}
	if len(buckets) > 0 {
		if err := db.Raw(seriesEventsSQL, map[string]interface{}{
			"bmSelesai": constant.StatusBMSelesai,
			"bkSelesai": constant.StatusBKSelesai,
			"rtSelesai": constant.StatusReturSelesai,
			"soSelesai": constant.StatusSOSelesai,
			"fromTs":    buckets[0],
			"offset":    int64(utcOffsetSeconds()),
			"gudangID":  int64(f.GudangID),
		}).Scan(&raw).Error; err != nil {
			return nil, err
		}
	}
	ev := make([]DeltaDay, 0, len(raw))
	loc := f.Dari.Location()
	for _, r := range raw {
		y, m, d := r.Day.Date() // kolom date dibaca sebagai UTC 00:00 — ambil Y/M/D-nya saja
		ev = append(ev, DeltaDay{GudangID: r.GudangID, Day: time.Date(y, m, d, 0, 0, 0, 0, loc), Delta: r.Delta})
	}

	saldo := HitungSaldoSeries(buckets, f.Granularity, current, ev)

	ids := make([]uint, 0, len(saldo))
	for id := range saldo {
		ids = append(ids, id)
	}
	names := map[uint]string{}
	if len(ids) > 0 {
		var gs []struct {
			ID   uint
			Nama string
		}
		if err := db.Table("gudangs").Select("id, nama").Where("id IN ?", ids).Scan(&gs).Error; err != nil {
			return nil, err
		}
		for _, g := range gs {
			names[g.ID] = g.Nama
		}
	}

	res := &SeriesResult{
		Granularity: string(f.Granularity),
		Dari:        BucketStart(f.Dari, Harian).Format("2006-01-02"),
		Sampai:      BucketStart(f.Sampai, Harian).Format("2006-01-02"),
		Periode:     make([]Period, 0, len(buckets)),
		Series:      []SeriesGudang{},
		Total:       make([]int, len(buckets)),
	}
	for _, b := range buckets {
		res.Periode = append(res.Periode, periodOf(b, f.Granularity))
	}
	for id, vals := range saldo {
		nonZero := false
		for i, v := range vals {
			res.Total[i] += v
			if v != 0 {
				nonZero = true
			}
		}
		if !nonZero {
			continue // gudang yang kosong sepanjang rentang tidak perlu digambar
		}
		res.Series = append(res.Series, SeriesGudang{GudangID: id, NamaGudang: names[id], Saldo: vals})
	}
	sort.Slice(res.Series, func(i, j int) bool { return res.Series[i].NamaGudang < res.Series[j].NamaGudang })
	return res, nil
}
