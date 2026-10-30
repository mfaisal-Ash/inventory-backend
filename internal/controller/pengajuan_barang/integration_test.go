package pengajuan_barang

// Tes HTTP untuk GET /stok-menipis dan GET /:id/ttd/:peran terhadap PostgreSQL
// sungguhan. Dilewati kecuali TEST_DATABASE_DSN diisi (database wajib
// berakhiran "_test" karena tes menghapus schema public).

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
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	pengajuanRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestStokMenipisDanTTD_HTTP(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN tidak diisi — tes integrasi dilewati")
	}
	if !regexp.MustCompile(`dbname=\S*_test\b`).MatchString(dsn) {
		t.Fatal("nama database tes harus berakhiran _test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	must(t, err)
	must(t, db.Exec("DROP SCHEMA public CASCADE").Error)
	must(t, db.Exec("CREATE SCHEMA public").Error)
	must(t, config.AutoMigrate(db))

	role := model.Role{Name: "admin"}
	must(t, db.Create(&role).Error)
	u := model.User{Username: "u", PasswordHash: "x", RoleID: role.ID}
	must(t, db.Create(&u).Error)
	g := model.Gudang{Nama: "Gudang A", Kode: "GA", Tipe: "pusat"}
	must(t, db.Create(&g).Error)
	kat := model.Kategori{Nama: "Kabel", Singkatan: "KB"}
	sat := model.Satuan{Nama: "Pieces", Singkatan: "pcs"}
	must(t, db.Create(&kat).Error)
	must(t, db.Create(&sat).Error)
	menipis := model.Barang{KodeBarang: "KB-001", Nama: "Router X", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		StokMinimum: 10, Stok: 4, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	aman := model.Barang{KodeBarang: "KB-002", Nama: "Switch Y", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		StokMinimum: 5, Stok: 50, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	must(t, db.Create(&menipis).Error)
	must(t, db.Create(&aman).Error)

	now := time.Now()
	p := model.PengajuanBarang{NomorPengajuan: "PJ-1", Jenis: constant.JenisPengajuanMasuk, GudangID: g.ID, Tanggal: now,
		Status: constant.StatusPengajuanDiajukan, DiajukanOleh: u.ID, Items: []model.PengajuanBarangItem{{BarangID: menipis.ID, Qty: 20}}}
	p.TandaTangan = []model.PengajuanTandaTangan{{Peran: model.PeranTTDPencatat, UserID: u.ID, Nama: "U",
		Data: []byte("PNGDATA"), DokumenHash: p.ComputeDokumenHash(), SignedAt: now}}
	must(t, db.Create(&p).Error)

	h := &Controller{repo: pengajuanRepo.New(db), barangRepo: barangRepo.New(db)}
	app := fiber.New()
	app.Get("/stok-menipis", h.StokMenipis)
	app.Get("/:id/ttd/:peran", h.ServeTTD)

	resp, err := app.Test(httptest.NewRequest("GET", "/stok-menipis", nil), -1)
	must(t, err)
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("stok-menipis -> %d: %s", resp.StatusCode, body)
	}
	var env struct {
		Data []StokMenipisRow `json:"data"`
	}
	must(t, json.Unmarshal(body, &env))
	if len(env.Data) != 1 {
		t.Fatalf("hanya barang menipis yang diharapkan, dapat %d: %s", len(env.Data), body)
	}
	r := env.Data[0]
	if r.BarangID != menipis.ID || r.Stok != 4 || r.StokMinimum != 10 || r.SaranQty != 16 || r.PengajuanAktif != "PJ-1" {
		t.Fatalf("baris stok menipis = %+v", r)
	}

	// Gambar tanda tangan: sukses dengan header aman; peran salah / tidak ada ditolak.
	resp, _ = app.Test(httptest.NewRequest("GET", "/1/ttd/pencatat", nil), -1)
	img, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(img) != "PNGDATA" || resp.Header.Get("Content-Type") != "image/png" ||
		resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("ttd pencatat -> %d %q %v", resp.StatusCode, img, resp.Header)
	}
	for url, want := range map[string]int{"/1/ttd/penyetuju": 404, "/1/ttd/ngawur": 400, "/999/ttd/pencatat": 404} {
		resp, _ = app.Test(httptest.NewRequest("GET", url, nil), -1)
		if resp.StatusCode != want {
			t.Errorf("%s -> %d, want %d", url, resp.StatusCode, want)
		}
	}
}
