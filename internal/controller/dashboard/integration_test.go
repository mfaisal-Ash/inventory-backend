package dashboard

// Tes integrasi untuk endpoint Analisa Data (grafik stok, riwayat, stok & aset
// per gudang) terhadap PostgreSQL sungguhan. Dilewati kecuali
// TEST_DATABASE_DSN diisi, mis.:
//
//	TEST_DATABASE_DSN="host=localhost user=postgres dbname=inventory_test sslmode=disable" go test ./internal/controller/dashboard/
//
// PERINGATAN: tes ini MENGHAPUS seluruh isi schema public. Karena itu nama
// database wajib berakhiran "_test".

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 10, 0, 0, 0, time.Local) }

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN tidak diisi — tes integrasi dilewati")
	}
	if !regexp.MustCompile(`dbname=\S*_test\b`).MatchString(dsn) {
		t.Fatal("nama database tes harus berakhiran _test (tes ini menghapus schema public)")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	must(t, db.Exec("DROP SCHEMA public CASCADE").Error)
	must(t, db.Exec("CREATE SCHEMA public").Error)
	must(t, config.AutoMigrate(db))
	return db
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// seed: 2 gudang; satu barang (harga 1000). Stok sekarang A=100, B=20.
// Gudang A: masuk +30 (20 Okt), keluar -5 (24 Okt), masuk +10 (26 Okt).
// Gudang B: keluar -4 (26 Okt).
func seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	role := model.Role{Name: "karyawan"}
	must(t, db.Create(&role).Error)
	user := model.User{Username: "tester", PasswordHash: "x", RoleID: role.ID}
	must(t, db.Create(&user).Error)
	ga := model.Gudang{Nama: "Gudang A", Kode: "GA", Tipe: "pusat"}
	gb := model.Gudang{Nama: "Gudang B", Kode: "GB", Tipe: "cabang"}
	must(t, db.Create(&ga).Error)
	must(t, db.Create(&gb).Error)
	kat := model.Kategori{Nama: "Kabel", Singkatan: "KB"}
	sat := model.Satuan{Nama: "Pieces", Singkatan: "pcs"}
	must(t, db.Create(&kat).Error)
	must(t, db.Create(&sat).Error)
	br := model.Barang{KodeBarang: "KB-001", Nama: "Router X", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		HargaBeli: 1000, StokMinimum: 10, Stok: 120, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	must(t, db.Create(&br).Error)
	must(t, db.Create(&model.BarangStokGudang{BarangID: br.ID, GudangID: ga.ID, Stok: 100}).Error)
	must(t, db.Create(&model.BarangStokGudang{BarangID: br.ID, GudangID: gb.ID, Stok: 20}).Error)

	masuk := func(nomor string, g uint, at time.Time, qty int) {
		bm := model.BarangMasuk{NomorPenerimaan: nomor, GudangID: g, Status: constant.StatusBMSelesai, Tanggal: at, CompletedAt: &at,
			Items: []model.BarangMasukItem{{BarangID: br.ID, Qty: qty, HargaSatuan: 1000}}}
		must(t, db.Create(&bm).Error)
	}
	keluar := func(nomor string, g uint, at time.Time, qty int) {
		bk := model.BarangKeluar{NomorPengeluaran: nomor, GudangID: g, Status: constant.StatusBKSelesai, Tanggal: at, CompletedAt: &at,
			Penerima: "Tim Lapangan", Items: []model.BarangKeluarItem{{BarangID: br.ID, Qty: qty}}}
		must(t, db.Create(&bk).Error)
	}
	masuk("BM-1", ga.ID, day(2026, 10, 20), 30)
	masuk("BM-2", ga.ID, day(2026, 10, 26), 10)
	keluar("BK-1", ga.ID, day(2026, 10, 24), 5)
	keluar("BK-2", gb.ID, day(2026, 10, 26), 4)

	must(t, db.Create(&model.BarangRusak{BarangID: &br.ID, LabelBarang: "RSK-1", NamaBarang: "Router X", Status: constant.StatusPengecekan, DilaporkanOleh: user.ID}).Error)
	must(t, db.Create(&model.ReturPenggantian{NomorRetur: "RTR-1", BarangID: br.ID, GudangID: gb.ID, Klien: "PT Maju", SerialLama: "SN1", Keluhan: "mati", Status: "dilaporkan", DilaporkanOleh: user.ID}).Error)
	must(t, db.Create(&model.PengajuanBarang{NomorPengajuan: "PJ-1", Jenis: constant.JenisPengajuanMasuk, GudangID: ga.ID, Tanggal: day(2026, 10, 25),
		Status: constant.StatusPengajuanDiajukan, Keperluan: "restock", DiajukanOleh: user.ID,
		Items: []model.PengajuanBarangItem{{BarangID: br.ID, Qty: "50"}}}).Error)

	must(t, db.Create(&model.Asset{Nama: "Server 1", JenisAset: "perangkat", GudangID: ga.ID, Status: "aktif", NilaiAset: 5000}).Error)
	must(t, db.Create(&model.Asset{Nama: "Server 2", JenisAset: "perangkat", GudangID: ga.ID, Status: "rusak", NilaiAset: 3000}).Error)
	must(t, db.Create(&model.Asset{Nama: "Mobil", JenisAset: "transportasi", GudangID: gb.ID, Status: "aktif", NilaiAset: 90000}).Error)
}

func get(t *testing.T, app *fiber.App, url string, out interface{}) {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", url, nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("GET %s -> %d: %s", url, resp.StatusCode, body)
	}
	var env struct {
		Data json.RawMessage `json:"data"`
	}
	must(t, json.Unmarshal(body, &env))
	must(t, json.Unmarshal(env.Data, out))
}

