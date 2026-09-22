package orphan

import (
	"path/filepath"
	"testing"
)

func testDetector() *Detector {
	d := &Detector{
		installed: map[string]bool{"chrome": true, "spotify": true},
		roots:     []string{`C:\Users\bud\AppData\Local`, `C:\Users\bud\AppData\Roaming`},
		loaded:    true,
	}
	return d
}

func TestIsOrphan_FlagsDepartedApp(t *testing.T) {
	d := testDetector()
	res := d.IsOrphan(filepath.Join(`C:\Users\bud\AppData\Local`, "SomeDepartedApp", "data.bin"))
	if res == nil {
		t.Fatal("seharusnya terdeteksi orphan")
	}
	if res.AppName != "SomeDepartedApp" {
		t.Errorf("AppName = %q, want %q", res.AppName, "SomeDepartedApp")
	}
}

func TestIsOrphan_InstalledAppIsSafe(t *testing.T) {
	d := testDetector()
	p := filepath.Join(`C:\Users\bud\AppData\Local`, "Google", "Chrome", "User Data", "Default")
	if res := d.IsOrphan(filepath.Join(p, "cache.bin")); res != nil {
		t.Errorf("segmen milik aplikasi terpasang harus aman, got %+v", res)
	}
}

func TestIsOrphan_CompanyProductLayoutCaseInsensitive(t *testing.T) {
	d := testDetector()
	p := filepath.Join(`C:\Users\bud\AppData\Roaming`, "Google", "Chrome")
	if res := d.IsOrphan(filepath.Join(p, "data")); res != nil {
		t.Errorf("case berbeda tetap harus aman, got %+v", res)
	}
}

func TestIsOrphan_CompanySegmentOnlyStillOrphan(t *testing.T) {
	// vendor di segmen pertama (Google) tidak terpasang, produk (Chrome) juga
	// tidak → tetap orphan dengan nama vendor.
	d := testDetector()
	res := d.IsOrphan(filepath.Join(`C:\Users\bud\AppData\Local`, "Google", "Chromium", "data.bin"))
	if res == nil {
		t.Fatal("vendor tanpa aplikasi terpasang seharusnya orphan")
	}
}

func TestIsOrphan_KnownSafeFolder(t *testing.T) {
	d := testDetector()
	p := filepath.Join(`C:\Users\bud\AppData\Local`, "Temp", "x.bin")
	if res := d.IsOrphan(p); res != nil {
		t.Errorf("folder milik sistem harus aman, got %+v", res)
	}
}

func TestIsOrphan_UnconfiguredReturnsNil(t *testing.T) {
	d := &Detector{}
	if res := d.IsOrphan(`C:\Users\bud\AppData\Local\App\data.bin`); res != nil {
		t.Errorf("detector belum load harusnya nil, got %+v", res)
	}
}

func TestConfigured(t *testing.T) {
	if !testDetector().Configured() {
		t.Error("index terisi + loaded harusnya Configured true")
	}
	empty := &Detector{}
	if empty.Configured() {
		t.Error("tanpa load harusnya Configured false")
	}
}