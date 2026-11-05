package pengajuan_barang

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

var qtyPattern = regexp.MustCompile(`^\d+(?:[.,]\d+)?(?:\s+[[:alnum:][:space:]._/-]+)?$`)

func (h *Controller) validateItems(items []ItemRequest, jenis string) error {
	for _, it := range items {
		if strings.TrimSpace(it.Qty) == "" || !qtyPattern.MatchString(strings.TrimSpace(it.Qty)) {
			return errors.New("qty harus berupa jumlah, contoh: 5 rim atau 1 roll")
		}
		if jenis == constant.JenisPengajuanBarang && it.IsSerialized {
			if len(it.SerialNumbers) != it.Qty {
				return fmt.Errorf("jumlah nomor seri harus sama dengan qty (%d)", it.Qty)
			}
			seen := make(map[string]bool, len(it.SerialNumbers))
			for _, sn := range it.SerialNumbers {
				key := strings.ToLower(strings.TrimSpace(sn))
				if key == "" || seen[key] {
					return errors.New("nomor seri wajib diisi dan tidak boleh duplikat dalam satu barang")
				}
				seen[key] = true
			}
		}
		if jenis == constant.JenisPengajuanBarang && it.BarangID == 0 {
			if strings.TrimSpace(it.Kode) == "" || strings.TrimSpace(it.Nama) == "" || strings.TrimSpace(it.Merek) == "" || strings.TrimSpace(it.Tipe) == "" {
				return errors.New("kode barang, nama, merek/model, dan tipe wajib diisi untuk barang manual")
			}
			if it.IsSerialized && len(it.SerialNumbers) != it.Qty {
				return fmt.Errorf("jumlah nomor seri harus sama dengan qty (%d)", it.Qty)
			}
			continue
		}
		if _, err := h.barangRepo.FindByID(it.BarangID); err != nil {
			return fmt.Errorf("barang id %d tidak ditemukan", it.BarangID)
		}
		if it.SatuanID != nil && *it.SatuanID != 0 {
			if _, err := h.gudangRepo.FindSatuanByID(*it.SatuanID); err != nil {
				return fmt.Errorf("satuan id %d tidak ditemukan", *it.SatuanID)
			}
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
	return h.validateItems(req.Items, jenis)
}

func toItemModels(items []ItemRequest) []model.PengajuanBarangItem {
	out := make([]model.PengajuanBarangItem, 0, len(items))
	for _, it := range items {
		m := model.PengajuanBarangItem{BarangID: it.BarangID, KodeManual: strings.TrimSpace(it.Kode), Qty: it.Qty, NamaManual: strings.TrimSpace(it.Nama), MerekManual: strings.TrimSpace(it.Merek), TipeManual: strings.TrimSpace(it.Tipe), IsSerialized: it.IsSerialized}
		if len(it.SerialNumbers) > 0 {
			m.SerialNumbersRaw = strings.Join(it.SerialNumbers, "\n")
		}

		if it.SatuanID != nil && *it.SatuanID != 0 {
			m.SatuanID = it.SatuanID
		}
		out = append(out, m)
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
	jenis := normalizeJenis(req.Jenis)
	var ttdData []byte
	if strings.TrimSpace(req.TTDPencatat) != "" {
		if ttdData, err = decodeTTD(req.TTDPencatat); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	} else if constant.PengajuanWajibTTD(jenis) {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrPengajuanTTDPencatatWajib, nil)
	}
	nomor, err := h.repo.NextNomor()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat nomor pengajuan", nil)
	}
	pengajuan := &model.PengajuanBarang{
		NomorPengajuan: nomor,
		Jenis:          normalizeJenis(req.Jenis),
		GudangID:       req.GudangID,
		Tanggal:        tanggal,
		Keperluan:      req.Keperluan,
		Perihal:        req.Perihal,
		TemplateID:     req.TemplateID,
		Status: func() string {
			if role, _ := c.Locals(constant.CtxRoleName).(string); role == constant.RoleSuperAdmin {
				return constant.StatusPengajuanDisetujui
			}
			return constant.StatusPengajuanDiajukan
		}(),
		DiajukanOleh:    userID,
		NamaPencatat:    req.NamaPencatat,
		JabatanPencatat: req.JabatanPencatat,
		Items:           toItemModels(req.Items),
	}
	if ttdData != nil {
		ttd := h.buildTTD(userID, model.PeranTTDPencatat, req.NamaPencatat, req.JabatanPencatat, ttdData, c.IP())
		ttd.DokumenHash = pengajuan.ComputeDokumenHash()
		pengajuan.TandaTangan = []model.PengajuanTandaTangan{*ttd}
		if pengajuan.NamaPencatat == "" {
			pengajuan.NamaPencatat = ttd.Nama
		}
	}
	if err := h.repo.Create(pengajuan); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat pengajuan barang", nil)
	}
	if pengajuan.Status == constant.StatusPengajuanDiajukan {
		notification.Notify(h.notifRepo, "pengajuan_barang",
			"Pengajuan Barang Baru",
			fmt.Sprintf("%s (%s) menunggu persetujuan.", pengajuan.NomorPengajuan, pengajuan.Keperluan),
			"/pengajuan-barang", nil, constant.RoleSuperAdmin)
		return utils.Created(c, "pengajuan barang berhasil dibuat, menunggu persetujuan"+alurKeterangan(pengajuan), pengajuan)
	}
	return utils.Created(c, "pengajuan barang berhasil dibuat dan langsung disetujui", pengajuan)
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

	userID, _ := c.Locals(constant.CtxUserID).(uint)
	jenis := normalizeJenis(req.Jenis)
	var ttdData []byte
	if strings.TrimSpace(req.TTDPencatat) != "" {
		if ttdData, err = decodeTTD(req.TTDPencatat); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
		}
	} else if constant.PengajuanWajibTTD(jenis) {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrPengajuanTTDPencatatWajib, nil)
	}

	pengajuan.Jenis = jenis
	pengajuan.GudangID = req.GudangID
	pengajuan.Tanggal = tanggal
	pengajuan.Keperluan = req.Keperluan
	pengajuan.Perihal = req.Perihal
	pengajuan.TemplateID = req.TemplateID
	pengajuan.NamaPencatat = req.NamaPencatat
	pengajuan.JabatanPencatat = req.JabatanPencatat
	items := toItemModels(req.Items)
	pengajuan.Items = items
	var ttd *model.PengajuanTandaTangan
	if ttdData != nil {
		ttd = h.buildTTD(userID, model.PeranTTDPencatat, req.NamaPencatat, req.JabatanPencatat, ttdData, c.IP())
		ttd.DokumenHash = pengajuan.ComputeDokumenHash()
	}
	if err := h.repo.Update(pengajuan, items, ttd); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui pengajuan barang", nil)
	}
	if fresh, ferr := h.repo.FindByID(pengajuan.ID); ferr == nil {
		pengajuan = fresh
	}
	return utils.OK(c, "pengajuan barang berhasil diperbarui", pengajuan)
}

