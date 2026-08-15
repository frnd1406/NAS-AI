package operations

import "testing"

func TestParseUPSVarLine(t *testing.T) {
	cases := []struct {
		line    string
		wantKey string
		wantVal string
		wantOK  bool
	}{
		{`VAR greencell ups.status "OL"`, "ups.status", "OL", true},
		{`VAR greencell battery.charge "100"`, "battery.charge", "100", true},
		{`VAR greencell ups.type "offline / line interactive"`, "ups.type", "offline / line interactive", true},
		{`VAR greencell input.voltage "230.8"`, "input.voltage", "230.8", true},
		{`VAR greencell broken`, "", "", false},
	}
	for _, c := range cases {
		k, v, ok := parseUPSVarLine(c.line)
		if ok != c.wantOK || k != c.wantKey || v != c.wantVal {
			t.Errorf("parseUPSVarLine(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.line, k, v, ok, c.wantKey, c.wantVal, c.wantOK)
		}
	}
}

func TestBuildUPSInfoState(t *testing.T) {
	cases := []struct {
		name      string
		status    string
		wantState string
		wantOB    bool
		wantLB    bool
	}{
		{"online", "OL", "online", false, false},
		{"online charging", "OL CHRG", "online", false, false},
		{"on battery", "OB", "on_battery", true, false},
		{"on battery discharging", "OB DISCHRG", "on_battery", true, false},
		{"low battery", "OB LB", "low_battery", true, true},
		{"empty", "", "unknown", false, false},
	}
	for _, c := range cases {
		info := buildUPSInfo("greencell", map[string]string{"ups.status": c.status})
		if info.State != c.wantState || info.OnBattery != c.wantOB || info.LowBattery != c.wantLB {
			t.Errorf("%s: state=%q ob=%v lb=%v, want state=%q ob=%v lb=%v",
				c.name, info.State, info.OnBattery, info.LowBattery, c.wantState, c.wantOB, c.wantLB)
		}
		if !info.Available {
			t.Errorf("%s: expected Available=true", c.name)
		}
	}
}

func TestBuildUPSInfoNumbers(t *testing.T) {
	info := buildUPSInfo("greencell", map[string]string{
		"ups.status":      "OL",
		"battery.charge":  "84",
		"battery.voltage": "13.53",
		"input.voltage":   "230.8",
		"ups.load":        "0",
		"ups.type":        "offline / line interactive",
	})
	if info.BatteryCharge == nil || *info.BatteryCharge != 84 {
		t.Errorf("battery charge = %v, want 84", info.BatteryCharge)
	}
	if info.InputVoltage == nil || *info.InputVoltage != 230.8 {
		t.Errorf("input voltage = %v, want 230.8", info.InputVoltage)
	}
	if info.Load == nil || *info.Load != 0 {
		t.Errorf("load = %v, want 0", info.Load)
	}
	if info.Type != "offline / line interactive" {
		t.Errorf("type = %q", info.Type)
	}
}
