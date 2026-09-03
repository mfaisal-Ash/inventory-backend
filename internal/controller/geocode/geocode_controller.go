// Package geocode adalah controller tipis di atas pkg/geocoding — dipanggil
// dari menu Manajemen Gudang ("Cari Koordinat dari Alamat") dan Manajemen
// Aset Barang (yang memakai alamat gudang terpilih) supaya browser tidak
// lagi memanggil Nominatim langsung (lihat komentar package di
// pkg/geocoding untuk alasannya).
package geocode

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geocoding"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	svc    *geocoding.Service
	jwtSvc *utils.JWTService
}

func New(svc *geocoding.Service, jwtSvc *utils.JWTService) *Controller {
	return &Controller{svc: svc, jwtSvc: jwtSvc}
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	router.Get("/geocode", middleware.JWTAuth(h.jwtSvc), h.Search)
}

type geocodeResponse struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	DisplayName string  `json:"display_name"`
	// Precision: "street" (level jalan/nomor rumah — paling akurat),
	// "area" (level kelurahan/desa/kecamatan), "region" (cuma level
	// kabupaten/kota/provinsi — sebaiknya digeser manual di peta), atau
	// "unknown".
	Precision string `json:"precision"`
}

// Search menerima ?address=... dan mengembalikan koordinat hasil geocoding
// beserta tingkat presisinya, supaya frontend bisa kasih tahu pengguna
// dengan jujur seberapa dekat titik yang ditemukan (bukan cuma "ditemukan"
// generik seperti sebelumnya).
func (h *Controller) Search(c *fiber.Ctx) error {
	address := strings.TrimSpace(c.Query("address"))
	if address == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "parameter address wajib diisi", nil)
	}

	result, err := h.svc.Geocode(c.Context(), address)
	if err != nil {
		return utils.Fail(
			c,
			fiber.StatusNotFound,
			"Alamat ini tidak ditemukan di OpenStreetMap — coba tulis dengan istilah lain, atau isi koordinat manual (salin dari Google Maps: klik kanan titik lokasi → salin koordinat).",
			nil,
		)
	}

	return utils.OK(c, "koordinat berhasil ditemukan", geocodeResponse{
		Latitude:    result.Latitude,
		Longitude:   result.Longitude,
		DisplayName: result.DisplayName,
		Precision:   string(result.Precision),
	})
}
