package version_test

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/HungNth/upit/internal/version"
)

func TestVersionInfoDefaults(t *testing.T) {
	info := version.Get()

	if info.Version != "0.9.0" {
		t.Errorf("Version = %q, want %q", info.Version, "0.9.0")
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	if info.Commit == "" {
		t.Error("Commit should not be empty")
	}
	if info.Date == "" {
		t.Error("Date should not be empty")
	}

	human := info.String()
	if !strings.HasPrefix(human, "upit 0.9.0 (commit: ") || !strings.Contains(human, ", built: ") || !strings.HasSuffix(human, ")") {
		t.Errorf("String() = %q, want format 'upit 0.9.0 (commit: <sha>, built: <date>)'", human)
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal(info) failed: %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal(json) failed: %v", err)
	}
	for _, key := range []string{"version", "commit", "date", "goVersion", "os", "arch"} {
		if val, ok := decoded[key]; !ok || val == "" {
			t.Errorf("JSON missing or empty key %q: %s", key, string(data))
		}
	}
}

func TestVersionInfoFormattingWithUnknowns(t *testing.T) {
	custom := version.Info{
		Version:   "0.9.0",
		Commit:    "unknown",
		Date:      "unknown",
		GoVersion: "go1.23.0",
		OS:        "linux",
		Arch:      "amd64",
	}

	want := "upit 0.9.0 (commit: unknown, built: unknown)"
	if got := custom.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	customWithTime := version.Info{
		Version:   "0.9.0",
		Commit:    "e62acf3-dirty",
		Date:      "2026-10-06T12:34:56Z",
		GoVersion: "go1.23.0",
		OS:        "linux",
		Arch:      "amd64",
	}
	wantTime := "upit 0.9.0 (commit: e62acf3-dirty, built: 2026-10-06)"
	if got := customWithTime.String(); got != wantTime {
		t.Errorf("String() = %q, want %q", got, wantTime)
	}
}
