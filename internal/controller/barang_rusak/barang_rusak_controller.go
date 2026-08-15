package barang_rusak

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	notification "github.com/projsonal/gowms/internal/controller/notification"
	"github.com/projsonal/gowms/internal/middleware"
	"github.com/projsonal/gowms/internal/model"
	barangRusakRepo "github.com/projsonal/gowms/internal/repositories/barang_rusak"
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

// List GET /barang-rusak?status=&search=
func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	f := barangRusakRepo.Filter{Status: c.Query("status", "")}

	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar barang rusak", nil)
	}
	return utils.OKWithMeta(c, "daftar barang rusak berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

// Detail GET /barang-rusak/:id
func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrIDInvalid, nil)
	}
	b, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrBarangRusakTidakDitemukan, nil)
	}
	return utils.OK(c, "detail barang rusak berhasil diambil", b)
}

// Create POST /barang-rusak — laporkan barang rusak. Foto (jika ada)
// diunggah terpisah lewat UploadFoto, mengikuti pola upload avatar di
// modul users (JSON dulu untuk data, baru multipart untuk file).
func (h *Controller) Create(c *fiber.Ctx) error {
	var req BarangRusakRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	if req.BarangID != nil {
		if _, err := h.barangRepo.FindByID(*req.BarangID); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "barang tidak ditemukan", nil)
		}
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	b := &model.BarangRusak{
		BarangID:       req.BarangID,
		LabelBarang:    req.LabelBarang,
		NamaBarang:     req.NamaBarang,
		Keterangan:     req.Keterangan,
		JenisBarang:    req.JenisBarang,
		Status:         constant.StatusBarangRusakPengecekan,
		DilaporkanOleh: userID,
	}
	if err := h.repo.Create(b); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat laporan barang rusak", nil)
	}

	notification.Notify(h.notifRepo, "barang_rusak", "Laporan Barang Rusak Baru",
		b.NamaBarang+" ("+b.LabelBarang+") dilaporkan rusak dan menunggu pengecekan.",
		"/barang-rusak", nil, constant.RoleAdmin)

	created, _ := h.repo.FindByID(b.ID)
	return utils.Created(c, "laporan barang rusak berhasil dibuat", created)
}

// UploadFoto POST /barang-rusak/:id/foto (multipart/form-data, field "foto")
// — simpan foto bukti kerusakan di disk lokal (StorageConfig.Path/barang-rusak/),
// dibatasi 2MB & hanya jpg/jpeg/png, mengikuti pola UploadAvatar di modul users.
func (h *Controller) UploadFoto(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrIDInvalid, nil)
	}
	b, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrBarangRusakTidakDitemukan, nil)
	}

	file, err := c.FormFile("foto")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "file foto tidak ditemukan (field: foto)", nil)
	}
	const maxFotoSize = 2 * 1024 * 1024 // 2MB
	if file.Size > maxFotoSize {
		return utils.Fail(c, fiber.StatusBadRequest, "ukuran file maksimal 2MB", nil)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return utils.Fail(c, fiber.StatusBadRequest, "format file harus jpg, jpeg, atau png", nil)
	}

	fotoDir := filepath.Join(h.storagePath, "barang-rusak")
	if err := os.MkdirAll(fotoDir, 0o755); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyiapkan folder upload", nil)
	}
	filename := fmt.Sprintf("barang-rusak-%d-%d%s", b.ID, time.Now().UnixNano(), ext)
	if err := c.SaveFile(file, filepath.Join(fotoDir, filename)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan file", nil)
	}

	b.FotoURL = "/uploads/barang-rusak/" + filename
	if err := h.repo.Update(b); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui data barang rusak", nil)
	}
	return utils.OK(c, "foto barang rusak berhasil diunggah", b)
}

// UpdateStatus PATCH /barang-rusak/:id/status — admin/super_admin
// menindaklanjuti hasil pengecekan barang rusak.
func (h *Controller) UpdateStatus(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrIDInvalid, nil)
	}
	b, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrBarangRusakTidakDitemukan, nil)
	}

	var req UpdateStatusRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	now := time.Now()
	b.Status = req.Status
	if req.Keterangan != "" {
		b.Keterangan = req.Keterangan
	}
	b.DicekOleh = &userID
	b.DicekPada = &now

	if err := h.repo.Update(b); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui status barang rusak", nil)
	}

	if b.DilaporkanOleh != 0 {
		reporterID := b.DilaporkanOleh
		notification.Notify(h.notifRepo, "barang_rusak", "Status Laporan Barang Rusak Diperbarui",
			b.NamaBarang+" ("+b.LabelBarang+") kini berstatus \""+b.Status+"\".",
			"/barang-rusak", &reporterID, "")
	}

	return utils.OK(c, "status barang rusak berhasil diperbarui", b)
}

// Delete DELETE /barang-rusak/:id (soft-delete, lihat catatan di
// repositories/barang_rusak Delete() — bisa dipulihkan lewat Tempat Sampah).
func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrIDInvalid, nil)
	}
	if _, err := h.repo.FindByID(id); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrBarangRusakTidakDitemukan, nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus data barang rusak", nil)
	}
	return utils.OK(c, "data barang rusak berhasil dihapus", nil)
}

// Summary GET /barang-rusak/summary
func (h *Controller) Summary(c *fiber.Ctx) error {
	pengecekan, err1 := h.repo.CountByStatus(constant.StatusBarangRusakPengecekan)
	diperbaiki, err2 := h.repo.CountByStatus(constant.StatusBarangRusakDiperbaiki)
	retur, err3 := h.repo.CountByStatus(constant.StatusRetur)
	dibuang, err4 := h.repo.CountByStatus(constant.StatusBarangRusakDibuang)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil ringkasan barang rusak", nil)
	}
	return utils.OK(c, "ringkasan barang rusak berhasil diambil", SummaryResponse{
		Pengecekan: pengecekan, Diperbaiki: diperbaiki, Retur: retur, Dibuang: dibuang,
		Total: pengecekan + diperbaiki + retur + dibuang,
	})
}

// RegisterRoutes mendaftarkan endpoint modul "Barang Rusak".
func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/barang-rusak", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, h.Create)
	g.Post("/:id/foto", tambah, h.UploadFoto)
	g.Patch("/:id/status", edit, onlyStaff, h.UpdateStatus)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
}
