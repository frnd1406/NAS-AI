package operations

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// UPSInfo is the runtime state of the UPS as reported by the local NUT server
// (upsd). It is designed to be safe to expose to the frontend: no credentials,
// no host paths, just the power/battery state needed to drive an alert banner.
type UPSInfo struct {
	Available      bool      `json:"available"`
	Name           string    `json:"name"`
	State          string    `json:"state"` // online | on_battery | low_battery | unknown | unavailable
	RawStatus      string    `json:"raw_status,omitempty"`
	OnBattery      bool      `json:"on_battery"`
	LowBattery     bool      `json:"low_battery"`
	BatteryCharge  *int      `json:"battery_charge,omitempty"`
	BatteryVoltage *float64  `json:"battery_voltage,omitempty"`
	InputVoltage   *float64  `json:"input_voltage,omitempty"`
	OutputVoltage  *float64  `json:"output_voltage,omitempty"`
	Load           *int      `json:"load,omitempty"`
	Model          string    `json:"model,omitempty"`
	Type           string    `json:"type,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
	Message        string    `json:"message,omitempty"`
}

// upsName / upsAddr resolve the UPS identity from the environment, defaulting to
// the local deployment (Green Cell PowerProof via NUT on the host network).
func upsName() string {
	if v := strings.TrimSpace(os.Getenv("UPS_NAME")); v != "" {
		return v
	}
	return "greencell"
}

func upsAddr() string {
	if v := strings.TrimSpace(os.Getenv("UPS_ADDR")); v != "" {
		return v
	}
	return "127.0.0.1:3493"
}

// GetUPSInfo reads the current UPS state from the local NUT server. It never
// returns nil: on any error it yields an "unavailable" snapshot so the endpoint
// stays a plain 200 and the UI can render a graceful "no UPS" state.
func (s *HardwareService) GetUPSInfo() *UPSInfo {
	name, addr := upsName(), upsAddr()
	info, err := fetchUPSStatus(name, addr, 3*time.Second)
	if err != nil {
		s.logger.WithError(err).Debug("UPS status unavailable")
		return &UPSInfo{
			Available: false,
			Name:      name,
			State:     "unavailable",
			UpdatedAt: time.Now(),
			Message:   "UPS/NUT nicht erreichbar",
		}
	}
	return info
}

// fetchUPSStatus speaks the plain-text NUT protocol over TCP and parses the UPS
// variables. It is strictly read-only: it only issues LIST VAR and LOGOUT, so it
// respects the "container never runs privileged host commands" rule.
func fetchUPSStatus(name, addr string, timeout time.Duration) (*UPSInfo, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := fmt.Fprintf(conn, "LIST VAR %s\n", name); err != nil {
		return nil, err
	}

	vars := make(map[string]string, 32)
	started := false
	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, 0, 4096), 256*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		switch {
		case strings.HasPrefix(line, "ERR "):
			return nil, fmt.Errorf("nut error: %s", strings.TrimPrefix(line, "ERR "))
		case strings.HasPrefix(line, "BEGIN LIST VAR"):
			started = true
		case strings.HasPrefix(line, "END LIST VAR"):
			_, _ = fmt.Fprint(conn, "LOGOUT\n")
			return buildUPSInfo(name, vars), nil
		case started && strings.HasPrefix(line, "VAR "):
			if k, v, ok := parseUPSVarLine(line); ok {
				vars[k] = v
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("nut: incomplete response for %q", name)
}

// parseUPSVarLine parses a NUT `VAR <ups> <key> "<value>"` line into key/value.
// The value is quoted and may itself contain spaces (e.g. "offline / line interactive").
func parseUPSVarLine(line string) (string, string, bool) {
	rest := strings.TrimPrefix(line, "VAR ")
	fields := strings.SplitN(rest, " ", 3)
	if len(fields) < 3 {
		return "", "", false
	}
	key, val := fields[1], fields[2]
	if i, j := strings.Index(val, "\""), strings.LastIndex(val, "\""); i >= 0 && j > i {
		val = val[i+1 : j]
	}
	return key, val, true
}

// buildUPSInfo maps raw NUT variables into a UPSInfo, deriving the semantic
// state from ups.status (OL online, OB on battery, LB low battery).
func buildUPSInfo(name string, v map[string]string) *UPSInfo {
	info := &UPSInfo{
		Available: true,
		Name:      name,
		RawStatus: v["ups.status"],
		Model:     upsFirstNonEmpty(v["device.model"], v["ups.model"]),
		Type:      v["ups.type"],
		UpdatedAt: time.Now(),
	}

	tokens := strings.Fields(v["ups.status"])
	for _, t := range tokens {
		switch t {
		case "OB":
			info.OnBattery = true
		case "LB":
			info.LowBattery = true
		}
	}
	switch {
	case info.LowBattery:
		info.State = "low_battery"
	case info.OnBattery:
		info.State = "on_battery"
	case upsContains(tokens, "OL"):
		info.State = "online"
	default:
		info.State = "unknown"
	}

	if n, err := strconv.Atoi(strings.TrimSpace(v["battery.charge"])); err == nil {
		info.BatteryCharge = &n
	}
	if n, err := strconv.Atoi(strings.TrimSpace(v["ups.load"])); err == nil {
		info.Load = &n
	}
	info.BatteryVoltage = parseUPSFloat(v["battery.voltage"])
	info.InputVoltage = parseUPSFloat(v["input.voltage"])
	info.OutputVoltage = parseUPSFloat(v["output.voltage"])
	return info
}

func parseUPSFloat(s string) *float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return &f
	}
	return nil
}

func upsFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func upsContains(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
