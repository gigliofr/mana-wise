package ota

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/manawise/api/domain"
)

const manifestFile = "manifest.json"

// StorageRepository stores OTA binaries and manifest metadata on local disk.
type StorageRepository struct {
	baseDir string
}

// NewStorageRepository creates a filesystem-backed OTA repository.
func NewStorageRepository(baseDir string) (*StorageRepository, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "./ota-releases"
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create ota storage dir: %w", err)
	}
	return &StorageRepository{baseDir: baseDir}, nil
}

// SaveRelease writes release payload to disk and returns relative path.
func (r *StorageRepository) SaveRelease(ctx context.Context, version, platform string, payload []byte) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	safeVersion := sanitize(version)
	safePlatform := sanitize(platform)
	rel := fmt.Sprintf("%s-%s.bin", safeVersion, safePlatform)
	fullPath := filepath.Join(r.baseDir, rel)
	if err := os.WriteFile(fullPath, payload, 0o644); err != nil {
		return "", fmt.Errorf("write ota release: %w", err)
	}
	return rel, nil
}

// LoadManifest reads the manifest, returning an empty one if absent.
func (r *StorageRepository) LoadManifest(ctx context.Context) (*domain.OTAManifest, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path := filepath.Join(r.baseDir, manifestFile)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &domain.OTAManifest{UpdatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
		}
		return nil, fmt.Errorf("read ota manifest: %w", err)
	}
	var m domain.OTAManifest
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("decode ota manifest: %w", err)
	}
	if m.UpdatedAt == "" {
		m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return &m, nil
}

// SaveManifest persists the latest manifest state.
func (r *StorageRepository) SaveManifest(ctx context.Context, manifest *domain.OTAManifest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	manifest.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	b, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ota manifest: %w", err)
	}
	path := filepath.Join(r.baseDir, manifestFile)
	if err = os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write ota manifest: %w", err)
	}
	return nil
}

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("..", "", "/", "-", "\\", "-", " ", "-")
	s = replacer.Replace(s)
	if s == "" {
		return "unknown"
	}
	return s
}
