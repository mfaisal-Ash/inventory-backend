package inspeksi_stok

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"gorm.io/gorm"
)

func (h *Controller) Create(c *fiber.Ctx) error {
	var req CreateRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	if req.NormalQty+req.RusakQty+req.BelumDiperiksaQty != req.DiperiksaQty {
		return utils.Fail(c, fiber.StatusBadRequest, "normal, rusak, dan belum diperiksa harus sama dengan jumlah diperiksa", nil)
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	row := &model.InspeksiStok{BarangID: req.BarangID, GudangID: req.GudangID, LabelBatch: req.LabelBatch, DiperiksaQty: req.DiperiksaQty, NormalQty: req.NormalQty, RusakQty: req.RusakQty, BelumDiperiksaQty: req.BelumDiperiksaQty, Catatan: req.Catatan, DiperiksaOleh: userID, DiperiksaPada: time.Now()}
	if err := h.repo.CreateInspection(row); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	return utils.Created(c, "inspeksi stok berhasil disimpan", row)
}

func (h *Controller) Balance(c *fiber.Ctx) error {
	barangID, err1 := strconv.ParseUint(c.Query("barang_id"), 10, 64)
	gudangID, err2 := strconv.ParseUint(c.Query("gudang_id"), 10, 64)
	if err1 != nil || err2 != nil || barangID == 0 || gudangID == 0 {
		return utils.Fail(c, fiber.StatusBadRequest, "barang_id dan gudang_id wajib valid", nil)
	}
	row, err := h.repo.GetBalance(uint(barangID), uint(gudangID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return utils.OK(c, "saldo kondisi belum tersedia", model.BalanceStok{BarangID: uint(barangID), GudangID: uint(gudangID)})
	}
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil saldo kondisi stok", nil)
	}
	return utils.OK(c, "saldo kondisi stok berhasil diambil", row)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/inspeksi-stok", middleware.JWTAuth(h.jwtSvc))
	view := middleware.RequirePermission(h.roleRepo, constant.ModuleBarangRusak, constant.ActionView)
	edit := middleware.RequirePermission(h.roleRepo, constant.ModuleBarangRusak, constant.ActionEdit)
	g.Post("/", edit, h.Create)
	g.Get("/balance", view, h.Balance)
}
