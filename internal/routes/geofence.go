package routes

import (
	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geofence"
)

// gudangGeofencePoints mengambil titik koordinat semua gudang aktif yang sudah
// punya latitude & longitude (isian di Manajemen Gudang). Gudang yang sudah
// dihapus (soft delete) otomatis tidak ikut.
func gudangGeofencePoints(db *gorm.DB) middleware.GeofenceProvider {
	return func() ([]geofence.Point, error) {
		var rows []model.Gudang
		if err := db.
			Select("id", "nama", "latitude", "longitude").
			Where("latitude IS NOT NULL AND longitude IS NOT NULL").
			Find(&rows).Error; err != nil {
			return nil, err
		}
		points := make([]geofence.Point, 0, len(rows))
		for _, g := range rows {
			if g.Latitude == nil || g.Longitude == nil || !geofence.ValidLatLng(*g.Latitude, *g.Longitude) {
				continue
			}
			points = append(points, geofence.Point{Name: g.Nama, Lat: *g.Latitude, Lng: *g.Longitude})
		}
		return points, nil
	}
}
