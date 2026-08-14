package operations

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nas-ai/api/src/pathsafe"
	"github.com/sirupsen/logrus"
)

type BackupInfo struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
}

// Restore limits, mirroring the hardening already present in ArchiveService.
const (
	maxRestoreEntries   = 100_000
	maxRestoreTotalSize = 50 << 30 // 50 GiB
	restoreFileMode     = 0o644
	restoreDirMode      = 0o755
)

type BackupService struct {
	dataPath     string
	backupPath   string
	allowedRoots []string
	logger       *logrus.Logger
	timeNowFunc  func() time.Time
}

// NewBackupService creates the service. allowedRoots limits where an operator
// may point the backup destination; an empty list denies every change.
func NewBackupService(dataPath, backupPath string, allowedRoots []string, logger *logrus.Logger) (*BackupService, error) {
	// Normalize the data path once: the restore containment check compares
	// against it, and a relative or trailing-slash value would weaken that.
	absDataPath, err := filepath.Abs(dataPath)
	if err != nil {
		return nil, fmt.Errorf("resolve data path: %w", err)
	}

	svc := &BackupService{
		dataPath:     absDataPath,
		allowedRoots: allowedRoots,
		logger:       logger,
		timeNowFunc:  time.Now,
	}

	if err := os.MkdirAll(absDataPath, 0o755); err != nil {
		return nil, fmt.Errorf("ensure data path: %w", err)
	}
	if err := svc.SetBackupPath(backupPath); err != nil {
		return nil, err
	}

	return svc, nil
}

// SetBackupPath updates the destination directory for backups.
func (s *BackupService) SetBackupPath(path string) error {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" || cleanPath == "." || cleanPath == string(os.PathSeparator) {
		return fmt.Errorf("invalid backup path")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	// The destination is operator-supplied, so it must resolve inside one of
	// the configured roots. (The previous check compared absPath against
	// filepath.Clean(absPath), which filepath.Abs already guarantees — it
	// could never fail and blocked nothing.)
	if err := pathsafe.WithinAnyRoot(s.allowedRoots, absPath); err != nil {
		return fmt.Errorf("backup path not allowed: %w", err)
	}

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		return fmt.Errorf("ensure backup path: %w", err)
	}
	s.backupPath = absPath
	return nil
}

func (s *BackupService) ListBackups() ([]BackupInfo, error) {
	if s.backupPath == "" {
		return nil, fmt.Errorf("backup path not configured")
	}

	entries, err := os.ReadDir(s.backupPath)
	if err != nil {
		return nil, err
	}
	var result []BackupInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, BackupInfo{
			ID:      e.Name(),
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return result, nil
}

func (s *BackupService) CreateBackup() (BackupInfo, error) {
	// SECURITY FIX [BUG-GO-010]: Removed targetPath parameter to prevent path traversal attacks
	// Backups must always use the configured backupPath, not dynamic user-controlled paths
	if s.backupPath == "" {
		return BackupInfo{}, fmt.Errorf("backup path not configured")
	}

	ts := s.timeNowFunc().UTC().Format("20060102T150405Z")
	name := fmt.Sprintf("backup-%s.tar.gz", ts)
	dest := filepath.Join(s.backupPath, name)

	file, err := os.Create(dest)
	if err != nil {
		return BackupInfo{}, err
	}
	defer file.Close()

	gw := gzip.NewWriter(file)
	tw := tar.NewWriter(gw)

	err = filepath.Walk(s.dataPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(s.dataPath, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		// Normalize to forward slashes for tar
		header.Name = filepath.ToSlash(rel)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			src, err := os.Open(path)
			if err != nil {
				return err
			}
			defer src.Close()
			if _, err := io.Copy(tw, src); err != nil {
				return err
			}
		}
		return nil
	})

	// FIX [BUG-GO-015]: Check Close() errors - data may not be fully flushed
	closeErr := tw.Close()
	if closeErr != nil && err == nil {
		err = closeErr
	}
	closeErr = gw.Close()
	if closeErr != nil && err == nil {
		err = closeErr
	}

	if err != nil {
		_ = os.Remove(dest)
		return BackupInfo{}, err
	}

	info, err := os.Stat(dest)
	if err != nil {
		return BackupInfo{}, err
	}

	return BackupInfo{
		ID:      name,
		Name:    name,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

// PruneBackups removes older backups while keeping the newest "retention" files.
func (s *BackupService) PruneBackups(retention int) error {
	if retention < 1 {
		return fmt.Errorf("retention must be >= 1")
	}

	if s.backupPath == "" {
		return fmt.Errorf("backup path not configured")
	}

	entries, err := os.ReadDir(s.backupPath)
	if err != nil {
		return err
	}

	type backupEntry struct {
		name string
		mod  time.Time
	}

	var backups []backupEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupEntry{
			name: e.Name(),
			mod:  info.ModTime(),
		})
	}

	if len(backups) <= retention {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].mod.After(backups[j].mod)
	})

	for _, old := range backups[retention:] {
		target, err := s.backupFilePath(old.name)
		if err != nil {
			continue
		}
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove old backup %s: %w", old.name, err)
		}
	}
	return nil
}

