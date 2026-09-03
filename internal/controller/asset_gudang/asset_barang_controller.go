package assetgudang

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
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

func handleCreateOrUpdateError(c *fiber.Ctx, action string, err error) error {
	log.Printf("aset_gudang: gagal %s aset: %v", action, err)
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") {
		return utils.Fail(c, fiber.StatusConflict,
			"nomor label aset ini kebetulan sudah dipakai (kemungkinan sisa aset yang sudah dihapus di Tempat Sampah) — coba simpan sekali lagi", nil)
	}
	return utils.Fail(c, fiber.StatusInternalServerError, fmt.Sprintf("gagal %s aset", action), nil)
}

func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)
	gudangID, _ := strconv.ParseUint(c.Query("gudang_id", "0"), 10, 64)
	f := assetRepo.Filter{
		JenisAset: c.Query("jenis_aset", ""),
		GudangID:  uint(gudangID),
		Status:    c.Query("status", ""),
		Merek:     c.Query("merek", ""),
		Tipe:      c.Query("tipe", ""),
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
			GudangID: p.GudangID, GudangNama: p.GudangNama, GudangKode: p.GudangKode, GudangTipe: p.GudangTipe,
			GudangLatitude: p.GudangLatitude, GudangLongitude: p.GudangLongitude,
			ParentAssetID: p.ParentAssetID, ParentLatitude: p.ParentLatitude, ParentLongitude: p.ParentLongitude,
			JumlahPort: p.JumlahPort, PortTerisi: p.PortTerisi,
			Merek: p.Merek, Tipe: p.Tipe, KodeBarang: p.KodeBarang,
		})
	}
	return utils.OK(c, "titik peta aset berhasil diambil", out)
}

