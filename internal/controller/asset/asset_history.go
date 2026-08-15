package asset

import (
	"github.com/gofiber/fiber/v2"

	"github.com/projsonal/gowms/internal/model"
	"github.com/projsonal/gowms/pkg/constant"
	"github.com/projsonal/gowms/pkg/utils"
)

func (h *Controller) logHistory(c *fiber.Ctx, assetID uint, eventType, TabelLama, TabelBaru, catatan string) {
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	var uid *uint
	NamaUser := ""
	if userID != 0 {
		uid = &userID
		if u, err := h.usersRepo.FindByID(userID); err == nil {
			NamaUser = u.FullName
		}
	}
	_ = h.historyRepo.Log(&model.AssetHistory{
		AssetID: assetID, EventType: eventType, TabelLama: TabelLama, TabelBaru: TabelBaru,
		Catatan: catatan, UserID: uid, NamaUser: NamaUser,
	})
}

func (h *Controller) ListHistory(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id aset tidak valid", nil)
	}
	rows, err := h.historyRepo.ListByAsset(id, 100)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat aset", nil)
	}
	out := make([]AssetHistoryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, AssetHistoryResponse{
			ID: r.ID, EventType: r.EventType, TabelLama: r.TabelLama, TabelBaru: r.TabelBaru,
			Catatan: r.Catatan, NamaUser: r.NamaUser, CreatedAt: r.CreatedAt,
		})
	}
	return utils.OK(c, "riwayat aset berhasil diambil", out)
}
