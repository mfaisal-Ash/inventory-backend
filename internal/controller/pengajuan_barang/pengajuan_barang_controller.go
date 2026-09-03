package pengajuan_barang

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	notification "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	pengajuanRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const Module = constant.ModulePengajuanBarang

const msgIDInvalid = "id pengajuan barang tidak valid"

func parseIDParam(c *fiber.Ctx) (uint, error) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}

func parseTanggal(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", raw)
}

// normalizeJenis mengembalikan jenis yang valid, default "keluar" kalau
// klien tidak mengirim field ini (kompatibel dengan klien lama).
func normalizeJenis(jenis string) string {
	if jenis == "" {
		return constant.JenisPengajuanKeluar
	}
	return jenis
}

func (h *Controller) validateItems(items []ItemRequest) error {
	for _, it := range items {
		if _, err := h.barangRepo.FindByID(it.BarangID); err != nil {
			return fmt.Errorf("barang id %d tidak ditemukan", it.BarangID)
		}
	}
	return nil
}

// validateJenisSpecific memvalidasi aturan yang berbeda per jenis: jenis
// "template" (pengajuan berbasis formulir yang diunggah admin) wajib
// memilih TemplateID yang benar-benar ada & masih aktif, dan tidak memakai
// daftar barang sama sekali, sedangkan jenis lain ("masuk", "keluar",
// "rusak") wajib mengisi minimal satu item dan memvalidasi keberadaan tiap
// barangnya seperti sebelumnya.
func (h *Controller) validateJenisSpecific(req *PengajuanRequest) error {
	jenis := normalizeJenis(req.Jenis)
	if jenis == constant.JenisPengajuanTemplate {
		if req.TemplateID == nil || *req.TemplateID == 0 {
			return errors.New(constant.ErrPengajuanTemplateWajib)
		}
		template, err := h.templateRepo.FindByID(*req.TemplateID)
		if err != nil || !template.IsActive {
			return errors.New(constant.ErrPengajuanTemplateTidakDitemukan)
		}
		return nil
	}
	if len(req.Items) == 0 {
		return errors.New("items wajib diisi minimal satu barang")
	}
	return h.validateItems(req.Items)
}

func toItemModels(items []ItemRequest) []model.PengajuanBarangItem {
	out := make([]model.PengajuanBarangItem, 0, len(items))
	for _, it := range items {
		out = append(out, model.PengajuanBarangItem{BarangID: it.BarangID, Qty: it.Qty})
	}
	return out
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	gudangID, _ := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64)
	barangID, _ := strconv.ParseUint(c.Query("barang_id", "0"), 10, 64)
	f := pengajuanRepo.Filter{
		Status:   c.Query("status", ""),
		GudangID: uint(gudangID),
		Jenis:    c.Query("jenis", ""),
		BarangID: uint(barangID),
	}

	list, total, err := h.repo.List(p, f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar pengajuan barang", nil)
	}
	return utils.OKWithMeta(c, "daftar pengajuan barang berhasil diambil", list, utils.BuildPaginationMeta(p, total))
}

func (h *Controller) Detail(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	pengajuan, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrPengajuanTidakDitemukan, nil)
	}
	return utils.OK(c, "detail pengajuan barang berhasil diambil", pengajuan)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req PengajuanRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	tanggal, err := parseTanggal(req.Tanggal)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "format tanggal tidak valid (YYYY-MM-DD)", nil)
	}
	if _, err := h.gudangRepo.FindGudangByID(req.GudangID); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}
	if err := h.validateJenisSpecific(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	nomor, err := h.repo.NextNomor()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat nomor pengajuan", nil)
	}
	pengajuan := &model.PengajuanBarang{
		NomorPengajuan:  nomor,
		Jenis:           normalizeJenis(req.Jenis),
		GudangID:        req.GudangID,
		Tanggal:         tanggal,
		Keperluan:       req.Keperluan,
		Perihal:         req.Perihal,
		TemplateID:      req.TemplateID,
		Status:          constant.StatusPengajuanDiajukan,
		DiajukanOleh:    userID,
		NamaPencatat:    req.NamaPencatat,
		JabatanPencatat: req.JabatanPencatat,
		Items:           toItemModels(req.Items),
	}
	if err := h.repo.Create(pengajuan); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat pengajuan barang", nil)
	}
	notification.Notify(h.notifRepo, "pengajuan_barang",
		"Pengajuan Barang Baru",
		fmt.Sprintf("%s (%s) menunggu persetujuan.", pengajuan.NomorPengajuan, pengajuan.Keperluan),
		"/pengajuan-barang", nil, constant.RoleSuperAdmin)
	return utils.Created(c, "pengajuan barang berhasil dibuat, menunggu persetujuan", pengajuan)
}

func (h *Controller) requireDiajukan(id uint) (*model.PengajuanBarang, error) {
	pengajuan, err := h.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if pengajuan.Status != constant.StatusPengajuanDiajukan {
		return nil, errors.New(constant.ErrPengajuanBukanDiajukan)
	}
	return pengajuan, nil
}

