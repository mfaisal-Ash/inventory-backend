// Package geofence menentukan apakah posisi perangkat berada di sekitar salah
// satu titik koordinat gudang. Hanya memakai stdlib supaya mudah diuji.
package geofence

import (
	"math"
	"strconv"
	"strings"
)

// DefaultRadiusMeters: radius default di sekitar titik gudang yang masih dianggap
// "di dalam gudang". Sengaja tidak terlalu kecil karena GPS di dalam bangunan
// bisa meleset puluhan meter.
const DefaultRadiusMeters = 200.0

// maxAccuracyBonusMeters: tambahan toleransi maksimum dari akurasi GPS yang
// dilaporkan perangkat. Dibatasi supaya perangkat dengan akurasi sangat buruk
// (mis. 5 km) tidak otomatis lolos.
const maxAccuracyBonusMeters = 100.0

const earthRadiusMeters = 6371000.0

// Point: titik koordinat gudang.
type Point struct {
	Name string
	Lat  float64
	Lng  float64
}

// Location: posisi yang dilaporkan perangkat pengguna.
type Location struct {
	Lat            float64
	Lng            float64
	AccuracyMeters float64
}

// Reason menjelaskan hasil evaluasi.
type Reason string

const (
	ReasonInside        Reason = "inside"
	ReasonNotConfigured Reason = "not_configured" // belum ada gudang berkoordinat
	ReasonNoLocation    Reason = "no_location"    // perangkat tidak mengirim lokasi
	ReasonOutside       Reason = "outside"
)

// Result hasil Evaluate.
type Result struct {
	Allowed        bool
	Reason         Reason
	DistanceMeters float64 // jarak ke gudang terdekat (0 kalau tidak dihitung)
	NearestName    string
}

// ValidLatLng true kalau pasangan koordinat ada di rentang yang sah.
func ValidLatLng(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// ParseLocation membaca lokasi dari teks header. accuracy boleh kosong/tidak valid
// (dianggap 0). ok=false kalau lat/lng tidak ada atau di luar rentang.
func ParseLocation(latStr, lngStr, accStr string) (Location, bool) {
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(lngStr), 64)
	if err1 != nil || err2 != nil || !ValidLatLng(lat, lng) {
		return Location{}, false
	}
	acc, err := strconv.ParseFloat(strings.TrimSpace(accStr), 64)
	if err != nil || math.IsNaN(acc) || math.IsInf(acc, 0) || acc < 0 {
		acc = 0
	}
	return Location{Lat: lat, Lng: lng, AccuracyMeters: acc}, true
}

// DistanceMeters: jarak lingkaran-besar (haversine) dua titik dalam meter.
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const rad = math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Min(1, math.Sqrt(a)))
}

// Evaluate memeriksa apakah loc berada dalam radiusMeters (+ toleransi akurasi
// GPS) dari salah satu points.
//   - points kosong  -> diizinkan (ReasonNotConfigured): belum ada koordinat
//     gudang yang diatur, jangan mengunci semua orang.
//   - hasLoc == false -> ditolak (ReasonNoLocation).
func Evaluate(points []Point, radiusMeters float64, loc Location, hasLoc bool) Result {
	if len(points) == 0 {
		return Result{Allowed: true, Reason: ReasonNotConfigured}
	}
	if !hasLoc {
		return Result{Allowed: false, Reason: ReasonNoLocation}
	}
	if radiusMeters <= 0 {
		radiusMeters = DefaultRadiusMeters
	}
	bonus := math.Min(loc.AccuracyMeters, maxAccuracyBonusMeters)

	best := math.MaxFloat64
	bestName := ""
	for _, p := range points {
		d := DistanceMeters(loc.Lat, loc.Lng, p.Lat, p.Lng)
		if d < best {
			best, bestName = d, p.Name
		}
	}
	res := Result{DistanceMeters: best, NearestName: bestName}
	if best <= radiusMeters+bonus {
		res.Allowed, res.Reason = true, ReasonInside
		return res
	}
	res.Reason = ReasonOutside
	return res
}
