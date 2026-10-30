-- Hapus SEMUA kolom harga/nilai uang pada database yang SUDAH berjalan.
-- (GORM AutoMigrate hanya menambah kolom; ia tidak pernah menghapus kolom lama.
--  Kolom `keluar_awal` di tabel barang ditambahkan otomatis oleh AutoMigrate saat
--  backend baru dijalankan.)
--
-- URUTAN: 1) deploy backend baru (jalankan sekali agar AutoMigrate jalan)
--         2) BACKUP database
--         3) psql -d stockrsd -f deploy/sql/2026-10-26-hapus-kolom-harga.sql
--
-- Tidak bisa dibatalkan selain dari backup.
BEGIN;

ALTER TABLE barang               DROP COLUMN IF EXISTS harga_beli;
ALTER TABLE barang_masuk_items   DROP COLUMN IF EXISTS harga_satuan;
ALTER TABLE purchase_order_items DROP COLUMN IF EXISTS harga_satuan;
ALTER TABLE purchase_order_items DROP COLUMN IF EXISTS subtotal;
ALTER TABLE purchase_orders      DROP COLUMN IF EXISTS total_estimasi;
ALTER TABLE assets               DROP COLUMN IF EXISTS nilai_aset;

COMMIT;
