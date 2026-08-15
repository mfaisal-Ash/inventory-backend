package constant

// Jenis aset gudang (modul Aset & GeoIP) — harus sinkron dengan validate
// tag `oneof=...` di internal/controller/asset/struct.go AssetRequest.
const (
	JenisAsetTiang        = "tiang"
	JenisAsetODC          = "odc"
	JenisAsetOLT          = "olt"
	JenisAsetONT          = "ont"
	JenisAsetODP          = "odp"
	JenisAsetModem        = "modem"
	JenisAsetTransportasi = "transportasi"
)
