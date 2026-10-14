// Package activity menyediakan endpoint Activity History: daftar riwayat
// aktivitas semua user dan pelaporan aksi sisi-browser (export/cetak/import).
package activity

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	activityRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/activity_log"
	"github.com/mfaisal-Ash/inventory-backend/pkg/activitylog"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const dateLayout = "2006-01-02"

type Controller struct {
	repo   activityRepo.Repository
	jwtSvc *utils.JWTService
}

func New(repo activityRepo.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, jwtSvc: jwtSvc}
}

func parseDate(c *fiber.Ctx, name string) (*time.Time, bool) {
	raw := strings.TrimSpace(c.Query(name))
	if raw == "" {
		return nil, true
	}
	t, err := time.ParseInLocation(dateLayout, raw, time.Local)
	if err != nil {
		return nil, false
	}
	return &t, true
}

// List: GET /activity-logs?page=&limit=&search=&action=&module=&user_id=&success=&dari=&sampai=
// Super admin & admin melihat riwayat semua user; karyawan hanya riwayatnya
// sendiri (filter user_id dipaksa ke dirinya).
func (h *Controller) List(c *fiber.Ctx) error {
	p := utils.PaginationFromContext(c)

	dari, ok := parseDate(c, "dari")
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "format dari harus YYYY-MM-DD", nil)
	}
	sampai, ok := parseDate(c, "sampai")
	if !ok {
		return utils.Fail(c, fiber.StatusBadRequest, "format sampai harus YYYY-MM-DD", nil)
	}

	f := activityRepo.Filter{
		Action: strings.TrimSpace(c.Query("action")),
		Module: strings.TrimSpace(c.Query("module")),
		Search: p.Search,
		Dari:   dari,
		Sampai: sampai,
		Page:   p.Page,
		Limit:  p.Limit,
	}
	switch c.Query("success") {
	case "true":
		v := true
		f.Success = &v
	case "false":
		v := false
		f.Success = &v
	}

	roleName, _ := c.Locals(constant.CtxRoleName).(string)
	selfID, _ := c.Locals(constant.CtxUserID).(uint)
	if roleName == constant.RoleSuperAdmin || roleName == constant.RoleAdmin {
		if uid, err := strconv.ParseUint(c.Query("user_id", "0"), 10, 64); err == nil {
			f.UserID = uint(uid)
		}
	} else {
		f.UserID = selfID
	}

	rows, total, err := h.repo.List(f)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat aktivitas", nil)
	}
	return utils.OKWithMeta(c, "riwayat aktivitas berhasil diambil", rows, utils.BuildPaginationMeta(p, total))
}

// Options: GET /activity-logs/options — isi dropdown filter.
func (h *Controller) Options(c *fiber.Ctx) error {
	return utils.OK(c, "opsi filter berhasil diambil", OptionsResponse{
		Actions: activitylog.Actions,
		Modules: activitylog.ModuleOptions(),
	})
}

// Track: POST /activity-logs/track — mencatat aksi yang terjadi di browser.
func (h *Controller) Track(c *fiber.Ctx) error {
	var req TrackRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	roleName, _ := c.Locals(constant.CtxRoleName).(string)
	if userID == 0 {
		return utils.Fail(c, fiber.StatusUnauthorized, "token tidak valid atau kedaluwarsa", nil)
	}

	module := strings.TrimSpace(req.Module)
	entry := &model.ActivityLog{
		UserID:      &userID,
		RoleName:    roleName,
		Action:      req.Action,
		Module:      module,
		ModuleLabel: activitylog.ModuleLabel(module),
		Description: activitylog.Truncate(strings.TrimSpace(req.Description), 480),
		Method:      c.Method(),
		Path:        activitylog.Truncate(c.Path(), 250),
		StatusCode:  fiber.StatusOK,
		Success:     true,
		IPAddress:   c.IP(),
		UserAgent:   activitylog.Truncate(c.Get(fiber.HeaderUserAgent), 250),
		Detail:      activitylog.BuildDetail("", req.Detail, ""),
		CreatedAt:   time.Now(),
	}
	if err := h.repo.Create(entry); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mencatat aktivitas", nil)
	}
	return utils.OK(c, "aktivitas tercatat", nil)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/activity-logs", middleware.JWTAuth(h.jwtSvc))
	g.Get("/options", h.Options)
	g.Get("/", h.List)
	g.Post("/track", h.Track)
}
