// Package geocoding menerjemahkan alamat teks (khususnya alamat Indonesia,
// yang sering memuat notasi RT/RW) menjadi koordinat lintang/bujur lewat
// Nominatim (OpenStreetMap).
//
// Kenapa ini dipindah ke backend (sebelumnya frontend memanggil Nominatim
// langsung dari browser):
//  1. Kebijakan penggunaan Nominatim mewajibkan header User-Agent yang jelas
//     mengidentifikasi aplikasi pemanggil — ini tidak bisa diatur dari
//     fetch() di browser (User-Agent dikunci oleh browser itu sendiri).
//  2. Kebijakan penggunaan Nominatim membatasi maksimal 1 request/detik,
//     dihitung per aplikasi (bukan per user) — cuma bisa ditegakkan dengan
//     benar kalau semua request lewat satu titik (backend), bukan dari
//     banyak browser pengguna sekaligus.
//  3. Caching di sini mengurangi jumlah request keluar untuk alamat yang
//     sama/mirip yang dicoba berkali-kali.
//
// Akar masalah "koordinat tidak akurat" yang dilaporkan: alamat gaya
// Indonesia sering menulis nomor rumah sebagai "No.RT03/13" (notasi RT/RW
// dipakai menggantikan nomor rumah biasa). Nominatim tidak mengenal notasi
// ini sebagai house_number yang valid, jadi pencarian dengan token itu
// gagal — dan algoritma fallback lama (drop token per token dari kiri)
// langsung membuang SELURUH segmen jalan ("Jl. Cimuncang No.RT03/13") lalu
// lompat ke segmen berikutnya (kelurahan/desa), sehingga hasil yang didapat
// cuma akurat di level kelurahan, bukan level jalan. Perbaikan utama di
// sini adalah membersihkan notasi RT/RW/No. tersebut TANPA membuang nama
// jalannya, supaya pencarian tetap mencoba mencocokkan di level jalan dulu.
package geocoding

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	baseURL           = "https://nominatim.openstreetmap.org/search"
	requestTimeout    = 6 * time.Second
	minRequestGap     = 1100 * time.Millisecond // kebijakan Nominatim: maks 1 req/detik, kasih sedikit jeda ekstra
	cacheTTL          = 24 * time.Hour
	maxResponseBytes  = 1 << 20
	interCandidateGap = 250 * time.Millisecond
)

// Precision menyatakan seberapa spesifik titik koordinat yang ditemukan,
// supaya pengguna tahu seberapa jauh mereka masih perlu menggeser pin di
// peta secara manual.
type Precision string

const (
	PrecisionStreet  Precision = "street" // ketemu sampai level jalan/nomor rumah
	PrecisionArea    Precision = "area"   // ketemu sampai level kelurahan/desa/kecamatan
	PrecisionRegion  Precision = "region" // cuma ketemu level kabupaten/kota/provinsi
	PrecisionUnknown Precision = "unknown"
)

type Result struct {
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	DisplayName string    `json:"display_name"`
	Precision   Precision `json:"precision"`
	// MatchedQuery: query persis yang berhasil, buat keperluan debug/log.
	MatchedQuery string `json:"-"`
}

type cacheEntry struct {
	result    *Result
	expiresAt time.Time
}

// Service adalah client Nominatim singleton — sengaja dibuat sebagai satu
// instance yang dipakai bersama (lihat rate limiter di bawah), bukan
// dibuat baru setiap request.
type Service struct {
	client    *http.Client
	userAgent string

	rateMu      sync.Mutex
	lastRequest time.Time

	cacheMu sync.RWMutex
	cache   map[string]cacheEntry
}

func NewService(appName string) *Service {
	ua := strings.TrimSpace(appName)
	if ua == "" {
		ua = "WMS-RSD"
	}
	return &Service{
		client:    &http.Client{Timeout: requestTimeout},
		userAgent: fmt.Sprintf("%s/1.0 (internal warehouse management system; geocoding proxy)", ua),
		cache:     make(map[string]cacheEntry),
	}
}

