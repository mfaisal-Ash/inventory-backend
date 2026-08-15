package asset

import (
	"time"

	assetRepo "github.com/projsonal/gowms/internal/repositories/asset"
	assetHistoryRepo "github.com/projsonal/gowms/internal/repositories/asset_history"
	assetPortRepo "github.com/projsonal/gowms/internal/repositories/asset_port"
	gudangRepo "github.com/projsonal/gowms/internal/repositories/gudang"
	notificationRepo "github.com/projsonal/gowms/internal/repositories/notification"
	"github.com/projsonal/gowms/internal/repositories/role"
	usersRepo "github.com/projsonal/gowms/internal/repositories/users"
	"github.com/projsonal/gowms/pkg/constant"
	"github.com/projsonal/gowms/pkg/utils"
)

const Module = constant.ModuleAsetGudang

type Controller struct {
	repo        assetRepo.Repository
	gudangRepo  gudangRepo.Repository
	roleRepo    role.Repository
	jwtSvc      *utils.JWTService
	notifRepo   notificationRepo.Repository
	historyRepo assetHistoryRepo.Repository
	portRepo    assetPortRepo.Repository
	usersRepo   usersRepo.Repository
}

func New(
	repo assetRepo.Repository,
	gudangRepo gudangRepo.Repository,
	roleRepo role.Repository,
	jwtSvc *utils.JWTService,
	notifRepo notificationRepo.Repository,
	historyRepo assetHistoryRepo.Repository,
	portRepo assetPortRepo.Repository,
	usersRepo usersRepo.Repository,
) *Controller {
	return &Controller{
		repo:        repo,
		gudangRepo:  gudangRepo,
		roleRepo:    roleRepo,
		jwtSvc:      jwtSvc,
		notifRepo:   notifRepo,
		historyRepo: historyRepo,
		portRepo:    portRepo,
		usersRepo:   usersRepo,
	}
}

type AssetRequest struct {
	Nama       string   `json:"nama" validate:"required,max=150"`
	JenisAset  string   `json:"jenis_aset" validate:"required,oneof=tiang odc olt ont odp modem transportasi"`
	GudangID   uint     `json:"gudang_id" validate:"required"`
	JumlahPort int      `json:"jumlah_port" validate:"omitempty,min=0,max=144"`
	Latitude   *float64 `json:"latitude" validate:"omitempty,min=-90,max=90"`
	Longitude  *float64 `json:"longitude" validate:"omitempty,min=-180,max=180"`
	IPAddress  string   `json:"ip_address" validate:"omitempty,ip"`
	Keterangan string   `json:"keterangan" validate:"max=500"`
}

// AssetHistoryResponse adalah bentuk data riwayat aset yang dikirim ke client.
type AssetHistoryResponse struct {
	ID        uint      `json:"id"`
	EventType string    `json:"event_type"`
	TabelLama string    `json:"tabel_lama"`
	TabelBaru string    `json:"tabel_baru"`
	Catatan   string    `json:"catatan"`
	NamaUser  string    `json:"nama_user"`
	CreatedAt time.Time `json:"created_at"`
}

// AssetPortRequest adalah payload untuk mengisi/mengganti data satu port aset.
type AssetPortRequest struct {
	ChildAssetID  *uint  `json:"child_asset_id" validate:"omitempty"`
	CustomerName  string `json:"customer_name" validate:"omitempty,max=150"`
	CustomerPhone string `json:"customer_phone" validate:"omitempty,max=20"`
	Keterangan    string `json:"keterangan" validate:"max=255"`
}

// AssetPortResponse adalah bentuk data satu port aset yang dikirim ke client.
type AssetPortResponse struct {
	PortNumber      int    `json:"port_number"`
	Status          string `json:"status"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	Keterangan      string `json:"keterangan"`
	ChildAssetID    *uint  `json:"child_asset_id,omitempty"`
	ChildAssetNama  string `json:"child_asset_nama,omitempty"`
	ChildAssetLabel string `json:"child_asset_label,omitempty"`
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
