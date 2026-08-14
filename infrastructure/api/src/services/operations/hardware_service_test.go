package operations

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHardwareService_GetStorageInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewHardwareService(logger)
	disks, err := svc.GetStorageInfo()
	if err != nil {
		t.Fatalf("GetStorageInfo failed: %v", err)
	}

	if len(disks) == 0 {
		t.Fatal("Expected at least one disk, got none")
	}

	// Check root is present
	foundRoot := false
	for _, d := range disks {
		if d.MountPoint == "/" {
			foundRoot = true
			t.Logf("Root disk: Total=%d, Used=%d, Free=%d, UsedPercent=%.2f%%", 
				d.Total, d.Used, d.Free, d.UsedPercent)
		}
	}
	if !foundRoot {
		t.Error("Root '/' mount point not found")
	}

	t.Logf("Found %d disk(s)", len(disks))
}

func TestHardwareService_GetNetworkInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewHardwareService(logger)
	interfaces, err := svc.GetNetworkInfo()
	if err != nil {
		t.Fatalf("GetNetworkInfo failed: %v", err)
	}

	if len(interfaces) == 0 {
		t.Fatal("Expected at least one network interface, got none")
	}

	for _, iface := range interfaces {
		t.Logf("Interface: %s, MAC: %s, IPs: %v, BytesRecv: %d, BytesSent: %d",
			iface.Name, iface.MAC, iface.IPs, iface.BytesRecv, iface.BytesSent)
	}

	t.Logf("Found %d interface(s)", len(interfaces))
}
