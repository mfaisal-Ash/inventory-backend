package pengajuan_template

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	pengajuanTemplateRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_template"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// maxTemplateFileSize dijaga di bawah BodyLimit global app (4MB, lihat
// routes/router.go) supaya request multipart tidak ditolak duluan oleh
// Fiber sebelum sempat divalidasi di sini.
const maxTemplateFileSize = 3 * 1024 * 1024

// allowedTemplateExt: hanya format dokumen yang lazim dipakai untuk formulir
// kantor (bukan gambar — itu sudah dipakai untuk foto bukti Barang Rusak).
var allowedTemplateExt = map[string]string{
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
}

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	f := pengajuanTemplateRepo.Filter{OnlyActive: c.Query("only_active", "") == "true"}
	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar template pengajuan", nil)
	}
	return utils.OKWithMeta(c, "daftar template pengajuan berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

// Create mengunggah satu formulir kosong (docx/pdf) baru — dipakai admin/
// super admin untuk menambah pilihan template yang bisa dipilih siapa pun
// saat membuat pengajuan jenis "template" (lihat pengajuan_barang.validateJenisSpecific).
func (h *Controller) Create(c *fiber.Ctx) error {
	nama := strings.TrimSpace(c.FormValue("nama"))
	if nama == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "nama template wajib diisi", nil)
	}
	if len(nama) > 150 {
		return utils.Fail(c, fiber.StatusBadRequest, "nama template maksimal 150 karakter", nil)
	}
	deskripsi := strings.TrimSpace(c.FormValue("deskripsi"))
	if len(deskripsi) > 255 {
		return utils.Fail(c, fiber.StatusBadRequest, "deskripsi maksimal 255 karakter", nil)
	}

	file, err := c.FormFile("file")
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "file template tidak ditemukan (field: file)", nil)
	}
	if file.Size > maxTemplateFileSize {
		return utils.Fail(c, fiber.StatusBadRequest, "ukuran file maksimal 3MB", nil)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	contentType, ok := allowedTemplateExt[ext]
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "format file harus pdf, doc, atau docx", nil)
	}

	opened, err := file.Open()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membaca file template", nil)
	}
	defer opened.Close()
	data, err := io.ReadAll(opened)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membaca file template", nil)
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	template := &model.PengajuanTemplate{
		Nama:            nama,
		Deskripsi:       deskripsi,
		IsActive:        true,
		FileName:        file.Filename,
		FileContentType: contentType,
		FileSize:        file.Size,
		FileData:        data,
		UploadedBy:      userID,
	}
	if err := h.repo.Create(template); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan template", nil)
	}
	return utils.Created(c, "template formulir berhasil diunggah", template)
}

// ServeFile mengunduh/menampilkan berkas asli formulir apa adanya (kosong)
// — bukan dokumen hasil generate sistem, karena sistem tidak tahu struktur
// internal tiap formulir yang diunggah admin.
func (h *Controller) ServeFile(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id template tidak valid", nil)
	}
	t, err := h.repo.FindByID(id)
	if err != nil || len(t.FileData) == 0 {
		return utils.Fail(c, fiber.StatusNotFound, "file template tidak ditemukan", nil)
	}
	c.Set("Content-Type", t.FileContentType)
	c.Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, t.FileName))
	c.Set("Cache-Control", "private, max-age=3600")
	return c.Send(t.FileData)
}

func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id template tidak valid", nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus template", nil)
	}
	return utils.OK(c, "template berhasil dihapus", nil)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/pengajuan-templates", middleware.JWTAuth(h.jwtSvc))
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	// List & unduh file dibiarkan bisa diakses siapa pun yang login (semua
	// role perlu bisa melihat & memilih template saat membuat pengajuan) —
	// hanya unggah & hapus yang dibatasi admin/super admin.
	g.Get("/", h.List)
	g.Get("/:id/file", h.ServeFile)
	g.Post("/", onlyStaff, h.Create)
	g.Delete("/:id", onlyStaff, h.Delete)
}
