package neraca

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	neracaRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Neraca stok memakai izin modul Laporan (view).
const Module = constant.ModuleLaporan

type Controller struct {
	repo     neracaRepo.Repository
	roleRepo role.Repository
	jwtSvc   *utils.JWTService
}

func New(repo neracaRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, roleRepo: roleRepo, jwtSvc: jwtSvc}
}

const layout = "2006-01-02"

func parseUint(c *fiber.Ctx, name string) uint {
	v, _ := strconv.ParseUint(c.Query(name, "0"), 10, 64)
	return uint(v)
}

// Hitung: GET /neraca?dari=YYYY-MM-DD&sampai=YYYY-MM-DD&gudang_id=&barang_id=&search=
// Tanpa dari/sampai: awal bulan berjalan s.d. hari ini.
func (h *Controller) Hitung(c *fiber.Ctx) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	dari := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	sampai := today

	if raw := strings.TrimSpace(c.Query("dari")); raw != "" {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "format dari harus YYYY-MM-DD", nil)
		}
		dari = t
	}
	if raw := strings.TrimSpace(c.Query("sampai")); raw != "" {
		t, err := time.ParseInLocation(layout, raw, time.Local)
		if err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "format sampai harus YYYY-MM-DD", nil)
		}
		sampai = t
	}
	if sampai.Before(dari) {
		return utils.Fail(c, fiber.StatusBadRequest, "tanggal sampai tidak boleh sebelum tanggal dari", nil)
	}

	res, err := h.repo.Hitung(neracaRepo.Filter{
		Dari:     dari,
		Sampai:   sampai,
		GudangID: parseUint(c, "gudang_id"),
		BarangID: parseUint(c, "barang_id"),
		Search:   c.Query("search", ""),
	})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyusun neraca stok", nil)
	}
	return utils.OK(c, "neraca stok berhasil disusun", res)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/neraca", middleware.JWTAuth(h.jwtSvc))
	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	g.Get("/", view, h.Hitung)
}
