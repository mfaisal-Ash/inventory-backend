package skucode

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	p := Periode(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC))
	if p != "102026" {
		t.Fatalf("periode = %s", p)
	}
	got := Format(Prefix(KategoriCode("OP", "Operasional"), ProdukCode("Modem ZTE F670L"), p), 1)
	if got != "OP-MODEM-102026-001" {
		t.Fatalf("got %s", got)
	}
	if Periode(time.Date(2027, 1, 3, 0, 0, 0, 0, time.UTC)) != "012027" {
		t.Fatal("periode januari")
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
		"OP-MODEM-102026-001", "OP-ROUTER-102026-007", "OP-MODEM-092026-050",
		"ELK-KABEL-102026-012", "OPS-MODEM-102026-099", "BRG-001", "OP-MODEM-102026-abc",
	}
	if n := MaxNomor(codes, "OP", "102026"); n != 7 {
		t.Errorf("OP 102026 = %d want 7", n)
	}
	if n := MaxNomor(codes, "ELK", "102026"); n != 12 {
		t.Errorf("ELK = %d want 12", n)
	}
	if n := MaxNomor(codes, "OP", "112026"); n != 0 {
		t.Errorf("periode baru = %d want 0", n)
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
