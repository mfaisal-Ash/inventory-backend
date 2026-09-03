package assetgudang

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func (h *Controller) logHistory(c *fiber.Ctx, assetID uint, eventType, fieldLama, fieldBaru, catatan string) {
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	var uid *uint
	userNama := ""
	if userID != 0 {
		uid = &userID
		if u, err := h.usersRepo.FindByID(userID); err == nil {
			userNama = u.FullName
		}
	}
	_ = h.historyRepo.Log(&model.AssetHistory{
		AssetID: assetID, EventType: eventType, FieldLama: fieldLama, FieldBaru: fieldBaru,
		Catatan: catatan, UserID: uid, UserNama: userNama,
	})
}

// ListHistory: default-nya (tanpa query "bulan") tetap perilaku lama —
// 100 kejadian terakhir, dipakai tampilan tracking Harian. Kalau query
// "bulan" diisi (format "2026-08"), fitur tracking Bulanan: ambil SEMUA
// kejadian dalam bulan itu (tanpa batas 100) supaya rekap bulanannya utuh.
func (h *Controller) ListHistory(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id aset tidak valid", nil)
	}

	bulan := c.Query("bulan", "")
	var rows []model.AssetHistory
	if bulan != "" {
		dari, perr := time.Parse("2006-01", bulan)
		if perr != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "format bulan tidak valid — gunakan YYYY-MM", nil)
		}
		sampai := dari.AddDate(0, 1, 0)
		rows, err = h.historyRepo.ListByAssetRange(id, dari, sampai)
	} else {
		rows, err = h.historyRepo.ListByAsset(id, 100)
	}
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil riwayat aset", nil)
	}
	out := make([]AssetHistoryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, AssetHistoryResponse{
			ID: r.ID, EventType: r.EventType, FieldLama: r.FieldLama, FieldBaru: r.FieldBaru,
			Catatan: r.Catatan, UserNama: r.UserNama, CreatedAt: r.CreatedAt,
		})
	}
	return utils.OK(c, "riwayat aset berhasil diambil", out)
}
