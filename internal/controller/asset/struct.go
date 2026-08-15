package asset

import (
	"time"

	assetRepo "github.com/projsonal/gowms/internal/repositories/asset"
	gudangRepo "github.com/projsonal/gowms/internal/repositories/gudang"
	notificationRepo "github.com/projsonal/gowms/internal/repositories/notification"
	"github.com/projsonal/gowms/internal/repositories/role"
	"github.com/projsonal/gowms/pkg/constant"
	"github.com/projsonal/gowms/pkg/utils"
)

const Module = constant.ModuleAsetGudang

type Controller struct {
	repo       assetRepo.Repository
	gudangRepo gudangRepo.Repository
	roleRepo   role.Repository
	jwtSvc     *utils.JWTService
	notifRepo  notificationRepo.Repository
}

func New(repo assetRepo.Repository, gudangRepo gudangRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService, notifRepo notificationRepo.Repository) *Controller {
	return &Controller{repo: repo, gudangRepo: gudangRepo, roleRepo: roleRepo, jwtSvc: jwtSvc, notifRepo: notifRepo}
}

type AssetRequest struct {
	Nama       string   `json:"nama" validate:"required,max=150"`
	JenisAset  string   `json:"jenis_aset" validate:"required,oneof=tiang odc olt ont odp modem transportasi"`
	GudangID   uint     `json:"gudang_id" validate:"required"`
	Latitude   *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude  *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	IPAddress  string   `json:"ip_address" validate:"omitempty,ip"`
	Keterangan string   `json:"keterangan" validate:"max=500"`
}

type PingResponse struct {
	ID         uint       `json:"id"`
	IPAddress  string     `json:"ip_address"`
	PingStatus string     `json:"ping_status"`
	LastPingAt *time.Time `json:"last_ping_at"`
	RTTMs      int64      `json:"rtt_ms,omitempty"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=aktif rusak nonaktif"`
}

type SummaryResponse struct {
	Tiang        int64 `json:"tiang"`
	Odc          int64 `json:"odc"`
	Olt          int64 `json:"olt"`
	Ont          int64 `json:"ont"`
	Odp          int64 `json:"odp"`
	Modem        int64 `json:"modem"`
	Transportasi int64 `json:"transportasi"`
	Total        int64 `json:"total"`
}

type MapPoint struct {
	ID         uint    `json:"id"`
	Nama       string  `json:"nama"`
	JenisAset  string  `json:"jenis_aset"`
	LabelRSD   string  `json:"label_rsd"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Status     string  `json:"status"`
	GudangID   uint    `json:"gudang_id"`
	GudangNama string  `json:"gudang_nama"`
	GudangKode string  `json:"gudang_kode"`
	GudangTipe string  `json:"gudang_tipe"` // "pusat" | "cabang"
}
