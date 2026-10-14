package activity

// TrackRequest: laporan aksi yang terjadi di browser dan tidak lewat request
// tulis biasa (export, cetak, ringkasan import).
type TrackRequest struct {
	Action      string `json:"action" validate:"required,oneof=import export print"`
	Module      string `json:"module" validate:"required,max=60"`
	Description string `json:"description" validate:"required,max=300"`
	Detail      string `json:"detail" validate:"omitempty,max=2000"`
}

type OptionsResponse struct {
	Actions []string            `json:"actions"`
	Modules []map[string]string `json:"modules"`
}
