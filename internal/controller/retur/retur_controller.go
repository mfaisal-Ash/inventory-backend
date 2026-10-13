package retur

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	notification "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	returRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/retur"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

// failRepo memetakan error repository: record tidak ada -> 404, pelanggaran
// aturan bisnis (mis. "unit pengganti harus Normal") -> 400 dengan pesan
// aslinya, selain itu -> 500 tanpa membocorkan detail error DB.
func failRepo(c *fiber.Ctx, err error, fallback string) error {
	var be *returRepo.BizError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrReturTidakDitemukan, nil)
	case errors.As(err, &be):
		return utils.Fail(c, fiber.StatusBadRequest, be.Msg, nil)
	default:
		return utils.Fail(c, fiber.StatusInternalServerError, fallback, nil)
	}
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	barangID, _ := strconv.ParseUint(c.Query("barang_id", "0"), 10, 64)
	list, total, err := h.repo.List(p, returRepo.Filter{Status: c.Query("status", ""), BarangID: uint(barangID)})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar retur", nil)
	}
	return utils.OKWithMeta(c, "daftar retur berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id retur tidak valid", nil)
	}
	rt, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrReturTidakDitemukan, nil)
	}
	return utils.OK(c, "detail retur berhasil diambil", rt)
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	var s SummaryResponse
	s.Dilaporkan, _ = h.repo.CountByStatus(constant.StatusReturDilaporkan)
	s.PerluGanti, _ = h.repo.CountByStatus(constant.StatusReturPerluGanti)
	s.TidakPerluGanti, _ = h.repo.CountByStatus(constant.StatusReturTidakPerlu)
	s.Selesai, _ = h.repo.CountByStatus(constant.StatusReturSelesai)
	s.Total = s.Dilaporkan + s.PerluGanti + s.TidakPerluGanti + s.Selesai
	return utils.OK(c, "ringkasan retur berhasil diambil", s)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req CreateRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	brg, err := h.barangRepo.FindByID(req.BarangID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "barang tidak ditemukan", nil)
	}
	if !brg.IsSerialized {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrSerialBarangBukanSerial, nil)
	}
	if _, err := h.gudangRepo.FindGudangByID(req.GudangID); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	rt := &model.ReturPenggantian{
		BarangID:       req.BarangID,
		GudangID:       req.GudangID,
		Klien:          strings.TrimSpace(req.Klien),
		AlamatKlien:    strings.TrimSpace(req.AlamatKlien),
		SerialLama:     strings.TrimSpace(req.SerialLama),
		Keluhan:        strings.TrimSpace(req.Keluhan),
		DilaporkanOleh: userID,
	}
	if err := h.repo.Create(rt); err != nil {
		return failRepo(c, err, "gagal membuat laporan retur")
	}
	msg := brg.Nama + " (" + rt.SerialLama + ") di klien " + rt.Klien + " dilaporkan rusak, menunggu pengecekan teknisi."
	notification.Notify(h.notifRepo, "retur", "Laporan Retur Baru", msg, "/retur", nil, constant.RoleAdmin)
	notification.Notify(h.notifRepo, "retur", "Laporan Retur Baru", msg, "/retur", nil, constant.RoleSuperAdmin)

	created, _ := h.repo.FindByID(rt.ID)
	return utils.Created(c, "laporan retur berhasil dibuat, menunggu pengecekan teknisi", created)
}

func (h *Controller) Pengecekan(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id retur tidak valid", nil)
	}
	var req PengecekanRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	rt, err := h.repo.Pengecekan(id, req.Hasil, strings.TrimSpace(req.Catatan), userID)
	if err != nil {
		return failRepo(c, err, "gagal menyimpan hasil pengecekan")
	}
	if rt.Status == constant.StatusReturPerluGanti {
		msg := rt.NomorRetur + ": modem di klien " + rt.Klien + " perlu diganti. Pilih unit pengganti berstatus Normal."
		notification.Notify(h.notifRepo, "retur", "Retur Perlu Penggantian", msg, "/retur", nil, constant.RoleAdmin)
		notification.Notify(h.notifRepo, "retur", "Retur Perlu Penggantian", msg, "/retur", nil, constant.RoleSuperAdmin)
		return utils.OK(c, "hasil pengecekan disimpan: perlu diganti", rt)
	}
	return utils.OK(c, "hasil pengecekan disimpan: tidak perlu diganti, retur ditutup", rt)
}

func (h *Controller) Ganti(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id retur tidak valid", nil)
	}
	var req GantiRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	rt, err := h.repo.Ganti(id, req.SerialBaru, userID)
	if err != nil {
		return failRepo(c, err, "gagal memproses penggantian")
	}
	return utils.OK(c, "penggantian selesai: unit baru keluar ke klien, unit lama ditarik ke gudang sebagai rusak", rt)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/retur", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, h.Create)
	// Pengecekan teknisi: cukup izin edit (teknisi tidak harus admin).
	g.Patch("/:id/pengecekan", edit, h.Pengecekan)
	// Penggantian menggerakkan stok & status serial, jadi khusus staf.
	g.Patch("/:id/ganti", edit, onlyStaff, h.Ganti)
}
