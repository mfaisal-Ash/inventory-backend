package utils

import "strconv"

// UintToString mengonversi uint ke string desimal. Dipakai di tempat-tempat
// yang perlu menampilkan ID/label numerik sebagai teks (mis. subjudul item
// tempat sampah), supaya konversi angka->string konsisten di satu tempat
// dan tidak diulang-ulang dengan strconv langsung di banyak file.
func UintToString(u uint) string {
	return strconv.FormatUint(uint64(u), 10)
}
