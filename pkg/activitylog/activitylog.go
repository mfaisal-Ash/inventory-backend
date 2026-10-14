// Package activitylog berisi logika murni (tanpa I/O) untuk fitur Activity
// History: mengklasifikasi request HTTP menjadi aksi + modul, menyusun
// deskripsi bahasa Indonesia, dan menyamarkan data sensitif dari body.
package activitylog

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionApprove  = "approve"
	ActionReject   = "reject"
	ActionComplete = "complete"
	ActionCancel   = "cancel"
	ActionProtect  = "protect"
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionImport   = "import"
	ActionExport   = "export"
	ActionPrint    = "print"
	ActionOther    = "other"
)

// Actions: daftar aksi yang valid (dipakai filter di frontend).
var Actions = []string{
	ActionCreate, ActionUpdate, ActionDelete, ActionApprove, ActionReject,
	ActionComplete, ActionCancel, ActionProtect, ActionLogin, ActionLogout,
	ActionImport, ActionExport, ActionPrint, ActionOther,
}

// ClientActions: aksi yang boleh dilaporkan frontend lewat endpoint track
// (aksi yang tidak lewat request tulis ke backend, mis. export/print yang
// dibuat di browser).
var ClientActions = []string{ActionImport, ActionExport, ActionPrint}

var verbs = map[string]string{
	ActionCreate:   "Menambah",
	ActionUpdate:   "Mengubah",
	ActionDelete:   "Menghapus",
	ActionApprove:  "Menyetujui",
	ActionReject:   "Menolak",
	ActionComplete: "Menyelesaikan",
	ActionCancel:   "Membatalkan",
	ActionProtect:  "Mengatur proteksi",
	ActionLogin:    "Login",
	ActionLogout:   "Logout",
	ActionImport:   "Mengimpor",
	ActionExport:   "Mengekspor",
	ActionPrint:    "Mencetak",
	ActionOther:    "Melakukan aksi pada",
}

var moduleLabels = map[string]string{
	"barang":              "Kelola Barang",
	"barang-masuk":        "Barang Masuk",
	"barang-keluar":       "Barang Keluar",
	"barang-serial":       "Unit Barang (Nomor Seri)",
	"barang-rusak":        "Barang Rusak",
	"retur":               "Retur (Penggantian)",
	"kabel":               "Kabel (Meter)",
	"pengajuan-barang":    "Pengajuan Barang",
	"pengajuan-templates": "Template Pengajuan",
	"stock-opname":        "Manajemen Inventaris",
	"aset":                "Manajemen Aset Gudang",
	"asset-types":         "Jenis Aset",
	"gudang":              "Manajemen Gudang",
	"users":               "Manajemen User",
	"roles":               "Hak Akses & Role",
	"tasks":               "Tugas",
	"purchase-order":      "Purchase Order",
	"supplier":            "Supplier",
	"cod":                 "COD",
	"pengiriman":          "Pengiriman",
	"maintenance":         "Mode Maintenance",
	"trash":               "Tempat Sampah",
	"auth":                "Autentikasi",
	"laporan":             "Laporan",
	"neraca":              "Neraca Stok",
	"dashboard":           "Dashboard",
	"stok-barang":         "Stok Barang",
	"ringkasan-stok":      "Ringkasan Stok",
}

// ModuleLabel mengembalikan nama modul yang ramah dibaca. Modul yang belum
// terdaftar dikembalikan apa adanya supaya tetap terbaca.
func ModuleLabel(module string) string {
	if l, ok := moduleLabels[module]; ok {
		return l
	}
	if module == "" {
		return "-"
	}
	return module
}

// ModuleOptions: pasangan kode-label modul untuk dropdown filter.
func ModuleOptions() []map[string]string {
	out := make([]map[string]string, 0, len(moduleLabels))
	for k, v := range moduleLabels {
		out = append(out, map[string]string{"module": k, "label": v})
	}
	return out
}

// Prefix path yang TIDAK dicatat: polling/ping otomatis dan endpoint bantu
// yang bukan tindakan user (mencatatnya hanya membanjiri riwayat).
var skipPrefixes = []string{
	"/auth/refresh",
	"/auth/username-available",
	"/notifications",
	"/captcha",
	"/humancheck",
	"/human-check",
	"/security",
	"/geocode",
	"/activity-logs",
	"/app",
}

