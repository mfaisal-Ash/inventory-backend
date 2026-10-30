package neraca

import (
	"reflect"
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time { return time.Date(y, m, day, 0, 0, 0, 0, time.UTC) }

func TestParseGranularity(t *testing.T) {
	for in, want := range map[string]Granularity{"": Harian, "Harian": Harian, "weekly": Mingguan, "BULANAN": Bulanan, "year": Tahunan} {
		if got, ok := ParseGranularity(in); !ok || got != want {
			t.Errorf("%q -> %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := ParseGranularity("hourly"); ok {
		t.Error("hourly harus ditolak")
	}
}

func TestBucketStartDanBuildBuckets(t *testing.T) {
	// 28 Okt 2026 = Rabu; minggu mulai Senin 26 Okt.
	if got := BucketStart(d(2026, 10, 28), Mingguan); !got.Equal(d(2026, 10, 26)) {
		t.Fatalf("awal minggu salah: %v", got)
	}
	// Minggu (1 Nov) tetap masuk minggu yang dimulai Senin 26 Okt.
	if got := BucketStart(d(2026, 11, 1), Mingguan); !got.Equal(d(2026, 10, 26)) {
		t.Fatalf("hari Minggu harus masuk minggu sebelumnya: %v", got)
	}
	b := BuildBuckets(d(2026, 8, 15), d(2026, 10, 27), Bulanan)
	want := []time.Time{d(2026, 8, 1), d(2026, 9, 1), d(2026, 10, 1)}
	if !reflect.DeepEqual(b, want) {
		t.Fatalf("bucket bulanan: %v", b)
	}
	if n := len(BuildBuckets(d(2020, 1, 1), d(2026, 1, 1), Harian)); n <= MaxBuckets {
		t.Fatalf("rentang terlalu lebar harus melewati MaxBuckets agar bisa ditolak, n=%d", n)
	}
}

func TestDefaultRange(t *testing.T) {
	today := d(2026, 10, 27)
	for g, wantN := range map[Granularity]int{Harian: 30, Mingguan: 12, Bulanan: 12, Tahunan: 5} {
		dari, sampai := DefaultRange(today, g)
		if n := len(BuildBuckets(dari, sampai, g)); n != wantN {
			t.Errorf("%s: %d titik, want %d", g, n, wantN)
		}
	}
}

func TestHitungSaldoSeries(t *testing.T) {
	// Gudang 1: stok sekarang 100. Harian 3 hari: 25,26,27 Okt.
	// 26 Okt: masuk +30. 27 Okt: keluar -10. 28 Okt (setelah rentang): keluar -5.
	// Saldo akhir 27 = 100 - (-5) = 105; akhir 26 = 105 - (-10) = 115; akhir 25 = 115 - 30 = 85.
	buckets := BuildBuckets(d(2026, 10, 25), d(2026, 10, 27), Harian)
	got := HitungSaldoSeries(buckets, Harian, map[uint]int{1: 100}, []DeltaDay{
		{1, d(2026, 10, 26), 30}, {1, d(2026, 10, 27), -10}, {1, d(2026, 10, 28), -5},
		{1, d(2026, 10, 1), 999}, // sebelum rentang: diabaikan
	})
	if want := []int{85, 115, 105}; !reflect.DeepEqual(got[1], want) {
		t.Fatalf("saldo = %v, want %v", got[1], want)
	}
}

func TestHitungSaldoSeries_Bulanan_DanGudangTanpaStok(t *testing.T) {
	buckets := BuildBuckets(d(2026, 9, 1), d(2026, 10, 27), Bulanan)
	got := HitungSaldoSeries(buckets, Bulanan, map[uint]int{1: 50}, []DeltaDay{
		{1, d(2026, 10, 5), 20}, // Okt: masuk 20
		{2, d(2026, 9, 10), 8},  // gudang 2 tak punya stok sekarang (habis)
		{2, d(2026, 10, 10), -8},
	})
	if want := []int{30, 50}; !reflect.DeepEqual(got[1], want) {
		t.Fatalf("gudang 1 = %v, want %v", got[1], want)
	}
	// Gudang 2: sekarang 0; Okt keluar 8 -> akhir Sep = 0 - (-8) = 8; akhir Okt = 0.
	if want := []int{8, 0}; !reflect.DeepEqual(got[2], want) {
		t.Fatalf("gudang 2 = %v, want %v", got[2], want)
	}
}