var errNoMatch = fmt.Errorf("alamat tidak ditemukan")

// Geocode mencoba menerjemahkan rawAddress ke koordinat. Alur pencariannya:
//  1. Cek cache.
//  2. Bersihkan setiap segmen alamat dari notasi RT/RW/"No." administratif
//     (tanpa membuang nama jalan/tempatnya).
//  3. Coba query terstruktur (street/city/county/state/postalcode) hasil
//     parsing heuristik alamat Indonesia — biasanya lebih presisi kalau
//     datanya cocok dengan tag OSM.
//  4. Coba query bebas teks dengan alamat yang sudah dibersihkan, utuh
//     dulu (supaya nama jalan tidak langsung dibuang).
//  5. Kalau masih gagal, baru turun bertahap: buang segmen paling kiri
//     satu per satu (fallback lama), sampai akhirnya cuma tersisa
//     kabupaten/provinsi.
//
// Basically: sekarang lebih dari satu strategi dicoba, dan yang dipakai
// adalah hasil pertama yang ketemu (dengan urutan strategi yang paling
// mungkin presisi duluan) — bukan asal ambil kecocokan pertama seperti
// sebelumnya.
func (s *Service) Geocode(ctx context.Context, rawAddress string) (*Result, error) {
	original := strings.TrimSpace(rawAddress)
	if original == "" {
		return nil, fmt.Errorf("alamat kosong")
	}

	cacheKey := normalizeForCache(original)
	if cached, ok := s.getCached(cacheKey); ok {
		return cached, nil
	}

	parsed := parseIndonesianAddress(original)

	result, err := s.tryStructured(ctx, parsed)
	if err != nil {
		result, err = s.tryFreeTextCandidates(ctx, parsed)
	}
	if err != nil {
		return nil, err
	}

	s.setCached(cacheKey, result)
	return result, nil
}

// tryStructured mencoba parameter terstruktur Nominatim. Ini bisa gagal
// total (0 hasil) kalau data OSM di area tersebut tidak ditag serapi itu —
// makanya selalu ada fallback tryFreeTextCandidates setelahnya.
func (s *Service) tryStructured(ctx context.Context, parsed parsedAddress) (*Result, error) {
	cityValue := parsed.city()
	if parsed.street == "" && cityValue == "" && parsed.county == "" {
		return nil, errNoMatch
	}
	values := url.Values{}
	values.Set("format", "json")
	values.Set("limit", "3")
	values.Set("addressdetails", "1")
	values.Set("countrycodes", "id")
	if parsed.street != "" {
		values.Set("street", parsed.street)
	}
	if cityValue != "" {
		values.Set("city", cityValue)
	}
	if parsed.county != "" {
		values.Set("county", parsed.county)
	}
	if parsed.state != "" {
		values.Set("state", parsed.state)
	}
	if parsed.postalcode != "" {
		values.Set("postalcode", parsed.postalcode)
	}

	items, err := s.request(ctx, values)
	if err != nil || len(items) == 0 {
		return nil, errNoMatch
	}
	return bestOf(items), nil
}

// tryFreeTextCandidates menjalankan daftar kandidat query bebas-teks
// secara berurutan (dari paling spesifik ke paling umum) dan memakai hasil
// pertama yang ketemu.
func (s *Service) tryFreeTextCandidates(ctx context.Context, parsed parsedAddress) (*Result, error) {
	candidates := buildFreeTextCandidates(parsed)
	var lastErr error = errNoMatch
	for i, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if i > 0 {
			time.Sleep(interCandidateGap)
		}
		values := url.Values{}
		values.Set("format", "json")
		values.Set("limit", "3")
		values.Set("addressdetails", "1")
		values.Set("countrycodes", "id")
		values.Set("q", candidate)

		items, err := s.request(ctx, values)
		if err != nil {
			lastErr = err
			continue
		}
		if len(items) == 0 {
			continue
		}
		result := bestOf(items)
		result.MatchedQuery = candidate
		return result, nil
	}
	return nil, lastErr
}

