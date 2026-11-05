package pengajuan_barang

// Tes integrasi alur dua pihak + tanda tangan terhadap PostgreSQL sungguhan.
// Dilewati kecuali TEST_DATABASE_DSN diisi (nama database wajib berakhiran
// "_test" karena tes ini menghapus schema public), mis.:
//
//	TEST_DATABASE_DSN="host=localhost user=postgres dbname=inventory_test sslmode=disable" go test ./internal/repositories/pengajuan_barang/

import (
	"os"
	"regexp"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN tidak diisi — tes integrasi dilewati")
	}
	if !regexp.MustCompile(`dbname=\S*_test\b`).MatchString(dsn) {
		t.Fatal("nama database tes harus berakhiran _test")
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

func TestAlurDuaPihak_Integrasi(t *testing.T) {
	db := setupDB(t)
	repo := New(db)

	role := model.Role{Name: "admin"}
	must(t, db.Create(&role).Error)
	pengaju := model.User{Username: "pengaju", PasswordHash: "x", RoleID: role.ID}
	penyetuju := model.User{Username: "penyetuju", PasswordHash: "x", RoleID: role.ID}
	must(t, db.Create(&pengaju).Error)
	must(t, db.Create(&penyetuju).Error)
	g := model.Gudang{Nama: "Gudang A", Kode: "GA", Tipe: "pusat"}
	must(t, db.Create(&g).Error)
	kat := model.Kategori{Nama: "Kabel", Singkatan: "KB"}
	sat := model.Satuan{Nama: "Pieces", Singkatan: "pcs"}
	must(t, db.Create(&kat).Error)
	must(t, db.Create(&sat).Error)
	br := model.Barang{KodeBarang: "KB-001", Nama: "Router X", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		StokMinimum: 10, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	must(t, db.Create(&br).Error)

	buat := func() *model.PengajuanBarang {
		nomor, err := repo.NextNomor()
		must(t, err)
		p := &model.PengajuanBarang{NomorPengajuan: nomor, Jenis: constant.JenisPengajuanMasuk, GudangID: g.ID,
			Tanggal: day(2026, 10, 27), Status: constant.StatusPengajuanDiajukan, Keperluan: "restock", DiajukanOleh: pengaju.ID,
			Items: []model.PengajuanBarangItem{{BarangID: br.ID, Qty: "50"}}}
		p.TandaTangan = []model.PengajuanTandaTangan{{Peran: model.PeranTTDPencatat, UserID: pengaju.ID, Nama: "Pengaju",
			Data: []byte("png-pengaju"), DokumenHash: p.ComputeDokumenHash(), SignedAt: day(2026, 10, 27)}}
		must(t, repo.Create(p))
		return p
	}
	ttd := func() *model.PengajuanTandaTangan {
		return &model.PengajuanTandaTangan{Nama: "Penyetuju", Data: []byte("png-penyetuju")}
	}

	// Barang bersaldo <= minimum muncul di daftar stok menipis (filter yang sama dgn endpoint).
	low, total, err := barangRepo.New(db).List(utils.PaginationParams{Page: 1, Limit: 50}, barangRepo.Filter{
		StokMenipis: true, OnlyActive: true, ApprovalStatuses: []string{constant.ApprovalDisetujui},
	})
	if err != nil || total != 1 || len(low) != 1 || low[0].ID != br.ID {
		t.Fatalf("stok menipis = %d (%v) err=%v", total, low, err)
	}

	p := buat()

	// Detail memuat metadata TTD tetapi TIDAK gambarnya.
	got, err := repo.FindByID(p.ID)
	must(t, err)
	if s := got.TandaTanganPeran(model.PeranTTDPencatat); s == nil || len(s.Data) != 0 || s.DokumenHash == "" {
		t.Fatalf("metadata ttd pencatat harus ada tanpa gambar: %+v", s)
	}
	if img, err := repo.FindTandaTangan(p.ID, model.PeranTTDPencatat); err != nil || string(img.Data) != "png-pengaju" {
		t.Fatalf("FindTandaTangan: %v %+v", err, img)
	}

	// Pengajuan aktif muncul di daftar stok menipis.
	if m, err := repo.OpenMasukByBarang([]uint{br.ID}); err != nil || m[br.ID] != p.NomorPengajuan {
		t.Fatalf("OpenMasukByBarang = %v, %v", m, err)
	}

	// Penolakan: tanpa ttd, setuju sendiri.
	if _, err := repo.Setujui(p.ID, penyetuju.ID, "GA", "Manajer", "", nil); err == nil || err.Error() != constant.ErrPengajuanTTDPenyetujuWajib {
		t.Fatalf("tanpa ttd harus ditolak: %v", err)
	}
	if _, err := repo.Setujui(p.ID, pengaju.ID, "GA", "Manajer", "", ttd()); err == nil || err.Error() != constant.ErrPengajuanPenyetujuSamaDgnPengaju {
		t.Fatalf("setuju sendiri harus ditolak: %v", err)
	}

	// Edit isi -> tanda tangan lama gugur, lalu pengaju tanda tangan ulang.
	p.Items = []model.PengajuanBarangItem{{BarangID: br.ID, Qty: "80"}}
	must(t, repo.Update(p, p.Items, nil))
	if _, err := repo.Setujui(p.ID, penyetuju.ID, "GA", "Manajer", "", ttd()); err == nil || err.Error() != constant.ErrPengajuanTTDPencatatHilang {
		t.Fatalf("isi berubah tanpa tanda tangan ulang harus ditolak: %v", err)
	}
	baru := &model.PengajuanTandaTangan{Peran: model.PeranTTDPencatat, UserID: pengaju.ID, Nama: "Pengaju", Data: []byte("png-pengaju-2"),
		DokumenHash: p.ComputeDokumenHash(), SignedAt: day(2026, 10, 27)}
	must(t, repo.Update(p, p.Items, baru))

	// Alur normal: kedua pihak -> disetujui + dokumen barang masuk dibuat.
	hasil, err := repo.Setujui(p.ID, penyetuju.ID, "GA", "Manajer", "ok", ttd())
	if err != nil {
		t.Fatalf("alur normal harus lolos: %v", err)
	}
	if hasil.Status != constant.StatusPengajuanDisetujui {
		t.Fatalf("status = %s", hasil.Status)
	}
	if s := hasil.TandaTanganPeran(model.PeranTTDPenyetuju); s == nil || s.UserID != penyetuju.ID || s.DokumenHash != p.ComputeDokumenHash() {
		t.Fatalf("ttd penyetuju tidak tercatat benar: %+v", s)
	}
	var nBM int64
	db.Model(&model.BarangMasuk{}).Count(&nBM)
	if nBM != 1 {
		t.Fatalf("barang masuk draft harus otomatis dibuat, ada %d", nBM)
	}
	// Sudah disetujui -> tidak muncul lagi sebagai pengajuan aktif.
	if m, _ := repo.OpenMasukByBarang([]uint{br.ID}); len(m) != 0 {
		t.Fatalf("pengajuan yang sudah disetujui tidak boleh dianggap aktif: %v", m)
	}
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