func (h *Controller) Update(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	pengajuan, err := h.requireDiajukan(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}

	var req PengajuanRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	tanggal, err := parseTanggal(req.Tanggal)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "format tanggal tidak valid (YYYY-MM-DD)", nil)
	}
	if _, err := h.gudangRepo.FindGudangByID(req.GudangID); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}
	if err := h.validateJenisSpecific(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	pengajuan.Jenis = normalizeJenis(req.Jenis)
	pengajuan.GudangID = req.GudangID
	pengajuan.Tanggal = tanggal
	pengajuan.Keperluan = req.Keperluan
	pengajuan.Perihal = req.Perihal
	pengajuan.TemplateID = req.TemplateID
	pengajuan.NamaPencatat = req.NamaPencatat
	pengajuan.JabatanPencatat = req.JabatanPencatat
	if err := h.repo.Update(pengajuan, toItemModels(req.Items)); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui pengajuan barang", nil)
	}
	return utils.OK(c, "pengajuan barang berhasil diperbarui", pengajuan)
}

func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	if _, err := h.requireDiajukan(id); err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	if err := h.repo.Delete(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menghapus pengajuan barang", nil)
	}
	return utils.OK(c, "pengajuan barang berhasil dihapus", nil)
}

func setujuiSuccessMessage(pengajuan *model.PengajuanBarang) string {
	switch pengajuan.Jenis {
	case constant.JenisPengajuanMasuk:
		if pengajuan.BarangMasuk != nil {
			return "pengajuan barang disetujui — dokumen barang masuk (" + pengajuan.BarangMasuk.NomorPenerimaan +
				") otomatis dibuat berstatus draft, lengkapi harga satuannya di halaman Barang Masuk untuk menuntaskan penerimaan"
		}
		return "pengajuan barang disetujui, dokumen barang masuk otomatis dibuat"
	case constant.JenisPengajuanRusak:
		return fmt.Sprintf("pengajuan barang disetujui, %d laporan barang rusak otomatis dibuat dan menunggu pengecekan staf",
			len(pengajuan.BarangRusak))
	case constant.JenisPengajuanTemplate:
		return "pengajuan berhasil disetujui"
	default:
		if pengajuan.BarangKeluar != nil && pengajuan.BarangKeluar.Status == constant.StatusBKDraft {
			return "pengajuan barang disetujui — dokumen barang keluarnya (" + pengajuan.BarangKeluar.NomorPengeluaran +
				") masih berstatus draft karena ada barang ber-nomor-seri, tuntaskan di halaman Barang Keluar untuk memilih nomor serinya"
		}
		return "pengajuan barang disetujui, stok telah dipotong dan dokumen barang keluar otomatis dibuat"
	}
}

func (h *Controller) Setujui(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	var req ProsesRequest
	_ = c.BodyParser(&req)

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	pengajuan, err := h.repo.Setujui(id, userID, req.NamaGA, req.JabatanGA, req.Catatan)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}

	msg := setujuiSuccessMessage(pengajuan)
	notification.Notify(h.notifRepo, "pengajuan_barang",
		"Pengajuan Barang Disetujui",
		fmt.Sprintf("%s telah disetujui.", pengajuan.NomorPengajuan),
		"/pengajuan-barang", &pengajuan.DiajukanOleh, "")
	return utils.OK(c, msg, pengajuan)
}

func (h *Controller) Tolak(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	var req TolakRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	pengajuan, err := h.repo.Tolak(id, userID, req.NamaGA, req.JabatanGA, req.Catatan)
	if err != nil {
		return utils.Fail(c, fiber.StatusConflict, err.Error(), nil)
	}
	notification.Notify(h.notifRepo, "pengajuan_barang",
		"Pengajuan Barang Ditolak",
		fmt.Sprintf("%s ditolak: %s", pengajuan.NomorPengajuan, req.Catatan),
		"/pengajuan-barang", &pengajuan.DiajukanOleh, "")
	return utils.OK(c, "pengajuan barang berhasil ditolak", pengajuan)
}

func (h *Controller) Summary(c *fiber.Ctx) error {
	diajukan, err1 := h.repo.CountByStatus(constant.StatusPengajuanDiajukan)
	disetujui, err2 := h.repo.CountByStatus(constant.StatusPengajuanDisetujui)
	ditolak, err3 := h.repo.CountByStatus(constant.StatusPengajuanDitolak)
	if err1 != nil || err2 != nil || err3 != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil ringkasan pengajuan barang", nil)
	}
	return utils.OK(c, "ringkasan pengajuan barang berhasil diambil", SummaryResponse{
		TotalDiajukan: diajukan, TotalDisetujui: disetujui, TotalDitolak: ditolak,
	})
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/pengajuan-barang", middleware.JWTAuth(h.jwtSvc))

	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	tambah := middleware.RequirePermission(h.roleRepo, Module, constant.ActionTambah)
	edit := middleware.RequirePermission(h.roleRepo, Module, constant.ActionEdit)
	approval := middleware.RequirePermission(h.roleRepo, Module, constant.ActionApprovalReject)
	onlyStaff := middleware.RequireRole(constant.RoleSuperAdmin, constant.RoleAdmin)

	g.Get("/summary", view, h.Summary)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Post("/", tambah, h.Create)
	g.Put("/:id", edit, h.Update)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
	g.Patch("/:id/setujui", approval, h.Setujui)
	g.Patch("/:id/tolak", approval, h.Tolak)
}
