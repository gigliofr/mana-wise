package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

type memOTARepo struct {
	manifest *domain.OTAManifest
	files    map[string][]byte
}

func (m *memOTARepo) SaveRelease(ctx context.Context, version, platform string, payload []byte) (string, error) {
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	path := version + "-" + platform + ".bin"
	m.files[path] = payload
	return path, nil
}
func (m *memOTARepo) LoadManifest(ctx context.Context) (*domain.OTAManifest, error) {
	if m.manifest == nil {
		m.manifest = &domain.OTAManifest{}
	}
	cp := *m.manifest
	return &cp, nil
}
func (m *memOTARepo) SaveManifest(ctx context.Context, manifest *domain.OTAManifest) error {
	cp := *manifest
	m.manifest = &cp
	return nil
}

func TestOTAUpdateUseCase_PublishReleaseChecksum(t *testing.T) {
	repo := &memOTARepo{}
	uc := usecase.NewOTAUpdateUseCase(repo)
	payload := []byte("firmware-v1")
	h := sha256.Sum256(payload)
	res, err := uc.PublishRelease(context.Background(), domain.OTAReleaseRequest{
		Version:      "1.0.0",
		Platform:     "esp32",
		BinaryBase64: base64.StdEncoding.EncodeToString(payload),
		SHA256:       hex.EncodeToString(h[:]),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CurrentVersion != "1.0.0" || res.RolledBack {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestOTAUpdateUseCase_ReportBootRollback(t *testing.T) {
	repo := &memOTARepo{manifest: &domain.OTAManifest{CurrentVersion: "2.0.0", PreviousVersion: "1.0.0", CurrentPath: "2.bin", PreviousPath: "1.bin"}}
	uc := usecase.NewOTAUpdateUseCase(repo)
	res, err := uc.ReportBootResult(context.Background(), domain.OTABootReportRequest{Status: "failed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.RolledBack || res.CurrentVersion != "1.0.0" {
		t.Fatalf("rollback not applied: %+v", res)
	}
}
