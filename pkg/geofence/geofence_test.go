package geofence

import "testing"

// Titik gudang Bandung dari manajemen gudang.
var bandung = Point{Name: "Gudang Bandung", Lat: -6.922050329812074, Lng: 107.61644672392984}

func TestEvaluate(t *testing.T) {
	pts := []Point{bandung}
	tests := []struct {
		name   string
		points []Point
		loc    Location
		hasLoc bool
		want   bool
		reason Reason
	}{
		{"belum ada gudang berkoordinat -> bebas", nil, Location{}, false, true, ReasonNotConfigured},
		{"tidak kirim lokasi -> ditolak", pts, Location{}, false, false, ReasonNoLocation},
		{"tepat di titik gudang", pts, Location{Lat: bandung.Lat, Lng: bandung.Lng}, true, true, ReasonInside},
		{"~100 m dari gudang", pts, Location{Lat: bandung.Lat + 0.0009, Lng: bandung.Lng}, true, true, ReasonInside},
		{"~1 km dari gudang -> ditolak", pts, Location{Lat: bandung.Lat + 0.009, Lng: bandung.Lng}, true, false, ReasonOutside},
		{"Jakarta -> ditolak", pts, Location{Lat: -6.2, Lng: 106.8}, true, false, ReasonOutside},
		// ~250 m: di luar radius 200, tapi akurasi 100 m menambah toleransi.
		{"250 m tanpa toleransi -> ditolak", pts, Location{Lat: bandung.Lat + 0.00225, Lng: bandung.Lng}, true, false, ReasonOutside},
		{"250 m dengan akurasi 100 m -> boleh", pts, Location{Lat: bandung.Lat + 0.00225, Lng: bandung.Lng, AccuracyMeters: 100}, true, true, ReasonInside},
		{"akurasi 5 km tidak otomatis lolos", pts, Location{Lat: bandung.Lat + 0.009, Lng: bandung.Lng, AccuracyMeters: 5000}, true, false, ReasonOutside},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.points, 200, tt.loc, tt.hasLoc)
			if got.Allowed != tt.want || got.Reason != tt.reason {
				t.Fatalf("Evaluate = %+v, want allowed=%v reason=%v", got, tt.want, tt.reason)
			}
		})
	}
}

func TestEvaluateMultipleWarehousesUsesNearest(t *testing.T) {
	jakarta := Point{Name: "Gudang Jakarta", Lat: -6.2, Lng: 106.8}
	got := Evaluate([]Point{bandung, jakarta}, 200, Location{Lat: -6.2001, Lng: 106.8001}, true)
	if !got.Allowed || got.NearestName != "Gudang Jakarta" {
		t.Fatalf("want allowed near Jakarta, got %+v", got)
	}
}

func TestParseLocation(t *testing.T) {
	if _, ok := ParseLocation("", "", ""); ok {
		t.Fatal("kosong harus tidak valid")
	}
	if _, ok := ParseLocation("abc", "107", ""); ok {
		t.Fatal("bukan angka harus tidak valid")
	}
	if _, ok := ParseLocation("95", "107", ""); ok {
		t.Fatal("lat di luar rentang harus tidak valid")
	}
	if _, ok := ParseLocation("NaN", "107", ""); ok {
		t.Fatal("NaN harus tidak valid")
	}
	loc, ok := ParseLocation(" -6.92 ", "107.61", "12.5")
	if !ok || loc.Lat != -6.92 || loc.Lng != 107.61 || loc.AccuracyMeters != 12.5 {
		t.Fatalf("parse gagal: %+v ok=%v", loc, ok)
	}
	loc, ok = ParseLocation("-6.92", "107.61", "-5")
	if !ok || loc.AccuracyMeters != 0 {
		t.Fatalf("akurasi negatif harus jadi 0: %+v", loc)
	}
}
