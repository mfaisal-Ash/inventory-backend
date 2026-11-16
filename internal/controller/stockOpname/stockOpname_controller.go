package stock_opname

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	notification "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	soRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const Module = constant.ModuleStockOpname

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func (h *Controller) validateItems(items []ItemRequest) error {
	seen := make(map[uint]struct{}, len(items))
	for _, it := range items {
		if _, exists := seen[it.BarangID]; exists {
			return fmt.Errorf("barang id %d tidak boleh dicatat lebih dari satu kali", it.BarangID)
		}
		seen[it.BarangID] = struct{}{}
		if _, err := h.barangRepo.FindByID(it.BarangID); err != nil {
			return fmt.Errorf("barang id %d tidak ditemukan", it.BarangID)
		}
	}
	return nil
}

func toItemInputs(items []ItemRequest) []soRepo.ItemInput {
	out := make([]soRepo.ItemInput, 0, len(items))
	for _, it := range items {
		out = append(out, soRepo.ItemInput{
			BarangID: it.BarangID, StokFisik: it.StokFisik, Catatan: it.Catatan,
		})
	}
	return out
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	gudangID, _ := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64)
	f := soRepo.Filter{Status: c.Query("status", ""), GudangID: uint(gudangID)}

	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar stock opname", nil)
	}
	return utils.OKWithMeta(c, "daftar stock opname berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	so, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrSOTidakDitemukan, nil)
	}
	return utils.OK(c, "detail stock opname berhasil diambil", so)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req SORequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	tanggal, err := parseTanggalHarian(req.Tanggal)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "format tanggal tidak valid (YYYY-MM-DD)", nil)
	}
	if _, err := h.gudangRepo.FindGudangByID(req.GudangID); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}
	if err := h.validateItems(req.Items); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	nomor, err := h.repo.NextNomor()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat nomor opname", nil)
	}
	so := &model.StockOpname{
		NomorOpname: nomor,
		GudangID:    req.GudangID,
		Status:      constant.StatusSODraft,
		Tanggal:     tanggal,
		Catatan:     req.Catatan,
	}
	if err := h.repo.Create(so, toItemInputs(req.Items)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat dokumen stock opname", nil)
	}
	notification.Notify(h.notifRepo, "opname",
		"Stock Opname Baru",
		so.NomorOpname+" dilakukan.",
		"/home/inventory-management", nil, "all")
	return utils.Created(c, "dokumen stock opname berhasil dibuat", so)
}

func (h *Controller) requireDraft(id uint) (*model.StockOpname, error) {
	so, err := h.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if so.Status != constant.StatusSODraft && so.Status != constant.StatusSODitolak {
		return nil, errors.New(constant.ErrSOBukanDraft)
	}
	return so, nil
}

func (h *Controller) Update(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	so, err := h.requireDraft(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}

	var req SORequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	tanggal, err := parseTanggalHarian(req.Tanggal)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "format tanggal tidak valid (YYYY-MM-DD)", nil)
	}
	if _, err := h.gudangRepo.FindGudangByID(req.GudangID); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}
	if err := h.validateItems(req.Items); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	so.GudangID = req.GudangID
	so.Tanggal = tanggal
	so.Catatan = req.Catatan
	if err := h.repo.Update(so, toItemInputs(req.Items)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui dokumen stock opname", nil)
	}
	return utils.OK(c, "dokumen stock opname berhasil diperbarui", so)
}

func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	if _, err := h.requireDraft(id); err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus dokumen stock opname", nil)
	}
	return utils.OK(c, "dokumen stock opname berhasil dihapus", nil)
}

func (h *Controller) MulaiPenghitungan(c *fiber.Ctx) error {
	return h.transition(c, h.repo.MulaiPenghitungan)
}
func (h *Controller) KirimPemeriksaan(c *fiber.Ctx) error {
	return h.transition(c, h.repo.KirimPemeriksaan)
}
func (h *Controller) Setujui(c *fiber.Ctx) error {
	return h.transition(c, h.repo.Setujui)
}
func (h *Controller) Tolak(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	var body struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil || body.Reason == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "alasan penolakan wajib diisi", nil)
	}
	so, err := h.repo.Tolak(id, userID, body.Reason)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	return utils.OK(c, "stock opname ditolak", so)
}
func (h *Controller) Riwayat(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	rows, err := h.repo.Riwayat(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat stock opname", nil)
	}
	return utils.OK(c, "riwayat stock opname berhasil diambil", rows)
}
func (h *Controller) transition(c *fiber.Ctx, fn func(uint, uint) (*model.StockOpname, error)) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	so, err := fn(id, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	return utils.OK(c, "status stock opname berhasil diperbarui", so)
}

func (h *Controller) Complete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)

	so, err := h.repo.Complete(id, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	h.notifyLowStock(so.Items)
	return utils.OK(c, "stock opname berhasil diselesaikan, selisih telah diterapkan ke stok", so)
}

func (h *Controller) notifyLowStock(items []model.StockOpnameItem) {
	for _, item := range items {
		if item.Barang == nil || item.Barang.StokMinimum <= 0 || item.Selisih == 0 {
			continue
		}
		justCrossed := item.StokSistem > item.Barang.StokMinimum && item.StokFisik <= item.Barang.StokMinimum
		if !justCrossed {
			continue
		}
		notification.Notify(h.notifRepo, "stok_menipis",
			"Stok Menipis",
			item.Barang.Nama+" tersisa "+strconv.Itoa(item.StokFisik)+" (ambang minimum "+strconv.Itoa(item.Barang.StokMinimum)+").",
			"/home/kelola-barang", nil, "all")
	}
}

func (h *Controller) Batalkan(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id stock opname tidak valid", nil)
	}
	so, err := h.repo.Batalkan(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	return utils.OK(c, "dokumen stock opname berhasil dibatalkan", so)
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	total, err := h.repo.CountByStatus("")
	draft, err2 := h.repo.CountByStatus(constant.StatusSODraft)
	selesai, err3 := h.repo.CountByStatus(constant.StatusSOSelesai)
	if err != nil || err2 != nil || err3 != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil ringkasan", nil)
	}
	return utils.OK(c, "ringkasan stock opname berhasil diambil", SummaryResponse{
		TotalDokumen: total, Draft: draft, Selesai: selesai,
	})
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/stock-opname", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	hanyaStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)
	hanyaSuperAdmin := middleware.RequireRole(constant.RoleSuperAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, h.Create)
	g.Put("/:id", edit, h.Update)
	g.Delete("/:id", hanyaStaff, edit, h.Delete)
	g.Patch("/:id/mulai-penghitungan", edit, h.MulaiPenghitungan)
	g.Patch("/:id/kirim-pemeriksaan", edit, h.KirimPemeriksaan)
	g.Patch("/:id/setujui", hanyaStaff, edit, h.Setujui)
	g.Patch("/:id/tolak", hanyaStaff, edit, h.Tolak)
	g.Get("/:id/riwayat", view, h.Riwayat)
	g.Patch("/:id/selesai", hanyaSuperAdmin, edit, h.Complete)
	g.Patch("/:id/batalkan", edit, h.Batalkan)
}
