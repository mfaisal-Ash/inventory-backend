package assetgudang

import (
	"time"

	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	assetHistoryRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_history"
	assetPortRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_port"
	assetTypeRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_type"
	gudangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/gudang"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const Module = constant.ModuleAsetGudang

type Controller struct {
	repo          assetRepo.Repository
	gudangRepo    gudangRepo.Repository
	portRepo      assetPortRepo.Repository
	historyRepo   assetHistoryRepo.Repository
	usersRepo     usersRepo.Repository
	roleRepo      role.Repository
	jwtSvc        *utils.JWTService
	notifRepo     notificationRepo.Repository
	assetTypeRepo assetTypeRepo.Repository
}

func New(repo assetRepo.Repository, gudangRepo gudangRepo.Repository, portRepo assetPortRepo.Repository, historyRepo assetHistoryRepo.Repository, usersRepo usersRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService, notifRepo notificationRepo.Repository, assetTypeRepo assetTypeRepo.Repository) *Controller {
	return &Controller{repo: repo, gudangRepo: gudangRepo, portRepo: portRepo, historyRepo: historyRepo, usersRepo: usersRepo, roleRepo: roleRepo, jwtSvc: jwtSvc, notifRepo: notifRepo, assetTypeRepo: assetTypeRepo}
}

type AssetRequest struct {
	Nama      string   `json:"nama" validate:"required,max=150"`
	JenisAset string   `json:"jenis_aset" validate:"required,max=30"`
	GudangID  uint     `json:"gudang_id" validate:"required"`
	Latitude  *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`

	IPAddress string `json:"ip_address" validate:"omitempty,ip"`

	ParentAssetID *uint `json:"parent_asset_id"`

	JumlahPort int    `json:"jumlah_port" validate:"omitempty,min=0,max=512"`
	Keterangan string `json:"keterangan" validate:"max=500"`
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

// SummaryItem: hitungan aset per jenis — dinamis mengikuti jenis aset yang
// terdaftar di tabel asset_types (termasuk jenis buatan user), bukan
// daftar tetap lagi.
type SummaryItem struct {
	Kode  string `json:"kode"`
	Label string `json:"label"`
	Color string `json:"color"`
	Abbr  string `json:"abbr"`
	Count int64  `json:"count"`
}

type SummaryResponse struct {
	Items []SummaryItem `json:"items"`
	Total int64         `json:"total"`
}

type MapPoint struct {
	ID         uint    `json:"id"`
	Nama       string  `json:"nama"`
	JenisAset  string  `json:"jenis_aset"`
	LabelRSD   string  `json:"label_rsd"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Status     string  `json:"status"`
	IPAddress  string  `json:"ip_address"`
	PingStatus string  `json:"ping_status"`
	GudangID   uint    `json:"gudang_id"`
	GudangNama string  `json:"gudang_nama"`
	GudangKode string  `json:"gudang_kode"`
	GudangTipe string  `json:"gudang_tipe"`

	GudangLatitude  *float64 `json:"gudang_latitude"`
	GudangLongitude *float64 `json:"gudang_longitude"`

	ParentAssetID   *uint    `json:"parent_asset_id"`
	ParentLatitude  *float64 `json:"parent_latitude"`
	ParentLongitude *float64 `json:"parent_longitude"`
	JumlahPort      int      `json:"jumlah_port"`
	PortTerisi      int64    `json:"port_terisi"`
}

type AssetPortRequest struct {
	ChildAssetID  *uint  `json:"child_asset_id"`
	CustomerName  string `json:"customer_name" validate:"max=150"`
	CustomerPhone string `json:"customer_phone" validate:"max=20"`
	Keterangan    string `json:"keterangan" validate:"max=255"`
}

type AssetHistoryResponse struct {
	ID        uint      `json:"id"`
	EventType string    `json:"event_type"`
	FieldLama string    `json:"field_lama,omitempty"`
	FieldBaru string    `json:"field_baru,omitempty"`
	Catatan   string    `json:"catatan,omitempty"`
	UserNama  string    `json:"user_nama,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type AssetPortResponse struct {
	PortNumber      int    `json:"port_number"`
	Status          string `json:"status"`
	ChildAssetID    *uint  `json:"child_asset_id,omitempty"`
	ChildAssetNama  string `json:"child_asset_nama,omitempty"`
	ChildAssetLabel string `json:"child_asset_label,omitempty"`
	CustomerName    string `json:"customer_name,omitempty"`
	CustomerPhone   string `json:"customer_phone,omitempty"`
	Keterangan      string `json:"keterangan,omitempty"`
}