type nominatimItem struct {
	Lat         string        `json:"lat"`
	Lon         string        `json:"lon"`
	DisplayName string        `json:"display_name"`
	Address     nominatimAddr `json:"address"`
	Importance  json.Number   `json:"importance"`
}

type nominatimAddr struct {
	Road         string `json:"road"`
	HouseNumber  string `json:"house_number"`
	Village      string `json:"village"`
	Suburb       string `json:"suburb"`
	Town         string `json:"town"`
	CityDistrict string `json:"city_district"`
	County       string `json:"county"`
	State        string `json:"state"`
}

func classifyPrecision(addr nominatimAddr) Precision {
	if addr.Road != "" {
		return PrecisionStreet
	}
	if addr.Village != "" || addr.Suburb != "" || addr.Town != "" || addr.CityDistrict != "" {
		return PrecisionArea
	}
	if addr.County != "" || addr.State != "" {
		return PrecisionRegion
	}
	return PrecisionUnknown
}

// bestOf memilih hasil dengan presisi terbaik di antara beberapa kandidat
// yang dikembalikan Nominatim untuk satu query (bukan cuma ambil elemen
// pertama begitu saja).
func bestOf(items []nominatimItem) *Result {
	rank := map[Precision]int{PrecisionStreet: 3, PrecisionArea: 2, PrecisionRegion: 1, PrecisionUnknown: 0}
	best := items[0]
	bestPrecision := classifyPrecision(best.Address)
	for _, item := range items[1:] {
		p := classifyPrecision(item.Address)
		if rank[p] > rank[bestPrecision] {
			best = item
			bestPrecision = p
		}
	}
	return &Result{
		Latitude:    parseFloatSafe(best.Lat),
		Longitude:   parseFloatSafe(best.Lon),
		DisplayName: best.DisplayName,
		Precision:   bestPrecision,
	}
}

func parseFloatSafe(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%g", &f)
	return f
}

// request menegakkan rate limit (>=1 req/detik, kebijakan Nominatim),
// mengirim request dengan User-Agent yang jelas, dan membatasi ukuran
// response yang dibaca.
func (s *Service) request(ctx context.Context, values url.Values) ([]nominatimItem, error) {
	s.throttle()

	reqURL := baseURL + "?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("bikin request geocoding: %w", err)
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept-Language", "id")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("panggil nominatim: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nominatim mengembalikan status %d", resp.StatusCode)
	}

	reader := io.LimitReader(resp.Body, maxResponseBytes)
	var items []nominatimItem
	if err := json.NewDecoder(reader).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response nominatim: %w", err)
	}
	return items, nil
}

func (s *Service) throttle() {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	elapsed := time.Since(s.lastRequest)
	if elapsed < minRequestGap {
		time.Sleep(minRequestGap - elapsed)
	}
	s.lastRequest = time.Now()
}

func (s *Service) getCached(key string) (*Result, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	entry, ok := s.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.result, true
}

func (s *Service) setCached(key string, result *Result) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[key] = cacheEntry{result: result, expiresAt: time.Now().Add(cacheTTL)}
}

func normalizeForCache(address string) string {
	return strings.ToLower(strings.Join(strings.Fields(address), " "))
}

// --- Parsing alamat Indonesia ---

type parsedAddress struct {
	street     string
	village    string
	district   string // kecamatan
	county     string // kabupaten/kota
	state      string // provinsi
	postalcode string
	// cleanedParts: seluruh segmen (dipisah koma) setelah dibersihkan dari
	// notasi RT/RW/No. administratif, urutan tetap seperti aslinya —
	// dipakai buat membangun kandidat query bebas-teks.
	cleanedParts []string
}

