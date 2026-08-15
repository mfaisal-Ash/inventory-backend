package asset

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/projsonal/gowms/internal/middleware"
	"github.com/projsonal/gowms/internal/model"
	assetRepo "github.com/projsonal/gowms/internal/repositories/asset"
	"github.com/projsonal/gowms/pkg/constant"
	"github.com/projsonal/gowms/pkg/utils"
)

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// data list
func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	gudangID, _ := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64)
	f := assetRepo.Filter{
		JenisAset: c.Query("jenis_aset", ""),
		GudangID:  uint(gudangID),
		Status:    c.Query("status", ""),
	}
	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar aset", nil)
	}
	return utils.OKWithMeta(c, "daftar aset berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

// ambil data
func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Data id aset tidak valid", nil)
	}
	a, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Data aset gudang tidak ditemukan", nil)
	}
	return utils.OK(c, "detail aset berhasil diambil", a)
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	tiang, _ := h.repo.CountByJenis(constant.JenisAsetTiang)
	odc, _ := h.repo.CountByJenis(constant.JenisAsetODC)
	olt, _ := h.repo.CountByJenis(constant.JenisAsetOLT)
	ont, _ := h.repo.CountByJenis(constant.JenisAsetONT)
	odp, _ := h.repo.CountByJenis(constant.JenisAsetODP)
	modem, _ := h.repo.CountByJenis(constant.JenisAsetModem)
	transportasi, _ := h.repo.CountByJenis(constant.JenisAsetTransportasi)
	return utils.OK(c, "ringkasan aset berhasil diambil", SummaryResponse{
		Tiang: tiang, Odc: odc, Olt: olt, Ont: ont, Odp: odp, Modem: modem, Transportasi: transportasi,
		Total: tiang + odc + olt + ont + odp + modem + transportasi,
	})
}

func (h *Controller) MapPoints(c *fiber.Ctx) error {
	f := assetRepo.Filter{
		JenisAset: c.Query("jenis_aset", ""),
		Status:    c.Query("status", ""),
	}
	if gudangID, err := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64); err == nil {
		f.GudangID = uint(gudangID)
	}
	tipeGudang := c.Query("tipe_gudang", "")
	if tipeGudang != "" && tipeGudang != constant.TipeGudangPusat && tipeGudang != constant.TipeGudangCabang {
		return utils.Fail(c, fiber.StatusBadRequest, "tipe_gudang harus 'pusat' atau 'cabang'", nil)
	}

	points, err := h.repo.ListForMap(f, tipeGudang)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil titik peta aset", nil)
	}

	out := make([]MapPoint, 0, len(points))
	for _, p := range points {
		out = append(out, MapPoint{
			ID: p.ID, Nama: p.Nama, JenisAset: p.JenisAset, LabelRSD: p.LabelRSD,
			Latitude: p.Latitude, Longitude: p.Longitude, Status: p.Status,
			GudangID: p.GudangID, GudangNama: p.GudangNama, GudangKode: p.GudangKode, GudangTipe: p.GudangTipe,
		})
	}
	return utils.OK(c, "titik peta aset berhasil diambil", out)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req AssetRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	gudang, err := h.gudangRepo.FindGudangByID(req.GudangID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}

	a := &model.Asset{
		Nama:       req.Nama,
		JenisAset:  req.JenisAset,
		GudangID:   req.GudangID,
		Keterangan: req.Keterangan,
		IPAddress:  req.IPAddress,
		Status:     "aktif",
		PingStatus: "unknown",
	}

	if model.JenisAsetPunyaKoordinat(req.JenisAset) {
		if req.Latitude == nil || req.Longitude == nil {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				"latitude dan longitude wajib diisi untuk jenis aset ini", nil)
		}
		if gudang.Kode == "" {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				"gudang belum punya kode — isi kode gudang dulu sebelum menambah aset", nil)
		}
		nomor, err := h.repo.NextRSDNumber(req.GudangID)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat label RSD", nil)
		}
		a.LabelRSD = fmt.Sprintf("%s-RSD-%04d", gudang.Kode, nomor)
		a.Latitude = req.Latitude
		a.Longitude = req.Longitude
	} else {
		nomor, err := h.repo.NextBANumber()
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat kode BA", nil)
		}
		a.KodeBA = fmt.Sprintf("BA-%04d", nomor)
	}

	if err := h.repo.Create(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat aset", nil)
	}
	created, _ := h.repo.FindByID(a.ID)
	return utils.Created(c, "aset berhasil dibuat", created)
}

func (h *Controller) Update(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Data id aset tidak valid", nil)
	}
	a, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Data aset gudang tidak ditemukan", nil)
	}

	var req AssetRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	a.Nama = req.Nama
	a.Keterangan = req.Keterangan
	a.IPAddress = req.IPAddress
	if model.JenisAsetPunyaKoordinat(a.JenisAset) {
		if req.Latitude == nil || req.Longitude == nil {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				"latitude dan longitude wajib diisi untuk jenis aset ini", nil)
		}
		a.Latitude = req.Latitude
		a.Longitude = req.Longitude
	}
	if err := h.repo.Update(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui aset", nil)
	}
	return utils.OK(c, "aset berhasil diperbarui", a)
}

// func up status
func (h *Controller) UpdateStatus(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id aset tidak valid", nil)
	}
	a, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "aset tidak ditemukan", nil)
	}

	var req UpdateStatusRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	a.Status = req.Status
	if err := h.repo.Update(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui status aset", nil)
	}
	return utils.OK(c, "status aset berhasil diperbarui", a)
}

// Function Delete
func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id aset tidak valid", nil)
	}
	if _, err := h.repo.FindByID(id); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "aset tidak ditemukan", nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus aset", nil)
	}
	return utils.OK(c, "aset berhasil dihapus", nil)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/aset", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/map", view, h.MapPoints)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, onlyStaff, h.Create)
	g.Put("/:id", edit, onlyStaff, h.Update)
	g.Patch("/:id/status", edit, h.UpdateStatus)
	g.Post("/ping", edit, h.PingAll)
	g.Post("/:id/ping", edit, h.Ping)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
}
