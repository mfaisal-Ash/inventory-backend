package pengajuan_barang

import (
	"testing"
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func contoh(jenis string, diajukanOleh uint) *model.PengajuanBarang {
	return &model.PengajuanBarang{
		NomorPengajuan: "PJ-202610-0001", Jenis: jenis, GudangID: 1,
		Tanggal:   time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC),
		Keperluan: "Restock kabel", DiajukanOleh: diajukanOleh,
		Items: []model.PengajuanBarangItem{{BarangID: 7, Qty: 10}, {BarangID: 3, Qty: 2}},
	}
}

func ttdContoh() *model.PengajuanTandaTangan {
	return &model.PengajuanTandaTangan{Data: []byte("png")}
}

func pencatatUntuk(p *model.PengajuanBarang) *model.PengajuanTandaTangan {
	return &model.PengajuanTandaTangan{Peran: model.PeranTTDPencatat, DokumenHash: p.ComputeDokumenHash()}
}

func TestCekDuaPihak_AlurNormal(t *testing.T) {
	p := contoh(constant.JenisPengajuanMasuk, 1)
	ttd := ttdContoh()
	if err := CekDuaPihak(p, 2, ttd, pencatatUntuk(p)); err != nil {
		t.Fatalf("alur normal harus lolos, dapat: %v", err)
	}
	if ttd.DokumenHash != p.ComputeDokumenHash() {
		t.Fatal("hash dokumen harus ikut tersimpan di tanda tangan penyetuju")
	}
}

func TestCekDuaPihak_Penolakan(t *testing.T) {
	cases := []struct {
		nama    string
		mutasi  func(p *model.PengajuanBarang) (userID uint, ttd, pencatat *model.PengajuanTandaTangan)
		wantErr string
	}{
		{"tanpa TTD penyetuju", func(p *model.PengajuanBarang) (uint, *model.PengajuanTandaTangan, *model.PengajuanTandaTangan) {
			return 2, nil, pencatatUntuk(p)
		}, constant.ErrPengajuanTTDPenyetujuWajib},
		{"TTD penyetuju kosong", func(p *model.PengajuanBarang) (uint, *model.PengajuanTandaTangan, *model.PengajuanTandaTangan) {
			return 2, &model.PengajuanTandaTangan{}, pencatatUntuk(p)
		}, constant.ErrPengajuanTTDPenyetujuWajib},
		{"penyetuju = pengaju", func(p *model.PengajuanBarang) (uint, *model.PengajuanTandaTangan, *model.PengajuanTandaTangan) {
			return 1, ttdContoh(), pencatatUntuk(p)
		}, constant.ErrPengajuanPenyetujuSamaDgnPengaju},
		{"pengaju belum tanda tangan", func(p *model.PengajuanBarang) (uint, *model.PengajuanTandaTangan, *model.PengajuanTandaTangan) {
			return 2, ttdContoh(), nil
		}, constant.ErrPengajuanTTDPencatatHilang},
		{"isi berubah setelah pengaju tanda tangan", func(p *model.PengajuanBarang) (uint, *model.PengajuanTandaTangan, *model.PengajuanTandaTangan) {
			sig := pencatatUntuk(p)
			p.Items[0].Qty = 999 // diubah diam-diam sesudah ditandatangani
			return 2, ttdContoh(), sig
		}, constant.ErrPengajuanTTDPencatatHilang},
	}
	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			p := contoh(constant.JenisPengajuanMasuk, 1)
			userID, ttd, pencatat := tc.mutasi(p)
			err := CekDuaPihak(p, userID, ttd, pencatat)
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestCekDuaPihak_JenisLainTidakDiwajibkan(t *testing.T) {
	p := contoh(constant.JenisPengajuanKeluar, 1)
	if err := CekDuaPihak(p, 1, nil, nil); err != nil {
		t.Fatalf("jenis keluar tidak wajib TTD & boleh disetujui pengaju sendiri (perilaku lama), dapat: %v", err)
	}
}

func TestDokumenHash_StabilDanPekaIsi(t *testing.T) {
	a := contoh(constant.JenisPengajuanMasuk, 1)
	b := contoh(constant.JenisPengajuanMasuk, 1)
	b.Items[0], b.Items[1] = b.Items[1], b.Items[0] // urutan item tidak boleh memengaruhi hash
	if a.ComputeDokumenHash() != b.ComputeDokumenHash() {
		t.Fatal("urutan item tidak boleh mengubah hash")
	}
	// Tanggal yang dimuat ulang dari DB bisa berzona waktu lain — hash tetap sama.
	c := contoh(constant.JenisPengajuanMasuk, 1)
	c.Tanggal = c.Tanggal.In(time.FixedZone("WIB", 7*3600))
	if a.ComputeDokumenHash() != c.ComputeDokumenHash() {
		t.Fatal("zona waktu tidak boleh mengubah hash")
	}
	d := contoh(constant.JenisPengajuanMasuk, 1)
	d.Keperluan = "lain"
	if a.ComputeDokumenHash() == d.ComputeDokumenHash() {
		t.Fatal("perubahan isi harus mengubah hash")
	}
}
