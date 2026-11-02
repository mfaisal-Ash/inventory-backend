package skucode

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	p := Periode(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))
	if p != "31102026" {
		t.Fatalf("periode = %s", p)
	}
	got := Format(Prefix(KategoriCode("OP", "Operasional"), ProdukCode("ONT Huawei"), p), 1)
	if got != "OP-ONT-31102026-01" {
		t.Fatalf("got %s", got)
	}
	// Tanggal susulan (bukan hari ini) ikut tanggal yang dipilih pengguna.
	if Periode(time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)) != "05102026" {
		t.Fatal("tanggal 5 oktober")
	}
	if Periode(time.Date(2027, 1, 3, 0, 0, 0, 0, time.UTC)) != "03012027" {
		t.Fatal("tanggal januari")
	}
	if got := Format("OP-ONT-31102026", 123); got != "OP-ONT-31102026-123" {
		t.Fatalf("nomor >99: %s", got)
	}
}

func TestParseTanggal(t *testing.T) {
	now := time.Date(2026, 10, 31, 15, 4, 0, 0, time.UTC)
	if d, err := ParseTanggal("", now); err != nil || d.Day() != 31 {
		t.Fatalf("kosong harus hari ini: %v %v", d, err)
	}
	if d, err := ParseTanggal("2026-10-05", now); err != nil || Periode(d) != "05102026" {
		t.Fatalf("5 oktober: %v %v", d, err)
	}
	if d, err := ParseTanggal("2026-10-05T00:00:00Z", now); err != nil || d.Day() != 5 {
		t.Fatalf("ISO penuh: %v %v", d, err)
	}
	if _, err := ParseTanggal("2026-11-01", now); err == nil {
		t.Fatal("tanggal masa depan harus ditolak")
	}
	if _, err := ParseTanggal("05/10/2026", now); err == nil {
		t.Fatal("format salah harus ditolak")
	}
}

func TestKategoriCode(t *testing.T) {
	cases := [][3]string{
		{"ELK", "Elektronik", "ELK"},
		{"", "Operasional", "OPE"},
		{"", "Alat Tulis Kantor", "ATK"},
		{"op", "x", "OP"},
		{"", "", "BRG"},
	}
	for _, c := range cases {
		if got := KategoriCode(c[0], c[1]); got != c[2] {
			t.Errorf("KategoriCode(%q,%q)=%s want %s", c[0], c[1], got, c[2])
		}
	}
}

func TestProdukCode(t *testing.T) {
	cases := map[string]string{
		"Modem ZTE":         "MODEM",
		"kabel fiber optik": "KABEL",
		"  ONT Huawei":      "ONT",
		"Splitter-1:8":      "SPLITT",
		"":                  "BRG",
		"- Router":          "ROUTER",
	}
	for in, want := range cases {
		if got := ProdukCode(in); got != want {
			t.Errorf("ProdukCode(%q)=%s want %s", in, got, want)
		}
	}
}

func TestMaxNomor(t *testing.T) {
	codes := []string{
		"OP-MODEM-31102026-01", "OP-ROUTER-31102026-07", "OP-MODEM-30102026-50",
		"ELK-KABEL-31102026-12", "OPS-MODEM-31102026-99", "BRG-001", "OP-MODEM-31102026-abc",
		"OP-MODEM-102026-009", // format lama (MMYYYY) tidak ikut dihitung
	}
	if n := MaxNomor(codes, "OP", "31102026"); n != 7 {
		t.Errorf("OP 31102026 = %d want 7", n)
	}
	if n := MaxNomor(codes, "ELK", "31102026"); n != 12 {
		t.Errorf("ELK = %d want 12", n)
	}
	if n := MaxNomor(codes, "OP", "01112026"); n != 0 {
		t.Errorf("tanggal baru = %d want 0", n)
	}
}

func TestONT(t *testing.T) {
	for in, want := range map[string]bool{"ONT Huawei": true, "Modem ONT ZTE": true, "ont": true, "Fontana": false, "Kabel": false, "Mont-blanc": false} {
		if IsONT(in) != want {
			t.Errorf("IsONT(%q) want %v", in, want)
		}
	}
	if v, ok := NormalizeTipeONT("port hijau"); !ok || v != "Hijau" {
		t.Error("port hijau")
	}
	if _, ok := NormalizeTipeONT("Merah"); ok {
		t.Error("merah harus ditolak")
	}
}