func TestAnalisaGudang_Integrasi(t *testing.T) {
	db := setupDB(t)
	seed(t, db)
	h := &Controller{db: db}
	app := fiber.New()
	app.Get("/stok-grafik", h.StokGrafik)
	app.Get("/riwayat", h.Riwayat)
	app.Get("/stok-gudang", h.StokGudang)
	app.Get("/aset-gudang", h.AsetGudang)

	t.Run("grafik stok harian per gudang", func(t *testing.T) {
		var res struct {
			Periode []struct{ Label string } `json:"periode"`
			Series  []struct {
				NamaGudang string `json:"nama_gudang"`
				Saldo      []int  `json:"saldo"`
			} `json:"series"`
			Total []int `json:"total"`
		}
		get(t, app, "/stok-grafik?granularity=daily&dari=2026-10-18&sampai=2026-10-27", &res)
		if len(res.Periode) != 10 || len(res.Series) != 2 {
			t.Fatalf("periode=%d series=%d", len(res.Periode), len(res.Series))
		}
		a := res.Series[0] // Gudang A
		if a.NamaGudang != "Gudang A" {
			t.Fatalf("urutan gudang: %v", a.NamaGudang)
		}
		// 18..27 Okt
		want := []int{65, 65, 95, 95, 95, 95, 90, 90, 100, 100}
		for i := range want {
			if a.Saldo[i] != want[i] {
				t.Fatalf("saldo A = %v, want %v", a.Saldo, want)
			}
		}
		// Gudang B: sekarang 20; keluar 4 pada 26 Okt -> akhir 25 Okt = 24.
		if b := res.Series[1].Saldo; b[7] != 24 || b[9] != 20 {
			t.Fatalf("saldo B = %v", b)
		}
		if res.Total[9] != 120 {
			t.Fatalf("total akhir harus = stok sistem 120, dapat %d", res.Total[9])
		}
	})

	t.Run("grafik bulanan, mingguan, tahunan memakai angka yang sama di ujung", func(t *testing.T) {
		for _, g := range []string{"weekly", "monthly", "yearly"} {
			var res struct {
				Total []int `json:"total"`
			}
			get(t, app, "/stok-grafik?granularity="+g+"&sampai=2026-10-27", &res)
			if len(res.Total) == 0 || res.Total[len(res.Total)-1] != 120 {
				t.Fatalf("%s: titik terakhir harus 120, dapat %v", g, res.Total)
			}
		}
	})

	t.Run("filter gudang", func(t *testing.T) {
		var res struct {
			Series []struct {
				NamaGudang string `json:"nama_gudang"`
			} `json:"series"`
		}
		get(t, app, "/stok-grafik?granularity=daily&dari=2026-10-20&sampai=2026-10-27&gudang_id=2", &res)
		if len(res.Series) != 1 || res.Series[0].NamaGudang != "Gudang B" {
			t.Fatalf("hanya Gudang B yang diharapkan: %+v", res.Series)
		}
	})

	t.Run("riwayat semua jenis", func(t *testing.T) {
		var res RiwayatResponse
		get(t, app, "/riwayat", &res)
		got := map[string]int64{}
		for _, r := range res.Ringkasan {
			got[r.Jenis] = r.Jumlah
		}
		want := map[string]int64{"masuk": 2, "keluar": 2, "rusak": 1, "retur": 1, "pengadaan": 1}
		for k, v := range want {
			if got[k] != v {
				t.Fatalf("ringkasan = %v, want %v", got, want)
			}
		}
		if len(res.Items) != 7 {
			t.Fatalf("items = %d, want 7", len(res.Items))
		}
		for i := 1; i < len(res.Items); i++ {
			if res.Items[i].Tanggal.After(res.Items[i-1].Tanggal) {
				t.Fatal("riwayat harus terbaru dulu")
			}
		}
	})

	t.Run("riwayat difilter jenis, gudang, tanggal", func(t *testing.T) {
		var res RiwayatResponse
		get(t, app, "/riwayat?jenis=keluar&gudang_id=2&dari=2026-10-25&sampai=2026-10-27", &res)
		if len(res.Items) != 1 || res.Items[0].Nomor != "BK-2" || res.Items[0].Qty != 4 {
			t.Fatalf("items = %+v", res.Items)
		}
		var rs RiwayatResponse
		get(t, app, "/riwayat?jenis=masuk,pengadaan&search=router", &rs)
		if len(rs.Items) != 3 {
			t.Fatalf("masuk+pengadaan mengandung 'router' = %d, want 3", len(rs.Items))
		}
	})

	t.Run("stok barang per gudang = neraca", func(t *testing.T) {
		var res StokGudangResponse
		get(t, app, "/stok-gudang?dari=2026-10-18&sampai=2026-10-27", &res)
		if len(res.Gudang) != 2 {
			t.Fatalf("gudang = %d", len(res.Gudang))
		}
		a := res.Gudang[0]
		if a.NamaGudang != "Gudang A" || a.Total.SaldoAwal != 65 || a.Total.Masuk != 40 || a.Total.Keluar != 5 || a.Total.SaldoAkhir != 100 {
			t.Fatalf("neraca A = %+v", a.Total)
		}
		if a.NilaiAkhir != 100*1000 {
			t.Fatalf("nilai A = %d", a.NilaiAkhir)
		}
		if res.GrandTotal.SaldoAkhir != 120 || res.NilaiAkhir != 120*1000 {
			t.Fatalf("grand = %+v nilai=%d", res.GrandTotal, res.NilaiAkhir)
		}
	})

	t.Run("aset per gudang", func(t *testing.T) {
		var res AsetGudangResponse
		get(t, app, "/aset-gudang?detail=true", &res)
		if res.TotalAset != 3 || res.NilaiAset != 98000 || len(res.Gudang) != 2 {
			t.Fatalf("aset = %+v", res)
		}
		a := res.Gudang[0]
		if a.NamaGudang != "Gudang A" || a.TotalAset != 2 || a.NilaiAset != 8000 || len(a.Items) != 2 || len(a.PerStatus) != 2 {
			t.Fatalf("aset A = %+v", a)
		}
		var kosong AsetGudangResponse
		get(t, app, "/aset-gudang?status=rusak", &kosong)
		if kosong.TotalAset != 1 || len(kosong.Gudang[0].Items) != 0 {
			t.Fatalf("filter status / tanpa detail: %+v", kosong)
		}
	})

	t.Run("parameter tidak valid ditolak", func(t *testing.T) {
		for _, u := range []string{"/stok-grafik?granularity=hourly", "/stok-grafik?dari=bukan-tanggal", "/riwayat?jenis=ngawur",
			"/stok-grafik?granularity=daily&dari=2020-01-01&sampai=2026-10-27"} {
			resp, _ := app.Test(httptest.NewRequest("GET", u, nil), -1)
			if resp.StatusCode != 400 {
				t.Errorf("%s -> %d, want 400", u, resp.StatusCode)
			}
		}
	})
}
