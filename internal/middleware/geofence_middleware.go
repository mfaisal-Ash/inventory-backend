package middleware

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geofence"
)

// Header lokasi yang dikirim frontend di setiap request.
const (
	GeoLatHeader = "X-Geo-Lat"
	GeoLngHeader = "X-Geo-Lng"
	GeoAccHeader = "X-Geo-Acc"
)

// GeofenceNoLocationMessage: pesan saat perangkat tidak mengirim lokasi.
const GeofenceNoLocationMessage = "lokasi perangkat tidak terdeteksi. Aktifkan GPS dan izinkan akses lokasi di browser, lalu coba lagi"

// geofenceCacheTTL: seberapa lama daftar titik gudang di-cache. Perubahan
// koordinat di Manajemen Gudang berlaku paling lambat selama ini.
const geofenceCacheTTL = 30 * time.Second

// GeofenceProvider mengambil titik koordinat semua gudang yang sudah punya
// latitude/longitude.
type GeofenceProvider func() ([]geofence.Point, error)

var geofenceState struct {
	mu       sync.Mutex
	provider GeofenceProvider
	radius   float64
	points   []geofence.Point
	loadedAt time.Time
	loaded   bool
}

// ConfigureGeofence memasang sumber titik gudang dan radius (meter). Dipanggil
// saat router dibuat. provider nil = pembatasan lokasi mati.
func ConfigureGeofence(provider GeofenceProvider, radiusMeters float64) {
	if radiusMeters <= 0 {
		radiusMeters = geofence.DefaultRadiusMeters
	}
	geofenceState.mu.Lock()
	defer geofenceState.mu.Unlock()
	geofenceState.provider = provider
	geofenceState.radius = radiusMeters
	geofenceState.loaded = false
	geofenceState.points = nil
	if provider != nil {
		log.Printf("geofence: akun admin/karyawan dibatasi ke radius %.0f m dari titik gudang", radiusMeters)
	}
}

// InvalidateGeofenceCache memaksa daftar titik gudang dimuat ulang pada
// pemeriksaan berikutnya (dipanggil setelah gudang dibuat/diubah/dihapus).
func InvalidateGeofenceCache() {
	geofenceState.mu.Lock()
	geofenceState.loaded = false
	geofenceState.mu.Unlock()
}

func geofencePoints() ([]geofence.Point, float64, bool, error) {
	geofenceState.mu.Lock()
	defer geofenceState.mu.Unlock()
	if geofenceState.provider == nil {
		return nil, 0, false, nil
	}
	if !geofenceState.loaded || time.Since(geofenceState.loadedAt) > geofenceCacheTTL {
		points, err := geofenceState.provider()
		if err != nil {
			return nil, 0, true, err
		}
		geofenceState.points = points
		geofenceState.loadedAt = time.Now()
		geofenceState.loaded = true
	}
	return geofenceState.points, geofenceState.radius, true, nil
}

func geofenceRoleLabel(role string) string {
	if role == constant.RoleAdmin {
		return "admin"
	}
	return "karyawan"
}

// GeofenceAllowsRole memeriksa lokasi perangkat pada request ini.
//   - super_admin: tidak pernah dibatasi lokasi (bisa diakses dari mana saja).
//   - admin, karyawan, dan role lain (fail-closed): hanya boleh dari sekitar titik
//     koordinat gudang di Manajemen Gudang.
//   - Kalau belum ada gudang yang punya koordinat, akses ditolak (fail-closed)
//     sampai super admin mengisi titik koordinat gudang.
//
// Mengembalikan (diizinkan, pesan penolakan).
func GeofenceAllowsRole(c *fiber.Ctx, role string) (bool, string) {
	if role == constant.RoleSuperAdmin {
		return true, ""
	}
	points, radius, enabled, err := geofencePoints()
	if !enabled {
		return true, ""
	}
	if err != nil {
		log.Printf("geofence: gagal memuat titik gudang: %v", err)
		return false, "gagal memeriksa lokasi gudang, coba lagi sebentar"
	}

	loc, hasLoc := geofence.ParseLocation(c.Get(GeoLatHeader), c.Get(GeoLngHeader), c.Get(GeoAccHeader))
	res := geofence.Evaluate(points, radius, loc, hasLoc)
	switch res.Reason {
	case geofence.ReasonNoLocation:
		return false, GeofenceNoLocationMessage
	case geofence.ReasonNotConfigured:
		return false, "akses belum tersedia karena koordinat gudang belum ditentukan oleh super admin"
	case geofence.ReasonOutside:
		return false, fmt.Sprintf(
			"akun %s hanya bisa diakses dari titik koordinat gudang. Posisi Anda sekitar %.0f m dari %s (batas %.0f m)",
			geofenceRoleLabel(role), res.DistanceMeters, res.NearestName, radius,
		)
	default:
		return true, ""
	}
}

// AccessAllowsRole adalah pemeriksaan akses gabungan: daftar jaringan gudang
// (kalau diatur) DAN lokasi koordinat gudang. Dipakai di semua pintu masuk:
// login, register, penerbitan token, dan setiap request terautentikasi.
func AccessAllowsRole(c *fiber.Ctx, role string) (bool, string) {
	if !NetworkAllowsRole(role, c.IP()) {
		return false, NetworkMessageForRole(role)
	}
	return GeofenceAllowsRole(c, role)
}
