package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"gorm.io/gorm"
)

var barangIDPattern = regexp.MustCompile(`^RSD-(\d{8})-(\d+)$`)

type assignment struct {
	ID      uint
	Tanggal time.Time
	OldID   string
	NewID   string
	Kode    string
	Nama    string
}

func main() {
	apply := flag.Bool("apply", false, "simpan perubahan ke database; tanpa flag ini hanya dry-run")
	flag.Parse()

	db := config.NewDatabase(config.Load())
	var all []model.Barang
	if err := db.Order("tanggal ASC, id ASC").Find(&all).Error; err != nil {
		log.Fatalf("gagal mengambil barang: %v", err)
	}

	// Ambil nomor terbesar yang sudah dipakai per tanggal agar ID existing tidak tertabrak.
	maxByDate := map[string]int{}
	for _, b := range all {
		m := barangIDPattern.FindStringSubmatch(b.IDBarang)
		if len(m) != 3 {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil {
			continue
		}
		if n > maxByDate[m[1]] {
			maxByDate[m[1]] = n
		}
	}

	var plan []assignment
	for _, b := range all {
		if b.IDBarang != "" {
			continue
		}
		dateKey := b.Tanggal.Format("02012006")
		maxByDate[dateKey]++
		plan = append(plan, assignment{ID: b.ID, Tanggal: b.Tanggal, OldID: b.IDBarang, NewID: fmt.Sprintf("RSD-%s-%02d", dateKey, maxByDate[dateKey]), Kode: b.KodeBarang, Nama: b.Nama})
	}

	file, err := os.Create(fmt.Sprintf("backfill-barang-id-%s.csv", time.Now().Format("20060102-150405")))
	if err != nil {
		log.Fatalf("gagal membuat backup: %v", err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()
	_ = w.Write([]string{"database_id", "tanggal", "barang_id_lama", "barang_id_baru", "kode_barang", "nama", "mode"})
	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	for _, p := range plan {
		_ = w.Write([]string{fmt.Sprint(p.ID), p.Tanggal.Format("2006-01-02"), p.OldID, p.NewID, p.Kode, p.Nama, mode})
	}

	if !*apply {
		fmt.Printf("DRY-RUN: %d BarangID kosong akan diisi. Daftar: %s\n", len(plan), file.Name())
		return
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, p := range plan {
			if err := tx.Model(&model.Barang{}).Where("id = ? AND (id_barang IS NULL OR id_barang = '')", p.ID).Update("id_barang", p.NewID).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Fatalf("backfill dibatalkan: %v", err)
	}
	fmt.Printf("SELESAI: %d BarangID berhasil diisi. Backup: %s\n", len(plan), file.Name())
}