func (h *Controller) Delete(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, msgIDInvalid, nil)
	}
	if _, err := h.repo.FindByID(id); err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrPengajuanTidakDitemukan, nil)
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
	case constant.JenisPengajuanBarang:
		return "pengajuan barang berhasil disetujui"
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
	var ttd *model.PengajuanTandaTangan
	if strings.TrimSpace(req.TTD) != "" {
		data, derr := decodeTTD(req.TTD)
		if derr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, derr.Error(), nil)
		}
		ttd = h.buildTTD(userID, model.PeranTTDPenyetuju, req.NamaGA, req.JabatanGA, data, c.IP())
	}
	pengajuan, err := h.repo.Setujui(id, userID, req.NamaGA, req.JabatanGA, req.Catatan, ttd)
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
	// Harus sebelum "/:id" supaya "stok-menipis" tidak dianggap sebagai id.
	g.Get("/stok-menipis", view, h.StokMenipis)
	g.Get("/", view, h.List)
	g.Get("/:id", view, h.Detail)
	g.Get("/:id/ttd/:peran", view, h.ServeTTD)
	g.Post("/", tambah, h.Create)
	g.Put("/:id", edit, h.Update)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
	g.Patch("/:id/setujui", approval, h.Setujui)
	g.Patch("/:id/tolak", approval, h.Tolak)
}
