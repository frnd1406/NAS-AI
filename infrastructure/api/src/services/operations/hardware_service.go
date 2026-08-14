package operations

import (
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/sirupsen/logrus"
)

// HardwareService provides system hardware information
type HardwareService struct {
	logger *logrus.Logger
}

// NewHardwareService creates a new hardware service
func NewHardwareService(logger *logrus.Logger) *HardwareService {
	return &HardwareService{
		logger: logger,
	}
}

// DiskInfo represents physical disk usage
type DiskInfo struct {
	MountPoint  string  `json:"mount_point"`
	Device      string  `json:"device"`
	Filesystem  string  `json:"filesystem"`
	DriveType   string  `json:"drive_type"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInterfaceInfo represents network interface details
type NetworkInterfaceInfo struct {
	Name        string   `json:"name"`
	MAC         string   `json:"mac"`
	IPs         []string `json:"ips"`
	Flags       []string `json:"flags"`
	BytesSent   uint64   `json:"bytes_sent"`
	BytesRecv   uint64   `json:"bytes_recv"`
	PacketsSent uint64   `json:"packets_sent"`
	PacketsRecv uint64   `json:"packets_recv"`
}

// GetStorageInfo returns information about physical storage
// When running in Docker with /:/host:ro mount, reads from host filesystem
func (s *HardwareService) GetStorageInfo() ([]DiskInfo, error) {
	var disks []DiskInfo

	// Get partitions (physical devices only)
	partitions, err := disk.Partitions(false)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get partitions")
		return nil, err
	}

	seen := make(map[string]bool)

	for _, p := range partitions {
		// Skip if already seen
		if seen[p.Mountpoint] {
			continue
		}

		// === FILTER: Only real physical drives ===
		// Must be a real block device (not loop, docker, snap, etc)
		isPhysicalDrive := false

		// Check if it's a real block device on host
		if strings.HasPrefix(p.Device, "/dev/sd") || // SATA/USB drives
			strings.HasPrefix(p.Device, "/dev/nvme") || // NVMe SSDs
			strings.HasPrefix(p.Device, "/dev/hd") || // IDE drives (legacy)
			strings.HasPrefix(p.Device, "/dev/vd") { // Virtual disks (VMs)
			isPhysicalDrive = true
		}

		// Skip non-physical drives
		if !isPhysicalDrive {
			continue
		}

		// Skip system/special mounts
		if strings.HasPrefix(p.Mountpoint, "/snap") ||
			strings.HasPrefix(p.Mountpoint, "/var/lib/docker") ||
			strings.HasPrefix(p.Mountpoint, "/run") ||
			strings.HasPrefix(p.Mountpoint, "/sys") ||
			strings.HasPrefix(p.Mountpoint, "/proc") ||
			strings.HasPrefix(p.Mountpoint, "/dev") ||
			strings.Contains(p.Mountpoint, "/loop") {
			continue
		}

		// Skip Docker volume mounts inside container
		if strings.HasPrefix(p.Mountpoint, "/mnt/data") ||
			strings.HasPrefix(p.Mountpoint, "/mnt/backups") ||
			strings.HasPrefix(p.Mountpoint, "/etc/resolv") ||
			strings.HasPrefix(p.Mountpoint, "/etc/hostname") ||
			strings.HasPrefix(p.Mountpoint, "/etc/hosts") {
			continue
		}

		seen[p.Mountpoint] = true

		// Get usage stats
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			s.logger.WithField("mount", p.Mountpoint).Warn("Failed to get disk usage, skipping")
			continue
		}

		// Determine drive type label
		driveType := "HDD"
		if strings.Contains(p.Device, "nvme") {
			driveType = "NVMe SSD"
		} else if strings.Contains(p.Mountpoint, "/media") {
			driveType = "Removable"
		}

		disks = append(disks, DiskInfo{
			MountPoint:  p.Mountpoint,
			Device:      p.Device,
			Filesystem:  p.Fstype,
			DriveType:   driveType,
			Total:       usage.Total,
			Free:        usage.Free,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
		})
	}

	// Also check host mount at /host if it exists (Docker with host mount)
	hostRoot := "/host"
	if _, err := disk.Usage(hostRoot); err == nil && !seen[hostRoot] {
		// Read host partitions from /host/proc/mounts
		s.addHostDrives(&disks, hostRoot, seen)
	}

	return disks, nil
}

// addHostDrives reads physical drives from host mount
func (s *HardwareService) addHostDrives(disks *[]DiskInfo, hostRoot string, seen map[string]bool) {
	// Try to get usage of common host mount points
	hostMounts := []string{
		hostRoot,            // /host (host root)
		hostRoot + "/home",  // /host/home
		hostRoot + "/media", // /host/media (removable)
	}

	for _, mount := range hostMounts {
		if seen[mount] {
			continue
		}

		usage, err := disk.Usage(mount)
		if err != nil {
			continue
		}

		// Skip if it's basically the same as something we already have (same total size)
		duplicate := false
		for _, d := range *disks {
			if d.Total == usage.Total && d.UsedPercent == usage.UsedPercent {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		seen[mount] = true

		label := "Host Filesystem"
		if strings.Contains(mount, "/home") {
			label = "Host Home"
		} else if strings.Contains(mount, "/media") {
			label = "Removable"
		}

		*disks = append(*disks, DiskInfo{
			MountPoint:  mount,
			Device:      "host",
			Filesystem:  "ext4",
			DriveType:   label,
			Total:       usage.Total,
			Free:        usage.Free,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
		})
	}
}

// GetNetworkInfo returns information about network interfaces
func (s *HardwareService) GetNetworkInfo() ([]NetworkInterfaceInfo, error) {
	var results []NetworkInterfaceInfo

	interfaces, err := net.Interfaces()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get network interfaces")
		return nil, err
	}

	ioCounters, err := net.IOCounters(true)
	ioMap := make(map[string]net.IOCountersStat)
	if err == nil {
		for _, stat := range ioCounters {
			ioMap[stat.Name] = stat
		}
	}

	for _, iface := range interfaces {
		// Filter loopback if desired, but user might want to see it. Keeping it for now.
		var info NetworkInterfaceInfo
		info.Name = iface.Name
		info.MAC = iface.HardwareAddr
		info.Flags = iface.Flags

		for _, addr := range iface.Addrs {
			info.IPs = append(info.IPs, addr.Addr)
		}

		if stat, ok := ioMap[iface.Name]; ok {
			info.BytesSent = stat.BytesSent
			info.BytesRecv = stat.BytesRecv
			info.PacketsSent = stat.PacketsSent
			info.PacketsRecv = stat.PacketsRecv
		}

		results = append(results, info)
	}

	return results, nil
}