// validasiTransportasi: untuk jenis_aset "transportasi", nopol/jenis
// transportasi/nomor BPKB/tahun kendaraan wajib diisi — ini pengganti
// latitude/longitude yang wajib untuk jenis aset lain (lihat
// model.JenisAsetPunyaKoordinat).
func validasiTransportasi(req AssetRequest) string {
	if req.JenisAset != constant.JenisAsetTransportasi {
		return ""
	}
	if req.Nopol == "" || req.JenisTransportasi == "" || req.NomorBPKB == "" || req.TahunKendaraan == 0 {
		return "nomor polisi, jenis transportasi, nomor BPKB, dan tahun kendaraan wajib diisi untuk aset transportasi"
	}
	return ""
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req AssetRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	if msg := validasiTransportasi(req); msg != "" {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, msg, nil)
	}

	gudang, err := h.gudangRepo.FindGudangByID(req.GudangID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
	}
	if req.ParentAssetID != nil {
		parent, perr := h.repo.FindByID(*req.ParentAssetID)
		if perr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "aset induk tidak ditemukan", nil)
		}
		if !model.JenisIndukValid(req.JenisAset, parent.JenisAset) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("%s tidak bisa berinduk ke %s — cek urutan hierarki jaringan (OLT -> ODC -> ODP -> ONT)", req.JenisAset, parent.JenisAset), nil)
		}
	}
	if req.BarangID != nil {
		if _, berr := h.barangRepo.FindByID(*req.BarangID); berr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "kode barang tidak ditemukan", nil)
		}
	}

	a := &model.Asset{
		Nama:          req.Nama,
		JenisAset:     req.JenisAset,
		GudangID:      req.GudangID,
		Keterangan:    req.Keterangan,
		Merek:         req.Merek,
		Tipe:          req.Tipe,
		NilaiAset:     req.NilaiAset,
		ParentAssetID: req.ParentAssetID,
		JumlahPort:    req.JumlahPort,
		BarangID:      req.BarangID,
		Status:        "aktif",

		Nopol:             req.Nopol,
		JenisTransportasi: req.JenisTransportasi,
		NomorBPKB:         req.NomorBPKB,
		TahunKendaraan:    req.TahunKendaraan,
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
		return handleCreateOrUpdateError(c, "membuat", err)
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
	if msg := validasiTransportasi(AssetRequest{
		JenisAset: a.JenisAset, Nopol: req.Nopol, JenisTransportasi: req.JenisTransportasi,
		NomorBPKB: req.NomorBPKB, TahunKendaraan: req.TahunKendaraan,
	}); msg != "" {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, msg, nil)
	}
	if req.ParentAssetID != nil {
		if *req.ParentAssetID == a.ID {
			return utils.Fail(c, fiber.StatusUnprocessableEntity, "aset tidak bisa jadi induk untuk dirinya sendiri", nil)
		}
		parent, perr := h.repo.FindByID(*req.ParentAssetID)
		if perr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "aset induk tidak ditemukan", nil)
		}
		if !model.JenisIndukValid(a.JenisAset, parent.JenisAset) {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				fmt.Sprintf("%s tidak bisa berinduk ke %s — cek urutan hierarki jaringan (OLT -> ODC -> ODP -> ONT)", a.JenisAset, parent.JenisAset), nil)
		}
	}
	if req.BarangID != nil {
		if _, berr := h.barangRepo.FindByID(*req.BarangID); berr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "kode barang tidak ditemukan", nil)
		}
	}

	oldParentID := a.ParentAssetID
	oldLat, oldLng := a.Latitude, a.Longitude
	oldGudangID := a.GudangID
	oldLabelRSD := a.LabelRSD
	oldNilaiAset := a.NilaiAset
	oldNopol := a.Nopol
	oldJenisTransportasi := a.JenisTransportasi
	oldNomorBPKB := a.NomorBPKB
	oldTahunKendaraan := a.TahunKendaraan

	a.Nama = req.Nama
	a.Keterangan = req.Keterangan
	a.Merek = req.Merek
	a.Tipe = req.Tipe
	a.NilaiAset = req.NilaiAset
	a.ParentAssetID = req.ParentAssetID
	a.JumlahPort = req.JumlahPort
	a.BarangID = req.BarangID
	if a.JenisAset == constant.JenisAsetTransportasi {
		a.Nopol = req.Nopol
		a.JenisTransportasi = req.JenisTransportasi
		a.NomorBPKB = req.NomorBPKB
		a.TahunKendaraan = req.TahunKendaraan
	}

	if req.GudangID != 0 && req.GudangID != a.GudangID {
		newGudang, gerr := h.gudangRepo.FindGudangByID(req.GudangID)
		if gerr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "gudang tidak ditemukan", nil)
		}
		a.GudangID = req.GudangID

		if model.JenisAsetPunyaKoordinat(a.JenisAset) {
			if newGudang.Kode == "" {
				return utils.Fail(c, fiber.StatusUnprocessableEntity,
					"gudang tujuan belum punya kode — isi kode gudang dulu sebelum memindahkan aset ke sana", nil)
			}
			nomor, nerr := h.repo.NextRSDNumber(req.GudangID)
			if nerr != nil {
				return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat label RSD baru", nil)
			}
			a.LabelRSD = fmt.Sprintf("%s-RSD-%04d", newGudang.Kode, nomor)
		}
	}

	if model.JenisAsetPunyaKoordinat(a.JenisAset) {
		if req.Latitude == nil || req.Longitude == nil {
			return utils.Fail(c, fiber.StatusUnprocessableEntity,
				"latitude dan longitude wajib diisi untuk jenis aset ini", nil)
		}
		a.Latitude = req.Latitude
		a.Longitude = req.Longitude
	}
	if err := h.repo.Update(a); err != nil {
		return handleCreateOrUpdateError(c, "memperbarui", err)
	}

	if !samePtrUint(oldParentID, a.ParentAssetID) {
		h.logHistory(c, a.ID, "induk", ptrUintLabel(oldParentID), ptrUintLabel(a.ParentAssetID), "Aset induk (hierarki jaringan) diubah")
	}
	if !samePtrFloat(oldLat, a.Latitude) || !samePtrFloat(oldLng, a.Longitude) {
		h.logHistory(c, a.ID, "lokasi",
			fmt.Sprintf("%s, %s", ptrFloatLabel(oldLat), ptrFloatLabel(oldLng)),
			fmt.Sprintf("%s, %s", ptrFloatLabel(a.Latitude), ptrFloatLabel(a.Longitude)),
			"Titik koordinat lokasi diubah")
	}
	if oldGudangID != a.GudangID {
		h.logHistory(c, a.ID, "gudang", oldLabelRSD, a.LabelRSD,
			fmt.Sprintf("Aset dipindahkan ke gudang lain, label RSD diregenerasi (gudang #%d -> #%d)", oldGudangID, a.GudangID))
	}
	if oldNilaiAset != a.NilaiAset {
		h.logHistory(c, a.ID, "nilai_aset", strconv.FormatInt(oldNilaiAset, 10), strconv.FormatInt(a.NilaiAset, 10), "Nilai aset diubah")
	}
	if oldNopol != a.Nopol || oldJenisTransportasi != a.JenisTransportasi || oldNomorBPKB != a.NomorBPKB || oldTahunKendaraan != a.TahunKendaraan {
		h.logHistory(c, a.ID, "data_transportasi",
			fmt.Sprintf("%s / %s / %s / %d", oldNopol, oldJenisTransportasi, oldNomorBPKB, oldTahunKendaraan),
			fmt.Sprintf("%s / %s / %s / %d", a.Nopol, a.JenisTransportasi, a.NomorBPKB, a.TahunKendaraan),
			"Data transportasi (nopol/jenis/BPKB/tahun) diubah")
	}
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

	g.Get("/:id/port", view, h.ListPorts)
	g.Put("/:id/port/:nomor", edit, h.SetPort)
	g.Delete("/:id/port/:nomor", edit, h.ClearPort)
	g.Get("/:id/riwayat", view, h.ListHistory)
	g.Delete("/:id", onlyStaff, edit, h.Delete)
}
