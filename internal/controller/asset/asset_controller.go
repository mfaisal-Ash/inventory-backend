package assetgudang

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

var (
	errParentNotFound     = errors.New("aset induk tidak ditemukan")
	errParentSelf         = errors.New("aset tidak bisa jadi induk untuk dirinya sendiri")
	errCoordinateRequired = errors.New("latitude dan longitude wajib diisi untuk jenis aset ini")
	errGudangKodeMissing  = errors.New("gudang belum punya kode — isi kode gudang dulu sebelum menambah aset")
)

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func parentHierarchyError(childJenis, parentJenis string) error {
	return fmt.Errorf("%s tidak bisa berinduk ke %s — cek urutan hierarki jaringan (OLT -> ODC -> ODP -> ONT)", childJenis, parentJenis)
}

func parentErrorResponse(c *fiber.Ctx, err error) error {
	if errors.Is(err, errParentNotFound) {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	return utils.Fail(c, fiber.StatusUnprocessableEntity, err.Error(), nil)
}

func (h *Controller) findValidParent(childJenis string, parentID uint) (*model.Asset, error) {
	parent, err := h.repo.FindByID(parentID)
	if err != nil {
		return nil, errParentNotFound
	}
	if !model.JenisIndukValid(childJenis, parent.JenisAset) {
		return nil, parentHierarchyError(childJenis, parent.JenisAset)
	}
	return parent, nil
}

func (h *Controller) assignAssetIdentifiers(req AssetRequest, gudang *model.Gudang, a *model.Asset) error {
	if !model.JenisAsetPunyaKoordinat(req.JenisAset) {
		nomor, err := h.repo.NextBANumber()
		if err != nil {
			return errors.New("gagal membuat kode BA")
		}
		a.KodeBA = fmt.Sprintf("BA-%04d", nomor)
		return nil
	}

	if req.Latitude == nil || req.Longitude == nil {
		return errCoordinateRequired
	}
	if gudang.Kode == "" {
		return errGudangKodeMissing
	}
	nomor, err := h.repo.NextRSDNumber(req.GudangID)
	if err != nil {
		return errors.New("gagal membuat label RSD")
	}
	a.LabelRSD = fmt.Sprintf("%s-RSD-%04d", gudang.Kode, nomor)
	a.Latitude = req.Latitude
	a.Longitude = req.Longitude
	return nil
}

func (h *Controller) logAssetChanges(c *fiber.Ctx, a *model.Asset, oldParentID *uint, oldLat, oldLng *float64) {
	if !samePtrUint(oldParentID, a.ParentAssetID) {
		h.logHistory(c, a.ID, "induk", ptrUintLabel(oldParentID), ptrUintLabel(a.ParentAssetID), "Aset induk (hierarki jaringan) diubah")
	}
	if !samePtrFloat(oldLat, a.Latitude) || !samePtrFloat(oldLng, a.Longitude) {
		h.logHistory(c, a.ID, "lokasi",
			fmt.Sprintf("%s, %s", ptrFloatLabel(oldLat), ptrFloatLabel(oldLng)),
			fmt.Sprintf("%s, %s", ptrFloatLabel(a.Latitude), ptrFloatLabel(a.Longitude)),
			"Titik koordinat lokasi diubah")
	}
}

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
			IPAddress: p.IPAddress, PingStatus: p.PingStatus,
			GudangID: p.GudangID, GudangNama: p.GudangNama, GudangKode: p.GudangKode, GudangTipe: p.GudangTipe,
			GudangLatitude: p.GudangLatitude, GudangLongitude: p.GudangLongitude,
			ParentAssetID: p.ParentAssetID, ParentLatitude: p.ParentLatitude, ParentLongitude: p.ParentLongitude,
			JumlahPort: p.JumlahPort, PortTerisi: p.PortTerisi,
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
	if req.ParentAssetID != nil {
		if _, perr := h.findValidParent(req.JenisAset, *req.ParentAssetID); perr != nil {
			return parentErrorResponse(c, perr)
		}
	}

	a := &model.Asset{
		Nama:          req.Nama,
		JenisAset:     req.JenisAset,
		GudangID:      req.GudangID,
		Keterangan:    req.Keterangan,
		IPAddress:     req.IPAddress,
		ParentAssetID: req.ParentAssetID,
		JumlahPort:    req.JumlahPort,
		Status:        "aktif",
		PingStatus:    "unknown",
	}

	if err := h.assignAssetIdentifiers(req, gudang, a); err != nil {
		if errors.Is(err, errCoordinateRequired) || errors.Is(err, errGudangKodeMissing) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, err.Error(), nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	if err := h.repo.Create(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat aset", nil)
	}
	h.logHistory(c, a.ID, "dibuat", "", a.Status, "Aset ditambahkan ke sistem")
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
	if req.ParentAssetID != nil {
		if *req.ParentAssetID == a.ID {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, errParentSelf.Error(), nil)
		}
		if _, perr := h.findValidParent(a.JenisAset, *req.ParentAssetID); perr != nil {
			return parentErrorResponse(c, perr)
		}
	}

	oldParentID := a.ParentAssetID
	oldLat, oldLng := a.Latitude, a.Longitude

	a.Nama = req.Nama
	a.Keterangan = req.Keterangan
	a.IPAddress = req.IPAddress
	a.ParentAssetID = req.ParentAssetID
	a.JumlahPort = req.JumlahPort
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

	h.logAssetChanges(c, a, oldParentID, oldLat, oldLng)
	return utils.OK(c, "aset berhasil diperbarui", a)
}

func samePtrUint(a, b *uint) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func samePtrFloat(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func ptrUintLabel(v *uint) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("#%d", *v)
}

func ptrFloatLabel(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.6f", *v)
}

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
	statusLama := a.Status
	a.Status = req.Status
	if err := h.repo.Update(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui status aset", nil)
	}
	if statusLama != a.Status {
		h.logHistory(c, a.ID, "status", statusLama, a.Status, "")
	}
	return utils.OK(c, "status aset berhasil diperbarui", a)
}

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
	g.Get("/:id/port", view, h.ListPorts)
	g.Put("/:id/port/:nomor", edit, h.SetPort)
	g.Delete("/:id/port/:nomor", edit, h.ClearPort)
	g.Get("/:id/riwayat", view, h.ListHistory)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
}
