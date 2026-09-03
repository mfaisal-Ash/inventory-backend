// Command hashpw: bikin/cek hash bcrypt password secara manual, dipakai
// kalau kamu perlu isi/ubah kolom password_hash langsung lewat database
// (mis. pakai pgAdmin/DBeaver/Supabase Studio), bukan lewat form aplikasi.
//
// Password TETAP di-hash pakai bcrypt (sama seperti alur normal aplikasi
// lewat pkg/utils.HashPassword) — bukan disimpan apa adanya (plain text).
// Ini SENGAJA, bukan keterbatasan: bcrypt itu satu-arah (tidak bisa
// "didekode" balik ke password aslinya oleh siapa pun, termasuk kamu
// sendiri atau developer aplikasi ini), jadi walau database bocor/diretas,
// password asli semua user tetap tidak langsung kebaca. Tool ini cuma
// membantu kamu MEMBUAT hash yang valid dari password yang kamu mau,
// supaya bisa ditempel manual ke kolom password_hash.
//
// Cara pakai:
//
//	go run ./cmd/hashpw -password "12345678"
//	  → mencetak hash bcrypt buat ditempel ke kolom password_hash
//
//	go run ./cmd/hashpw -password "12345678" -verify '$2a$10$...'
//	  → cek apakah password itu cocok dengan hash yang sudah ada di DB
//	    (buat verifikasi, bukan buat generate)
//
// Kalau -password tidak diisi, tool akan minta diketik interaktif (supaya
// password tidak ikut kesimpan di riwayat command shell/terminal history).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func main() {
	password := flag.String("password", "", "password polos yang mau di-hash (kosongkan supaya diketik interaktif, lebih aman)")
	verifyAgainst := flag.String("verify", "", "opsional: hash bcrypt yang sudah ada (mis. dari kolom password_hash) untuk dicek kecocokannya, bukan generate baru")
	flag.Parse()

	plain := *password
	if plain == "" {
		plain = promptPassword("Ketik password yang mau di-hash: ")
	}
	if strings.TrimSpace(plain) == "" {
		fmt.Fprintln(os.Stderr, "password tidak boleh kosong")
		os.Exit(1)
	}

	if *verifyAgainst != "" {
		if utils.ComparePassword(*verifyAgainst, plain) {
			fmt.Println("COCOK — password ini menghasilkan hash yang sama dengan yang diberikan.")
		} else {
			fmt.Println("TIDAK COCOK — password ini BUKAN password asli di balik hash tersebut.")
			os.Exit(1)
		}
		return
	}

	hash, err := utils.HashPassword(plain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gagal membuat hash: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hash)
	fmt.Fprintln(os.Stderr, "\nCatatan: setiap kali dijalankan, hash yang dihasilkan BEDA walau passwordnya sama (bcrypt selalu pakai salt acak) — ini normal, keduanya tetap valid untuk password yang sama. Tempelkan hash di atas apa adanya ke kolom password_hash.")
}

func promptPassword(label string) string {
	fmt.Fprint(os.Stderr, label)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimRight(line, "\r\n")
}
