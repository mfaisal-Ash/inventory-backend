package geocoding

import (
	"strings"
	"testing"
)

func TestSplitStreet(t *testing.T) {
	cases := []struct{ in, name, number string }{
		{"Jl. Asia Afrika No.141-149", "Jalan Asia Afrika", "141"},
		{"Jl. Merdeka No 12A", "Jalan Merdeka", "12A"},
		{"Jln Braga Nomor 7", "Jalan Braga", "7"},
		{"Jalan Cimuncang", "Jalan Cimuncang", ""},
	}
	for _, c := range cases {
		name, number := splitStreet(c.in)
		if name != c.name || number != c.number {
			t.Errorf("splitStreet(%q) = %q,%q want %q,%q", c.in, name, number, c.name, c.number)
		}
	}
}

func TestParseAddressWithRange(t *testing.T) {
	p := parseIndonesianAddress("Jl. Asia Afrika No.141-149, Kb. Pisang, Kec. Sumur Bandung, Kota Bandung, Jawa Barat")
	if p.street != "Jalan Asia Afrika" || p.houseNumber != "141" {
		t.Fatalf("street=%q number=%q", p.street, p.houseNumber)
	}
	if p.county != "Kota Bandung" || p.state != "Jawa Barat" {
		t.Errorf("county=%q state=%q", p.county, p.state)
	}
	c := buildFreeTextCandidates(p)
	if len(c) < 3 || !strings.HasPrefix(c[0], "141 Jalan Asia Afrika, ") {
		t.Errorf("kandidat pertama harus memuat nomor rumah: %v", c)
	}
}

func TestRtRwStillStripped(t *testing.T) {
	p := parseIndonesianAddress("Jl. Cimuncang No.RT03/13, Cimuncang, Kab. Bandung, Jawa Barat 40375")
	if p.street != "Jalan Cimuncang" || p.houseNumber != "" || p.postalcode != "40375" {
		t.Errorf("street=%q number=%q postal=%q", p.street, p.houseNumber, p.postalcode)
	}
}

func TestClassifyPrecision(t *testing.T) {
	if classifyPrecision(nominatimAddr{HouseNumber: "141", Road: "Jalan Asia Afrika"}) != PrecisionHouse {
		t.Error("harus house")
	}
	if classifyPrecision(nominatimAddr{Road: "Jalan Asia Afrika"}) != PrecisionStreet {
		t.Error("harus street")
	}
}
