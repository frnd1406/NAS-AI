package operations

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// UpdaterStatus is the read-only host OS update state. It is sourced from a
// cache file that the privileged host updater (`nas-updater.sh status`, run by
// cron/systemd on the host) writes. The API container ONLY reads this file — it
// never runs the updater, SSH, or any host command — so the container stays
// unprivileged and can never trigger a system change. Safe to expose to the
// frontend: no secrets, no host paths, just the update posture.
type UpdaterStatus struct {
	Available           bool      `json:"available"`
	Host                string    `json:"host,omitempty"`
	TotalUpgradable     int       `json:"total_upgradable"`
	Security            int       `json:"security"`
	KernelRunning       string    `json:"kernel_running,omitempty"`
	KernelUpdatePending bool      `json:"kernel_update_pending"`
	RebootDue           bool      `json:"reboot_due"`
	ActionNeeded        bool      `json:"action_needed"`
	CachedAt            time.Time `json:"cached_at"`  // mtime of the cache file (how fresh the data is)
	UpdatedAt           time.Time `json:"updated_at"` // when the API read the cache
	Message             string    `json:"message,omitempty"`
}

// updaterStatusPath resolves the cache file location. It defaults to a path
// inside the already-mounted data volume, so no additional host bind mount is
// required (the host-root mount was deliberately removed in v2.1.1). Override
// with UPDATER_STATUS_FILE to point at a dedicated read-only mount instead.
func updaterStatusPath() string {
	if v := strings.TrimSpace(os.Getenv("UPDATER_STATUS_FILE")); v != "" {
		return v
	}
	return "/mnt/data/.system/updater-status"
}

// GetUpdaterStatus reads the host updater status cache. It never returns nil: if
// the cache is missing or unreadable it yields an {"available": false} snapshot
// so the endpoint stays a graceful 200 and the UI can render "status unknown"
// instead of erroring — matching the UPS endpoint's behaviour.
func (s *HardwareService) GetUpdaterStatus() *UpdaterStatus {
	path := updaterStatusPath()
	f, err := os.Open(path)
	if err != nil {
		s.logger.WithError(err).Debug("updater status cache unavailable")
		return &UpdaterStatus{
			Available: false,
			UpdatedAt: time.Now(),
			Message:   "Updater-Status nicht verfügbar (kein Cache)",
		}
	}
	defer f.Close()

	var modTime time.Time
	if fi, statErr := f.Stat(); statErr == nil {
		modTime = fi.ModTime()
	}

	vars := make(map[string]string, 8)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 4096), 64*1024)
	for sc.Scan() {
		if k, v, ok := parseUpdaterLine(sc.Text()); ok {
			vars[k] = v
		}
	}
	if err := sc.Err(); err != nil {
		s.logger.WithError(err).Debug("updater status cache read error")
		return &UpdaterStatus{
			Available: false,
			UpdatedAt: time.Now(),
			Message:   "Updater-Status nicht lesbar",
		}
	}
	return buildUpdaterStatus(vars, modTime)
}

// parseUpdaterLine parses one `KEY=VALUE` line from the updater status output.
// Blank lines and `#` comments are ignored. Only the first `=` splits, so a
// value may itself contain `=`.
func parseUpdaterLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	i := strings.IndexByte(line, '=')
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

// buildUpdaterStatus maps parsed KEY=VALUE pairs into an UpdaterStatus. It is a
// pure function (no I/O) so it is unit-testable; the yes/no fields follow the
// exact vocabulary emitted by `nas-updater.sh status`.
func buildUpdaterStatus(v map[string]string, modTime time.Time) *UpdaterStatus {
	st := &UpdaterStatus{
		Available:           true,
		Host:                v["HOST"],
		KernelRunning:       v["KERNEL_RUNNING"],
		KernelUpdatePending: updaterYes(v["KERNEL_UPDATE_PENDING"]),
		RebootDue:           updaterYes(v["REBOOT_DUE"]),
		ActionNeeded:        updaterYes(v["ACTION_NEEDED"]),
		CachedAt:            modTime,
		UpdatedAt:           time.Now(),
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v["TOTAL_UPGRADABLE"])); err == nil {
		st.TotalUpgradable = n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v["SECURITY"])); err == nil {
		st.Security = n
	}
	return st
}

// updaterYes reports whether a status token is the affirmative "yes" (case-insensitive).
func updaterYes(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "yes")
}
