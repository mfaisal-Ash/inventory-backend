package activitylog

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		method, path string
		want         Info
	}{
		{"POST", "/barang-masuk", Info{ActionCreate, "barang-masuk", ""}},
		{"PUT", "/barang/12", Info{ActionUpdate, "barang", "12"}},
		{"DELETE", "/gudang/3", Info{ActionDelete, "gudang", "3"}},
		{"PATCH", "/barang/7/approve", Info{ActionApprove, "barang", "7"}},
		{"PATCH", "/pengajuan-barang/9/tolak", Info{ActionReject, "pengajuan-barang", "9"}},
		{"PATCH", "/barang-masuk/4/selesai", Info{ActionComplete, "barang-masuk", "4"}},
		{"PATCH", "/barang/2/protect", Info{ActionProtect, "barang", "2"}},
		{"POST", "/auth/login", Info{ActionLogin, "auth", ""}},
		{"POST", "/auth/verify-otp", Info{ActionLogin, "auth", ""}},
		{"POST", "/auth/logout", Info{ActionLogout, "auth", ""}},
		{"DELETE", "/auth/sessions/5", Info{ActionDelete, "auth", ""}},
		{"PATCH", "/barang-keluar/8/items/3/spesifikasi", Info{ActionUpdate, "barang-keluar", "8"}},
	}
	for _, c := range cases {
		got := Classify(c.method, c.path)
		if got != c.want {
			t.Errorf("Classify(%s,%s) = %+v, want %+v", c.method, c.path, got, c.want)
		}
	}
}

func TestShouldSkip(t *testing.T) {
	for _, p := range []string{"/auth/refresh", "/notifications/3/read", "/activity-logs/track", "/app/update"} {
		if !ShouldSkip(p) {
			t.Errorf("%s harus di-skip", p)
		}
	}
	for _, p := range []string{"/aset", "/auth/login", "/barang/1", "/application"} {
		if ShouldSkip(p) {
			t.Errorf("%s tidak boleh di-skip", p)
		}
	}
}

func TestDescribe(t *testing.T) {
	if got := Describe(ActionCreate, "barang-masuk", "", true, ""); got != "Menambah Barang Masuk" {
		t.Errorf("got %q", got)
	}
	if got := Describe(ActionUpdate, "barang", "12", true, ""); got != "Mengubah Kelola Barang (ID 12)" {
		t.Errorf("got %q", got)
	}
	if got := Describe(ActionLogin, "auth", "", true, ""); got != "Login" {
		t.Errorf("got %q", got)
	}
	got := Describe(ActionDelete, "gudang", "3", false, "dilindungi")
	if !strings.Contains(got, "GAGAL: dilindungi") {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeBody(t *testing.T) {
	in := []byte(`{"username":"budi","password":"rahasia","nested":{"otp_code":"123456","nama":"x"},"list":[{"token":"abc"}]}`)
	out := SanitizeBody(in, 2000)
	for _, secret := range []string{"rahasia", "123456", "abc"} {
		if strings.Contains(out, secret) {
			t.Errorf("body masih memuat %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, "budi") || !strings.Contains(out, "***") {
		t.Errorf("hasil tidak sesuai: %s", out)
	}
	if SanitizeBody([]byte("bukan json"), 100) != "" {
		t.Error("body non-JSON harus dikosongkan")
	}
	if got := SanitizeBody(in, 10); len([]rune(got)) > 11 {
		t.Errorf("tidak terpotong: %q", got)
	}
}

func TestUsernameFromBody(t *testing.T) {
	if got := UsernameFromBody([]byte(`{"username":" admin ","password":"x"}`)); got != "admin" {
		t.Errorf("got %q", got)
	}
}
