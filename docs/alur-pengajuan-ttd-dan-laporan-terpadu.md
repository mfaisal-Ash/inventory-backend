# Alur Restock (Pengajuan + TTD Digital), Laporan Terpadu, dan Kendali User

Dokumen serah-terima untuk tim frontend. Semua endpoint berada di bawah `/stockrsd`.

## 1. Alur stok menipis → pengajuan → TTD → dua pihak setuju → barang masuk

```
Ringkasan Stok (stok <= minimum)
  └─ GET  /pengajuan-barang/stok-menipis        daftar barang + saran qty + pengajuan_aktif
       └─ POST /pengajuan-barang                 jenis="masuk" + ttd_pencatat   (pihak 1 tanda tangan)
            └─ PATCH /pengajuan-barang/:id/setujui   ttd (pihak 2 tanda tangan, orang BERBEDA)
                 └─ otomatis: dokumen Barang Masuk (draft) dibuat
                      └─ staf gudang lengkapi harga & selesaikan di halaman Barang Masuk
```

### GET `/pengajuan-barang/stok-menipis`
Izin: `pengajuan_barang.view`. Mendukung `?page=&limit=&search=`.
Hanya barang aktif & berstatus disetujui dengan `stok_minimum > 0` dan `stok <= stok_minimum`.

| field | arti |
|---|---|
| `saran_qty` | isi ulang sampai 2x stok minimum (minimal 1); hanya saran |
| `pengajuan_aktif` | nomor pengajuan "masuk" yang masih menunggu — kalau terisi, tampilkan badge dan jangan ajukan dua kali |

### POST `/pengajuan-barang` (pihak 1)
Body seperti biasa, dengan `"jenis": "masuk"` dan tambahan:
- `ttd_pencatat` — **wajib untuk jenis masuk**. Data URL PNG dari kanvas tanda tangan (`data:image/png;base64,...`), maks 300 KB, harus ada coretan (kanvas kosong ditolak).
- `nama_pencatat`, `jabatan_pencatat` — opsional; nama default = nama lengkap akun.

### PATCH `/pengajuan-barang/:id/setujui` (pihak 2)
Body: `{ "nama_ga", "jabatan_ga", "catatan", "ttd" }`. `ttd` wajib untuk jenis masuk.
Ditolak (HTTP 409/400) kalau:
1. `ttd` kosong/tidak valid,
2. penyetuju **sama** dengan pengaju,
3. pengaju belum tanda tangan, **atau isi pengajuan berubah** sejak ia tanda tangan.

Aturan 3 memakai sidik jari SHA-256 isi pengajuan (jenis, gudang, tanggal, keperluan, barang+qty).
`PUT /pengajuan-barang/:id` membuang tanda tangan lama, jadi pengaju **harus mengirim `ttd_pencatat` lagi** saat mengedit.

### GET `/pengajuan-barang/:id/ttd/:peran`
`peran` = `pencatat` | `penyetuju`. Mengembalikan gambar PNG (perlu login + izin `view`).
Metadata tanda tangan (nama, jabatan, waktu, IP, hash) ada di `tanda_tangan[]` pada detail/daftar; **gambarnya tidak** ikut JSON.
Pakai ini untuk menampilkan dua kolom tanda tangan di pratinjau/cetak.

Jenis pengajuan selain `masuk` tetap berjalan seperti sebelumnya (TTD opsional, tanpa larangan setuju sendiri).

## 2. Laporan terpadu
Tipe baru: `tipe=Ringkasan Gudang` pada `/laporan/tipe`, `/laporan/preview`, dan `/laporan/export` (Excel/PDF/Docs).
Kolom: `Bagian | Indikator | Nilai | Keterangan`. Bagian: Sorotan (jadi kartu ringkasan), Informasi, Inventaris, Perlu Restock (maks 15 barang paling kritis), Pengajuan Barang, Barang Masuk, Barang Keluar, Barang Rusak, Stock Opname, Tracking Aset, FIFO/FEFO.
Angkanya dibangun dari builder laporan masing-masing, jadi sama dengan laporan tunggalnya. `/laporan/tipe` kini berurutan tetap dengan laporan terpadu di paling atas.
Untuk sidebar: satu menu "Laporan" yang membuka tipe ini sebagai default; laporan lama tetap ada sebagai rincian.

## 3. Kabel jadi tab di Kelola Barang
Endpoint kabel tidak berubah, tetapi izinnya kini mengikuti modul **`kelola_barang`** (sebelumnya `barang_keluar`).
Role kustom yang punya `barang_keluar` tapi tidak `kelola_barang` perlu diberi izin Kelola Barang agar tab kabelnya terbuka.