var (
	postalCodeRe = regexp.MustCompile(`\b\d{5}\b`)
	// Notasi RT/RW dalam berbagai variasi umum di alamat Indonesia, mis:
	// "No.RT03/13", "RT.03/RW.13", "RT 03 RW 13", "No. RT 03/13".
	rtRwNoiseRe  = regexp.MustCompile(`(?i)\bNo\.?\s*RT\.?\s*\d+\s*/?\s*(?:RW\.?\s*)?\d*\b`)
	rtOnlyRe     = regexp.MustCompile(`(?i)\bRT\.?\s*\d+\s*(?:/\s*(?:RW\.?\s*)?\d+)?\b`)
	rwOnlyRe     = regexp.MustCompile(`(?i)\bRW\.?\s*\d+\b`)
	multiSpaceRe = regexp.MustCompile(`\s{2,}`)
	kecPrefixRe  = regexp.MustCompile(`(?i)^kec(?:amatan)?\.?\s+`)
	kabPrefixRe  = regexp.MustCompile(`(?i)^kab(?:upaten)?\.?\s+`)
	kotaPrefixRe = regexp.MustCompile(`(?i)^kota\.?\s+`)
)

// stripAdministrativeNoise membuang notasi RT/RW/"No.RT.." dari satu
// segmen alamat TANPA membuang nama jalan/tempat yang menyertainya —
// ini akar perbaikannya (lihat komentar package di atas).
func stripAdministrativeNoise(segment string) string {
	s := segment
	s = rtRwNoiseRe.ReplaceAllString(s, "")
	s = rtOnlyRe.ReplaceAllString(s, "")
	s = rwOnlyRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.Trim(s, ",.- ")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func parseIndonesianAddress(raw string) parsedAddress {
	rawParts := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		c := stripAdministrativeNoise(part)
		if c != "" {
			cleaned = append(cleaned, c)
		}
	}

	result := parsedAddress{cleanedParts: cleaned}
	if len(cleaned) == 0 {
		return result
	}

	result.street = cleaned[0]

	// Segmen terakhir biasanya "Provinsi KODEPOS" (mis. "Jawa Barat 40375").
	last := cleaned[len(cleaned)-1]
	if m := postalCodeRe.FindString(last); m != "" {
		result.postalcode = m
		result.state = strings.TrimSpace(postalCodeRe.ReplaceAllString(last, ""))
	} else {
		result.state = last
	}

	// Cari segmen kabupaten/kota dan kecamatan di antara segmen tengah.
	for i, part := range cleaned {
		if i == 0 || i == len(cleaned)-1 {
			continue
		}
		switch {
		case kecPrefixRe.MatchString(part):
			result.district = strings.TrimSpace(kecPrefixRe.ReplaceAllString(part, ""))
		case kabPrefixRe.MatchString(part):
			result.county = part // simpan utuh ("Kabupaten Bandung") — data OSM county Indonesia umumnya memakai bentuk lengkap ini
		case kotaPrefixRe.MatchString(part):
			result.county = part
		default:
			if result.village == "" {
				result.village = part
			}
		}
	}

	// "city" buat Nominatim structured query diisi dari unit yang paling
	// mendekati kota/desa — pakai desa/kelurahan kalau ada, else kecamatan.
	return result
}

// city mengembalikan segmen yang paling cocok dipakai sebagai parameter
// structured "city" Nominatim.
func (p parsedAddress) city() string {
	if p.village != "" {
		return p.village
	}
	return p.district
}

func buildFreeTextCandidates(p parsedAddress) []string {
	var candidates []string

	// 1) Alamat penuh yang sudah dibersihkan, utuh — supaya nama jalan
	//    tetap ikut dicoba di level paling spesifik dulu.
	if len(p.cleanedParts) > 0 {
		candidates = append(candidates, strings.Join(p.cleanedParts, ", "))
	}

	// 2) Turun bertahap dari kiri (fallback lama), tapi dari data yang
	//    SUDAH dibersihkan dari notasi RT/RW.
	for drop := 1; drop < len(p.cleanedParts); drop++ {
		attempt := strings.Join(p.cleanedParts[drop:], ", ")
		if attempt != "" {
			candidates = append(candidates, attempt)
		}
	}

	return dedupe(candidates)
}

func dedupe(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, item)
	}
	return out
}