// ShouldSkip: true kalau path (relatif terhadap /stockrsd) tidak dicatat.
func ShouldSkip(rel string) bool {
	for _, p := range skipPrefixes {
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}

// Info hasil klasifikasi sebuah request.
type Info struct {
	Action   string
	Module   string
	EntityID string
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

var subActions = map[string]string{
	"approve":           ActionApprove,
	"setujui":           ActionApprove,
	"reject":            ActionReject,
	"tolak":             ActionReject,
	"selesai":           ActionComplete,
	"batalkan":          ActionCancel,
	"protect":           ActionProtect,
	"batalkan-delegasi": ActionCancel,
	"restore":           ActionUpdate,
	"purge":             ActionDelete,
}

// Classify memetakan method + path (relatif, tanpa prefix /stockrsd) menjadi
// aksi, modul, dan ID entitas.
func Classify(method, rel string) Info {
	rel = strings.Trim(rel, "/")
	segs := strings.Split(rel, "/")
	info := Info{Action: ActionOther}
	if len(segs) == 0 || segs[0] == "" {
		return info
	}
	info.Module = segs[0]

	if info.Module == "auth" {
		last := segs[len(segs)-1]
		switch last {
		case "login", "verify-otp":
			info.Action = ActionLogin
		case "logout":
			info.Action = ActionLogout
		case "register":
			info.Action = ActionCreate
		default:
			if method == "DELETE" {
				info.Action = ActionDelete
			}
		}
		return info
	}

	sub := ""
	for _, s := range segs[1:] {
		if isNumeric(s) {
			if info.EntityID == "" {
				info.EntityID = s
			}
			continue
		}
		sub = s
	}

	if a, ok := subActions[sub]; ok {
		info.Action = a
		return info
	}
	switch method {
	case "POST":
		info.Action = ActionCreate
	case "PUT", "PATCH":
		info.Action = ActionUpdate
	case "DELETE":
		info.Action = ActionDelete
	}
	return info
}

// Describe menyusun kalimat ringkas, mis. "Menambah Barang Masuk (ID 12)".
func Describe(action, module, entityID string, success bool, errMsg string) string {
	verb, ok := verbs[action]
	if !ok {
		verb = verbs[ActionOther]
	}
	label := ModuleLabel(module)
	var d string
	switch action {
	case ActionLogin, ActionLogout:
		d = verb
	default:
		d = verb + " " + label
	}
	if entityID != "" {
		d += " (ID " + entityID + ")"
	}
	if !success {
		d += " — GAGAL"
		if errMsg != "" {
			d += ": " + errMsg
		}
	}
	return Truncate(d, 480)
}

// Truncate memotong string berdasarkan jumlah karakter (rune), bukan byte.
func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

var sensitiveKeys = []string{
	"password", "passwd", "pass", "token", "secret", "otp", "captcha",
	"authorization", "pin", "totp", "kata_sandi", "sandi",
}

func isSensitiveKey(k string) bool {
	lk := strings.ToLower(k)
	for _, s := range sensitiveKeys {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

func mask(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if isSensitiveKey(k) {
				t[k] = "***"
				continue
			}
			t[k] = mask(val)
		}
		return t
	case []interface{}:
		for i := range t {
			t[i] = mask(t[i])
		}
		return t
	default:
		return v
	}
}

// SanitizeBody menyamarkan field sensitif (password, token, otp, dst) dari
// body JSON lalu memotongnya. Body non-JSON tidak disimpan.
func SanitizeBody(body []byte, maxChars int) string {
	if len(body) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return ""
	}
	out, err := json.Marshal(mask(v))
	if err != nil {
		return ""
	}
	return Truncate(string(out), maxChars)
}

// UsernameFromBody mengambil field "username" dari body login (untuk
// mencatat percobaan login gagal yang user-nya belum teridentifikasi).
func UsernameFromBody(body []byte) string {
	var v struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &v); err != nil {
		return ""
	}
	return Truncate(strings.TrimSpace(v.Username), 50)
}

// BuildDetail menggabungkan potongan detail jadi satu string JSON.
func BuildDetail(query, body, errMsg string) string {
	m := map[string]string{}
	if query != "" {
		m["query"] = query
	}
	if body != "" {
		m["body"] = body
	}
	if errMsg != "" {
		m["error"] = errMsg
	}
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("%v", m)
	}
	return string(b)
}
