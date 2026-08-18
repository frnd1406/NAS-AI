package operations

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestParseUpdaterLine(t *testing.T) {
	cases := []struct {
		line    string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{"HOST=nasserver", "HOST", "nasserver", true},
		{"TOTAL_UPGRADABLE=3", "TOTAL_UPGRADABLE", "3", true},
		{"KERNEL_RUNNING=6.6.31+rpt-rpi-2712", "KERNEL_RUNNING", "6.6.31+rpt-rpi-2712", true},
		{"  REBOOT_DUE = yes ", "REBOOT_DUE", "yes", true},
		{"# a comment", "", "", false},
		{"", "", "", false},
		{"NOEQUALS", "", "", false},
		{"=leadingeq", "", "", false},
	}
	for _, c := range cases {
		k, v, ok := parseUpdaterLine(c.line)
		if ok != c.wantOK || k != c.wantKey || v != c.wantVal {
			t.Errorf("parseUpdaterLine(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.line, k, v, ok, c.wantKey, c.wantVal, c.wantOK)
		}
	}
}

func TestBuildUpdaterStatus(t *testing.T) {
	mod := time.Unix(1_700_000_000, 0)
	st := buildUpdaterStatus(map[string]string{
		"HOST":                  "nasserver",
		"TOTAL_UPGRADABLE":      "5",
		"SECURITY":              "2",
		"KERNEL_RUNNING":        "6.6.31+rpt-rpi-2712",
		"KERNEL_UPDATE_PENDING": "yes",
		"REBOOT_DUE":            "no",
		"ACTION_NEEDED":         "yes",
	}, mod)

	if !st.Available {
		t.Error("expected Available=true")
	}
	if st.Host != "nasserver" || st.KernelRunning != "6.6.31+rpt-rpi-2712" {
		t.Errorf("host/kernel = %q/%q", st.Host, st.KernelRunning)
	}
	if st.TotalUpgradable != 5 || st.Security != 2 {
		t.Errorf("counts = %d upgradable / %d security, want 5/2", st.TotalUpgradable, st.Security)
	}
	if !st.KernelUpdatePending || st.RebootDue || !st.ActionNeeded {
		t.Errorf("flags = kupd:%v reboot:%v action:%v, want true/false/true",
			st.KernelUpdatePending, st.RebootDue, st.ActionNeeded)
	}
	if !st.CachedAt.Equal(mod) {
		t.Errorf("CachedAt = %v, want %v", st.CachedAt, mod)
	}
}

func TestBuildUpdaterStatusDefaults(t *testing.T) {
	// Missing/garbage numeric fields must default to 0, not crash; absent
	// yes/no fields must be false.
	st := buildUpdaterStatus(map[string]string{
		"HOST":             "nasserver",
		"TOTAL_UPGRADABLE": "notanumber",
	}, time.Time{})
	if !st.Available {
		t.Error("expected Available=true even with sparse data")
	}
	if st.TotalUpgradable != 0 || st.Security != 0 {
		t.Errorf("defaults = %d/%d, want 0/0", st.TotalUpgradable, st.Security)
	}
	if st.KernelUpdatePending || st.RebootDue || st.ActionNeeded {
		t.Error("absent yes/no flags must be false")
	}
}

func TestUpdaterYes(t *testing.T) {
	for _, s := range []string{"yes", "YES", " Yes "} {
		if !updaterYes(s) {
			t.Errorf("updaterYes(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"no", "", "y", "true", "1"} {
		if updaterYes(s) {
			t.Errorf("updaterYes(%q) = true, want false", s)
		}
	}
}

func TestGetUpdaterStatusMissingCache(t *testing.T) {
	// Pointing at a non-existent file yields a graceful unavailable snapshot,
	// never nil and never a panic.
	t.Setenv("UPDATER_STATUS_FILE", filepath.Join(t.TempDir(), "does-not-exist"))
	svc := NewHardwareService(logrus.New())
	st := svc.GetUpdaterStatus()
	if st == nil {
		t.Fatal("GetUpdaterStatus returned nil")
	}
	if st.Available {
		t.Error("expected Available=false for missing cache")
	}
	if st.Message == "" {
		t.Error("expected a Message explaining the missing cache")
	}
}

func TestGetUpdaterStatusReadsCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "updater-status")
	content := "HOST=nasserver\nTOTAL_UPGRADABLE=4\nSECURITY=1\n" +
		"KERNEL_RUNNING=6.6.31+rpt-rpi-2712\nKERNEL_UPDATE_PENDING=no\n" +
		"REBOOT_DUE=yes\nACTION_NEEDED=yes\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UPDATER_STATUS_FILE", path)

	svc := NewHardwareService(logrus.New())
	st := svc.GetUpdaterStatus()
	if !st.Available {
		t.Fatal("expected Available=true for present cache")
	}
	if st.TotalUpgradable != 4 || st.Security != 1 {
		t.Errorf("counts = %d/%d, want 4/1", st.TotalUpgradable, st.Security)
	}
	if !st.RebootDue || !st.ActionNeeded || st.KernelUpdatePending {
		t.Errorf("flags reboot:%v action:%v kupd:%v, want true/true/false",
			st.RebootDue, st.ActionNeeded, st.KernelUpdatePending)
	}
	if st.CachedAt.IsZero() {
		t.Error("expected CachedAt to be set from the file mtime")
	}
}
