package role_change

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"gorm.io/gorm"
	"strings"
	"time"
)

type featureCreate struct {
	Module string `json:"module"`
	Reason string `json:"reason"`
}
type featureDecision struct {
	Status       string `json:"status"`
	DecisionNote string `json:"decision_note"`
}

func (h *Controller) ListFeatures(c *fiber.Ctx) error {
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	role, _ := c.Locals(constant.CtxRoleName).(string)
	var rows []model.FeatureRequest
	q := h.db.Preload("User").Order("created_at DESC")
	if role != constant.RoleSuperAdmin {
		q = q.Where("user_id = ?", uid)
	}
	if err := q.Find(&rows).Error; err != nil {
		return utils.Fail(c, 500, "gagal mengambil pengajuan fitur", nil)
	}
	return utils.OK(c, "pengajuan fitur berhasil diambil", rows)
}
func (h *Controller) CreateFeature(c *fiber.Ctx) error {
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	var req featureCreate
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "format tidak valid", nil)
	}
	req.Module = strings.TrimSpace(req.Module)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Module == "" || len(req.Module) > 80 || len(req.Reason) < 20 || len(req.Reason) > 1000 {
		return utils.Fail(c, 400, "modul wajib diisi dan alasan harus 20 sampai 1000 karakter", nil)
	}
	row := &model.FeatureRequest{UserID: uid, Module: req.Module, Reason: req.Reason, Status: "pending"}
	if err := h.db.Create(row).Error; err != nil {
		return utils.Fail(c, 500, "gagal menyimpan pengajuan fitur", nil)
	}
	return utils.OK(c, "pengajuan fitur berhasil dikirim", row)
}
func (h *Controller) DecideFeature(c *fiber.Ctx) error {
	var req featureDecision
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "format keputusan tidak valid", nil)
	}
	if req.Status != "approved" && req.Status != "rejected" {
		return utils.Fail(c, 400, "status tidak valid", nil)
	}
	var row model.FeatureRequest
	if err := h.db.First(&row, c.Params("id")).Error; err != nil {
		return utils.Fail(c, 404, "pengajuan fitur tidak ditemukan", nil)
	}
	if row.Status != "pending" {
		return utils.Fail(c, 409, "pengajuan sudah diputuskan", nil)
	}
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	now := time.Now()
	row.Status = req.Status
	row.DecisionNote = strings.TrimSpace(req.DecisionNote)
	row.DecidedBy = &uid
	row.DecidedAt = &now
	if err := h.db.Save(&row).Error; err != nil {
		return utils.Fail(c, 500, "gagal menyimpan keputusan", nil)
	}
	return utils.OK(c, "keputusan pengajuan fitur berhasil disimpan", row)
}
func (h *Controller) RegisterFeatureRoutes(router fiber.Router) {
	g := router.Group("/feature-requests", middleware.JWTAuth(h.jwt))
	g.Get("/", h.ListFeatures)
	g.Post("/", h.CreateFeature)
	g.Patch("/:id/decision", middleware.RequireRole(constant.RoleSuperAdmin), h.DecideFeature)
}

var _ *gorm.DB
