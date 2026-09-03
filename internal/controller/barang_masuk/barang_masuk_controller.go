package barang_masuk

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	notification "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	bmRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const Module = constant.ModuleBarangMasuk

const msgIdBM = "id barang masuk tidak valid"

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func (h *Controller) validateItems(req BMRequest) error {
	return h.validateItemRequests(req.Items)
}

func (h *Controller) validateItemRequests(items []ItemRequest) error {
	for _, it := range items {
		if err := h.validateItem(it); err != nil {
			return err
		}
	}
	return nil
}

func (h *Controller) validateItem(it ItemRequest) error {
	if _, err := h.barangRepo.FindByID(it.BarangID); err != nil {
		return fmt.Errorf("barang id %d tidak ditemukan", it.BarangID)
	}
	return nil
}

func toItemModels(items []ItemRequest) []model.BarangMasukItem {
	out := make([]model.BarangMasukItem, 0, len(items))
	for _, it := range items {
		out = append(out, model.BarangMasukItem{
			BarangID: it.BarangID, Qty: it.Qty, HargaSatuan: it.HargaSatuan,
		})
	}
	return out
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	gudangID, _ := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64)
	kategoriID, _ := strconv.ParseUint(c.Query("kategori_id", "0"), 10, 64)
	barangID, _ := strconv.ParseUint(c.Query("barang_id", "0"), 10, 64)
	f := bmRepo.Filter{
		Status:     c.Query("status", ""),
		GudangID:   uint(gudangID),
		KategoriID: uint(kategoriID),
		BarangID:   uint(barangID),
		Merek:      c.Query("merek", ""),
		Tipe:       c.Query("tipe", ""),
	}

	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar barang masuk", nil)
	}
	return utils.OKWithMeta(c, "daftar barang masuk berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}
	bm, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrBMTidakDitemukan, nil)
	}
	return utils.OK(c, "detail barang masuk berhasil diambil", bm)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req BMRequest
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
	if err := h.validateItems(req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	nomor, err := h.repo.NextNomor()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat nomor penerimaan", nil)
	}
	bm := &model.BarangMasuk{
		NomorPenerimaan: nomor,
		GudangID:        req.GudangID,
		Status:          constant.StatusBMDraft,
		Tanggal:         tanggal,
		Catatan:         req.Catatan,
		Items:           toItemModels(req.Items),
	}
	if err := h.repo.Create(bm); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat dokumen barang masuk", nil)
	}
	notification.Notify(h.notifRepo, "in",
		"Barang Masuk Baru",
		bm.NomorPenerimaan+" ditambahkan.",
		"/home/barang-masuk", nil, "all")
	return utils.Created(c, "dokumen barang masuk berhasil dibuat", bm)
}

func (h *Controller) requireDraft(id uint) (*model.BarangMasuk, error) {
	bm, err := h.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if bm.Status != constant.StatusBMDraft {
		return nil, errors.New(constant.ErrBMBukanDraft)
	}
	return bm, nil
}

func (h *Controller) Update(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}
	bm, err := h.requireDraft(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	if bm.IsProtected {
		return utils.Fail(c, fiber.StatusForbidden,
			"data ini dikunci (Protect) oleh super admin — buka kuncinya dulu sebelum diubah", nil)
	}

	var req BMRequest
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
	if err := h.validateItems(req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	bm.GudangID = req.GudangID
	bm.Tanggal = tanggal
	bm.Catatan = req.Catatan
	if err := h.repo.Update(bm, toItemModels(req.Items)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui dokumen barang masuk", nil)
	}
	return utils.OK(c, "dokumen barang masuk berhasil diperbarui", bm)
}

func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}
	bm, err := h.requireDraft(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	if bm.IsProtected {
		return utils.Fail(c, fiber.StatusForbidden,
			"data ini dikunci (Protect) oleh super admin — buka kuncinya dulu sebelum dihapus", nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus dokumen barang masuk", nil)
	}
	return utils.OK(c, "dokumen barang masuk berhasil dihapus", nil)
}

func (h *Controller) Protect(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}
	var req ProtectRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	if _, err := h.repo.FindByID(id); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "dokumen barang masuk tidak ditemukan", nil)
	}
	if err := h.repo.SetProtected(id, *req.IsProtected); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengubah status proteksi", nil)
	}
	bm, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil dokumen setelah diperbarui", nil)
	}
	return utils.OK(c, "status proteksi berhasil diubah", bm)
}

func (h *Controller) Complete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)

	// Complete/Batalkan tadinya tidak pernah cek IsProtected sama sekali
	// (beda dengan Update/Delete di atas) — padahal Complete tetap mengubah
	// stok & status dokumen, jadi dokumen yang sudah di-Protect super admin
	// seharusnya juga tidak bisa diselesaikan/dibatalkan lewat sini.
	existing, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "dokumen barang masuk tidak ditemukan", nil)
	}
	if existing.IsProtected {
		return utils.Fail(c, fiber.StatusForbidden,
			"data ini dikunci (Protect) oleh super admin — buka kuncinya dulu sebelum diselesaikan", nil)
	}

	var req CompleteBMRequest
	_ = c.BodyParser(&req)
	serials := make(map[uint][]string, len(req.Items))
	for _, it := range req.Items {
		serials[it.BarangMasukItemID] = it.SerialNumbers
	}

	bm, err := h.repo.Complete(id, userID, serials)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	return utils.OK(c, "barang masuk berhasil diselesaikan, stok & rak telah diperbarui", bm)
}

func (h *Controller) Batalkan(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIdBM, nil)
	}

	existing, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "dokumen barang masuk tidak ditemukan", nil)
	}
	if existing.IsProtected {
		return utils.Fail(c, fiber.StatusForbidden,
			"data ini dikunci (Protect) oleh super admin — buka kuncinya dulu sebelum dibatalkan", nil)
	}

	bm, err := h.repo.Batalkan(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	return utils.OK(c, "dokumen barang masuk berhasil dibatalkan", bm)
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	total, err := h.repo.CountByStatus("")
	draft, err2 := h.repo.CountByStatus(constant.StatusBMDraft)
	selesai, err3 := h.repo.CountByStatus(constant.StatusBMSelesai)
	if err != nil || err2 != nil || err3 != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil ringkasan", nil)
	}
	return utils.OK(c, "ringkasan barang masuk berhasil diambil", SummaryResponse{
		TotalDokumen: total, Draft: draft, Selesai: selesai,
	})
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/barang-masuk", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)
	onlySuperAdmin := middleware.RequireRole(constant.RoleSuperAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, h.Create)
	g.Put("/:id", edit, h.Update)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
	g.Patch("/:id/selesai", edit, h.Complete)
	g.Patch("/:id/batalkan", edit, h.Batalkan)
	g.Patch("/:id/protect", onlySuperAdmin, h.Protect)
}
