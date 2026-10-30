package pengajuan_barang

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func pngBase64(t *testing.T, w, h int, ink bool, prefix string) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h)) // transparan penuh
	if ink {
		for x := 5; x < w-5; x++ {
			img.Set(x, h/2, color.NRGBA{0, 0, 0, 255})
			img.Set(x, h/2+1, color.NRGBA{0, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return prefix + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestDecodeTTD(t *testing.T) {
	if _, err := decodeTTD(pngBase64(t, 200, 80, true, "data:image/png;base64,")); err != nil {
		t.Fatalf("PNG bercoretan (data URL) harus diterima: %v", err)
	}
	if _, err := decodeTTD(pngBase64(t, 200, 80, true, "")); err != nil {
		t.Fatalf("base64 mentah harus diterima: %v", err)
	}
	tolak := map[string]string{
		"kanvas kosong":        pngBase64(t, 200, 80, false, ""),
		"bukan base64":         "%%%bukan-base64%%%",
		"string kosong":        "",
		"bukan gambar":         base64.StdEncoding.EncodeToString([]byte("<html><script>alert(1)</script></html>")),
		"terlalu besar piksel": pngBase64(t, 3000, 3000, true, ""),
	}
	for nama, raw := range tolak {
		if _, err := decodeTTD(raw); err == nil {
			t.Errorf("%s: seharusnya ditolak", nama)
		}
	}
}

func TestSaranQty(t *testing.T) {
	cases := []struct{ stok, min, want int }{{0, 10, 20}, {4, 10, 16}, {10, 10, 10}, {30, 10, 1}}
	for _, c := range cases {
		if got := saranQty(c.stok, c.min); got != c.want {
			t.Errorf("saranQty(%d,%d)=%d, want %d", c.stok, c.min, got, c.want)
		}
	}
}
