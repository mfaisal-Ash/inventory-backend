package utils

import "net/http"

// ImageMatchesType memastikan ISI file benar-benar gambar bertipe want
// ("image/jpeg" atau "image/png"), bukan sekadar berekstensi .jpg/.png.
// Mencegah file lain (HTML, skrip, dsb.) disimpan lalu dilayani sebagai gambar.
func ImageMatchesType(data []byte, want string) bool {
	if len(data) == 0 {
		return false
	}
	return http.DetectContentType(data) == want
}
