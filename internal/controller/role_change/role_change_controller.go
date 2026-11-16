package role_change

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"gorm.io/gorm"
)

type Controller struct {
	db  *gorm.DB
	jwt *utils.JWTService
}
type createRequest struct {
	RequestedRole string `json:"requested_role"`
	Reason        string `json:"reason"`
}
type decisionRequest struct {
	Status       string `json:"status"`
	DecisionNote string `json:"decision_note"`
}

func New(db *gorm.DB, jwt *utils.JWTService) *Controller { return &Controller{db: db, jwt: jwt} }
func (h *Controller) List(c *fiber.Ctx) error {
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	role, _ := c.Locals(constant.CtxRoleName).(string)
	var rows []model.RoleChangeRequest
	q := h.db.Preload("User").Order("created_at DESC")
	if role != constant.RoleSuperAdmin {
		q = q.Where("user_id = ?", uid)
	}
	if err := q.Find(&rows).Error; err != nil {
		return utils.Fail(c, 500, "gagal mengambil permintaan perubahan role", nil)
	}
	return utils.OK(c, "permintaan perubahan role berhasil diambil", rows)
}
func (h *Controller) Create(c *fiber.Ctx) error {
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	current, _ := c.Locals(constant.CtxRoleName).(string)
	var req createRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "format permintaan tidak valid", nil)
	}
	req.RequestedRole = strings.TrimSpace(req.RequestedRole)
	req.Reason = strings.TrimSpace(req.Reason)
	if current != constant.RoleKaryawan {
		return utils.Fail(c, 403, "hanya karyawan yang dapat mengajukan perubahan role", nil)
	}
	if req.RequestedRole != constant.RoleAdmin && req.RequestedRole != constant.RoleSuperAdmin {
		return utils.Fail(c, 400, "role tujuan tidak valid", nil)
	}
	if len(req.Reason) < 20 || len(req.Reason) > 1000 {
		return utils.Fail(c, 400, "alasan harus 20 sampai 1000 karakter", nil)
	}
	var pending int64
	h.db.Model(&model.RoleChangeRequest{}).Where("user_id = ? AND status = ?", uid, "pending").Count(&pending)
	if pending > 0 {
		return utils.Fail(c, 409, "masih ada permintaan yang menunggu keputusan", nil)
	}
	row := &model.RoleChangeRequest{UserID: uid, CurrentRole: current, RequestedRole: req.RequestedRole, Reason: req.Reason, Status: "pending"}
	if err := h.db.Create(row).Error; err != nil {
		return utils.Fail(c, 500, "gagal menyimpan permintaan", nil)
	}
	return utils.OK(c, "permintaan perubahan role berhasil diajukan", row)
}
func (h *Controller) Decide(c *fiber.Ctx) error {
	id := c.Params("id")
	var req decisionRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, 400, "format keputusan tidak valid", nil)
	}
	if req.Status != "approved" && req.Status != "rejected" {
		return utils.Fail(c, 400, "status keputusan tidak valid", nil)
	}
	var row model.RoleChangeRequest
	if err := h.db.First(&row, id).Error; err != nil {
		return utils.Fail(c, 404, "permintaan tidak ditemukan", nil)
	}
	if row.Status != "pending" {
		return utils.Fail(c, 409, "permintaan sudah diputuskan", nil)
	}
	uid, _ := c.Locals(constant.CtxUserID).(uint)
	now := time.Now()
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		row.Status, row.DecisionNote, row.DecidedBy, row.DecidedAt = req.Status, strings.TrimSpace(req.DecisionNote), &uid, &now
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if req.Status == "approved" {
			var role model.Role
			if err := tx.Where("name = ?", row.RequestedRole).First(&role).Error; err != nil {
				return err
			}
			return tx.Model(&model.User{}).Where("id = ?", row.UserID).Update("role_id", role.ID).Error
		}
		return nil
	}); err != nil {
		return utils.Fail(c, 500, "gagal memproses keputusan", nil)
	}
	return utils.OK(c, "keputusan permintaan berhasil disimpan", row)
}
func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/role-change-requests", middleware.JWTAuth(h.jwt))
	g.Get("/", h.List)
	g.Post("/", h.Create)
	g.Patch("/:id/decision", middleware.RequireRole(constant.RoleSuperAdmin), h.Decide)
	h.RegisterFeatureRoutes(router)
}