// backupFilePath resolves a caller-supplied backup id to a file inside the
// backup directory. The id is reduced to its base name and the result is
// confined to backupPath, so it can never address another directory.
func (s *BackupService) backupFilePath(id string) (string, error) {
	return pathsafe.SafeJoin(s.backupPath, filepath.Base(id))
}

func (s *BackupService) DeleteBackup(id string) error {
	target, err := s.backupFilePath(id)
	if err != nil {
		return fmt.Errorf("invalid backup id")
	}
	return os.Remove(target)
}

func (s *BackupService) RestoreBackup(id string) error {
	target, err := s.backupFilePath(id)
	if err != nil {
		return fmt.Errorf("invalid backup id")
	}

	if err := s.cleanDataPath(); err != nil {
		return err
	}

	file, err := os.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()

	gr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var entries int
	var totalWritten int64
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		entries++
		if entries > maxRestoreEntries {
			return fmt.Errorf("archive contains too many entries (limit %d)", maxRestoreEntries)
		}

		// Confine every entry to the data directory: a crafted archive must not
		// be able to write outside it ("Zip Slip").
		destPath, err := pathsafe.SafeJoin(s.dataPath, hdr.Name)
		if err != nil {
			return fmt.Errorf("archive entry %q: %w", hdr.Name, err)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			// Fixed modes: archive-supplied permissions are not trusted.
			if err := os.MkdirAll(destPath, restoreDirMode); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destPath), restoreDirMode); err != nil {
				return err
			}
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, restoreFileMode)
			if err != nil {
				return err
			}
			remaining := maxRestoreTotalSize - totalWritten
			written, err := io.Copy(out, io.LimitReader(tr, remaining+1))
			closeErr := out.Close()
			if err != nil {
				return err
			}
			if closeErr != nil {
				return fmt.Errorf("close restored file: %w", closeErr)
			}
			totalWritten += written
			if totalWritten > maxRestoreTotalSize {
				return fmt.Errorf("archive exceeds maximum restore size")
			}
		default:
			// Symlinks, hardlinks, devices and similar are skipped: restoring
			// them would let an archive redirect later writes out of the tree.
			s.logger.WithFields(logrus.Fields{
				"name": hdr.Name,
				"type": hdr.Typeflag,
			}).Warn("skipping unsupported archive entry type during restore")
		}
	}

	return nil
}

func (s *BackupService) cleanDataPath() error {
	entries, err := os.ReadDir(s.dataPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		p := filepath.Join(s.dataPath, e.Name())
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

// Logger exposes the internal logger for callers that need to share the same logging pipeline.
func (s *BackupService) Logger() *logrus.Logger {
	return s.logger
}
