package assetgudang

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	notifikasi "github.com/inventory-backend/internal/controller/notifikasi"
	"github.com/inventory-backend/internal/model"
	assetRepo "github.com/inventory-backend/internal/repositories/asset"
	"github.com/inventory-backend/pkg/netping"
	"github.com/inventory-backend/pkg/utils"
)

const pingTimeout = 2 * time.Second

func (h *Controller) Ping(c *fiber.Ctx) error {
	id, err := parseIDParam(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id aset tidak valid", nil)
	}
	a, err := h.repo.FindByID(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "aset tidak ditemukan", nil)
	}
	if a.IPAddress == "" {
		return utils.Fail(c, fiber.StatusUnprocessableEntity,
			"aset ini belum punya alamat IP — isi dulu lewat form Ubah Aset sebelum cek ping", nil)
	}

	res, perr := netping.Check(a.IPAddress, pingTimeout)
	now := time.Now()
	statusLama := a.PingStatus
	if perr != nil {
		a.PingStatus = "unknown"
		a.LastPingAt = &now
		_ = h.repo.Update(a)
		return utils.Fail(c, fiber.StatusBadGateway, "gagal melakukan ping: "+perr.Error(), nil)
	}

	if res.Online {
		a.PingStatus = "online"
	} else {
		a.PingStatus = "offline"
	}
	a.LastPingAt = &now
	if err := h.repo.Update(a); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan hasil ping", nil)
	}
	if statusLama != a.PingStatus {
		h.logHistory(c, a.ID, "ping", statusLama, a.PingStatus, "")
	}

	return utils.OK(c, "ping selesai", PingResponse{
		ID: a.ID, IPAddress: a.IPAddress, PingStatus: a.PingStatus,
		LastPingAt: a.LastPingAt, RTTMs: res.RTT.Milliseconds(),
	})
}

func (h *Controller) PingAll(c *fiber.Ctx) error {
	list, _, err := h.repo.List(utils.PaginationParams{Page: 1, Limit: 100000}, assetRepo.Filter{})
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar aset", nil)
	}

	const maxConcurrentPing = 20
	sem := make(chan struct{}, maxConcurrentPing)
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]PingResponse, 0, len(list))

	for i := range list {
		a := list[i]
		if a.IPAddress == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(a model.Asset) {
			defer wg.Done()
			defer func() { <-sem }()

			wasOffline := a.PingStatus == "offline"
			res, perr := netping.Check(a.IPAddress, pingTimeout)
			now := time.Now()
			a.LastPingAt = &now
			rttMs := int64(0)
			if perr != nil {
				a.PingStatus = "unknown"
			} else if res.Online {
				a.PingStatus = "online"
				rttMs = res.RTT.Milliseconds()
			} else {
				a.PingStatus = "offline"
			}
			_ = h.repo.Update(&a)

			if a.PingStatus == "offline" && !wasOffline {
				notifikasi.Notify(h.notifRepo, "ping",
					"Aset Terdeteksi Offline",
					a.Nama+" ("+a.LabelRSD+") tidak merespon ping.",
					"/home/aset-gudang", nil, "admin")
			}

			mu.Lock()
			results = append(results, PingResponse{
				ID: a.ID, IPAddress: a.IPAddress, PingStatus: a.PingStatus,
				LastPingAt: a.LastPingAt, RTTMs: rttMs,
			})
			mu.Unlock()
		}(a)
	}
	wg.Wait()

	return utils.OK(c, "ping massal selesai", results)
}