## 4. Kendali super admin & akses hanya dari gudang (sudah ada di backend)
Tidak ada perubahan kode; frontend perlu menampilkan kontrolnya:
- `POST /users/:id/reset-password` `{new_password}` — reset + cabut semua sesi (super admin saja; target admin/karyawan).
- `POST /users/:id/logout-all` — paksa logout.
- `GET /users/:id/sessions`, `DELETE /users/:id/sessions/:sessionId`.
- `PUT /users/:id` — ubah role/aktif; akun admin hanya bisa diubah super admin.
- `GET/PUT /users/access-policy` — `{karyawan_networks: ["192.168.1.0/24"], admin_gudang_only: true}`. Karyawan selalu dibatasi ke daftar ini (daftar kosong = tanpa batas); admin ikut dibatasi bila `admin_gudang_only`; super admin tidak pernah dibatasi.
- Respons `GET access-policy` menyertakan `client_ip` & `client_ip_allowed` agar UI bisa memperingatkan sebelum menyimpan.

**Syarat server:** IP klien dibaca lewat reverse proxy. Pastikan `TRUSTED_PROXIES` berisi IP proxy (nginx) Anda, kalau tidak semua pengguna terbaca berasal dari IP proxy.

## 5. Menu Analisa Data (4 endpoint baru, izin `dashboard.view`)

Semua tanggal berformat `YYYY-MM-DD`.

### GET `/dashboard/stok-grafik` — grafik stok per gudang
Query: `granularity=daily|weekly|monthly|yearly` (juga diterima: harian/mingguan/bulanan/tahunan), `dari`, `sampai`, `gudang_id`.
Tanpa `dari/sampai`: 30 hari, 12 minggu, 12 bulan, atau 5 tahun terakhir. Maksimal 400 titik (lebih dari itu HTTP 400).
Respons: `periode[]` (`label`, `mulai`, `selesai`), `series[]` (satu per gudang: `gudang_id`, `nama_gudang`, `saldo[]` sejajar dengan `periode`), dan `total[]`.
`saldo` = stok di **akhir** periode, dihitung mundur dari stok sistem saat ini memakai mutasi yang sama dengan neraca (masuk, keluar, kabel, retur, opname). Titik terakhir selalu sama dengan stok yang tampil di aplikasi. Minggu dimulai hari Senin. Total menjumlahkan angka stok apa adanya (kabel dalam meter, barang lain dalam unit — sama seperti neraca).

### GET `/dashboard/riwayat` — riwayat terpadu
Query: `jenis=semua` atau gabungan `masuk,keluar,rusak,retur,pengadaan`, `dari`, `sampai`, `gudang_id`, `status`, `search`, `page`, `limit`.
Respons: `data.items[]` (`jenis`, `id`, `nomor`, `tanggal`, `gudang_id`, `nama_gudang`, `barang`, `qty`, `status`, `keterangan`), `data.ringkasan[]` (jumlah dokumen & total qty per jenis, mengikuti filter), dan `meta` paginasi. Terbaru di atas.
- `masuk` / `keluar`: dokumen barang masuk/keluar (semua status; saring dengan `status`).
- `retur`: retur & penggantian.
- `pengadaan`: pengajuan barang masuk ke gudang (alur restock). Dokumen barang masuk hasil persetujuannya tercatat di `masuk`.
- `rusak`: tabel barang rusak tidak menyimpan gudang, jadi gudangnya diambil dari gudang asal barang (`barang.nama_gudang`). Laporan rusak tanpa barang terhubung tampil tanpa gudang dan tidak ikut filter `gudang_id`.

### GET `/dashboard/stok-gudang` — tab "Stok Barang"
Query: `dari`, `sampai` (bawaan: awal bulan s.d. hari ini), `gudang_id`, `search`.
Respons: `gudang[]` (per gudang: `total` neraca, `nilai_akhir`, `jumlah_barang`, `rows[]` per barang dengan saldo awal, masuk, keluar, pakai kabel, retur ganti, opname, **saldo akhir**, `harga_beli`, `nilai_akhir`), `grand_total`, `nilai_akhir`, `retur`, `tak_terlacak`.
Hasil akhir = saldo akhir neraca; nilai persediaan = saldo akhir x harga beli.

### GET `/dashboard/aset-gudang` — tab "Asset Barang"
Query: `gudang_id`, `status`, `jenis`, `search`, `detail=true` (menyertakan daftar aset, maks 500 per gudang).
Respons: `total_aset`, `nilai_aset`, dan `gudang[]` dengan `total_aset`, `nilai_aset`, `per_status[]`, `per_jenis[]`, `items[]` (kalau `detail=true`).
Aset dihitung per unit (bukan mutasi stok), jadi neraca tidak berlaku untuk aset; nilainya dari kolom `nilai_aset`.
