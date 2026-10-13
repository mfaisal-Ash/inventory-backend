package kabel

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	kabelRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/kabel"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func parseUintParam(c *fiber.Ctx, name string) (uint, error) {
	id, err := strconv.ParseUint(c.Params(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func parseUintQuery(c *fiber.Ctx, name string) uint {
	v, _ := strconv.ParseUint(c.Query(name, "0"), 10, 64)
	return uint(v)
}

// failRepo: record tidak ada -> 404, pelanggaran aturan bisnis -> 400 dengan
// pesan aslinya, selain itu 500 tanpa membocorkan detail error DB.
func failRepo(c *fiber.Ctx, err error, fallback string) error {
	var be *kabelRepo.BizError
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrKabelTidakDitemukan, nil)
	case errors.As(err, &be):
		return utils.Fail(c, fiber.StatusBadRequest, be.Msg, nil)
	default:
		return utils.Fail(c, fiber.StatusInternalServerError, fallback, nil)
	}
}

func filterFromQuery(c *fiber.Ctx) kabelRepo.Filter {
	return kabelRepo.Filter{
		GudangID: parseUintQuery(c, "gudang_id"),
		BarangID: parseUintQuery(c, "barang_id"),
		Status:   c.Query("status", ""),
	}
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	list, total, err := h.repo.List(p, filterFromQuery(c))
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar gulungan kabel", nil)
	}
	return utils.OKWithMeta(c, "daftar gulungan kabel berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	s, err := h.repo.Summary(filterFromQuery(c))
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil ringkasan kabel", nil)
	}
	return utils.OK(c, "ringkasan kabel berhasil diambil", s)
}

func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gulungan tidak valid", nil)
	}
	g, err := h.repo.FindByID(id)
	if err != nil {
		return failRepo(c, err, "gagal mengambil detail gulungan")
	}
	return utils.OK(c, "detail gulungan berhasil diambil", g)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req CreateRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	g := &model.KabelGulungan{
		BarangID:    req.BarangID,
		GudangID:    req.GudangID,
		PanjangAwal: req.PanjangAwal,
		Catatan:     strings.TrimSpace(req.Catatan),
		DibuatOleh:  userID,
	}
	if err := h.repo.Create(g); err != nil {
		return failRepo(c, err, "gagal mendaftarkan gulungan kabel")
	}
	created, err := h.repo.FindByID(g.ID)
	if err != nil {
		return failRepo(c, err, "gulungan dibuat, tetapi gagal memuat ulang datanya")
	}
	return utils.Created(c, "gulungan kabel berhasil didaftarkan", created)
}

func (h *Controller) Pakai(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gulungan tidak valid", nil)
	}
	var req PakaiRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	tanggal := time.Now()
	if t := strings.TrimSpace(req.Tanggal); t != "" {
		parsed, perr := time.ParseInLocation("2006-01-02", t, time.Local)
		if perr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "format tanggal harus YYYY-MM-DD", nil)
		}
		tanggal = parsed
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	g, err := h.repo.Pakai(id, kabelRepo.PemakaianInput{
		Jumlah:    req.Jumlah,
		Tanggal:   tanggal,
		Klien:     strings.TrimSpace(req.Klien),
		Keperluan: strings.TrimSpace(req.Keperluan),
		UserID:    userID,
	})
	if err != nil {
		return failRepo(c, err, "gagal mencatat pemakaian kabel")
	}
	return utils.OK(c, "pemakaian kabel dicatat, stok gudang ikut berkurang", g)
}

func (h *Controller) BatalPakai(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gulungan tidak valid", nil)
	}
	pid, err := parseUintParam(c, "pemakaianId")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id pemakaian tidak valid", nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	g, err := h.repo.BatalPakai(id, pid, userID)
	if err != nil {
		return failRepo(c, err, "gagal membatalkan pemakaian")
	}
	return utils.OK(c, "pemakaian dibatalkan, meter dikembalikan ke gulungan dan stok gudang", g)
}

func (h *Controller) Hapus(c *fiber.Ctx) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id gulungan tidak valid", nil)
	}
	if err := h.repo.Hapus(id); err != nil {
		return failRepo(c, err, "gagal menghapus gulungan")
	}
	return utils.OK(c, "gulungan kabel dihapus", nil)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/kabel", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, onlyStaff, h.Create)
	g.Post("/:id/pemakaian", tambah, h.Pakai)
	g.Patch("/:id/pemakaian/:pemakaianId/batal", edit, onlyStaff, h.BatalPakai)
	g.Delete("/:id", edit, onlyStaff, h.Hapus)
}
