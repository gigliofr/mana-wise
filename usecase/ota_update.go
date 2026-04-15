package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// OTAUpdateUseCase provides secure OTA publish and rollback operations.
type OTAUpdateUseCase struct {
	repo domain.OTARepository
}

// NewOTAUpdateUseCase creates OTA use case.
func NewOTAUpdateUseCase(repo domain.OTARepository) *OTAUpdateUseCase {
	return &OTAUpdateUseCase{repo: repo}
}

// PublishRelease validates checksum and publishes a new firmware artifact.
func (uc *OTAUpdateUseCase) PublishRelease(ctx context.Context, req domain.OTAReleaseRequest) (*domain.OTAReleaseResult, error) {
	if strings.TrimSpace(req.Version) == "" {
		return nil, fmt.Errorf("version is required")
	}
	if strings.TrimSpace(req.Platform) == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if strings.TrimSpace(req.BinaryBase64) == "" {
		return nil, fmt.Errorf("binary_base64 is required")
	}
	if strings.TrimSpace(req.SHA256) == "" {
		return nil, fmt.Errorf("sha256 is required")
	}

	payload, err := base64.StdEncoding.DecodeString(req.BinaryBase64)
	if err != nil {
		return nil, fmt.Errorf("decode binary_base64: %w", err)
	}
	if err = verifySHA256(payload, req.SHA256); err != nil {
		return nil, err
	}

	manifest, err := uc.repo.LoadManifest(ctx)
	if err != nil {
		return nil, err
	}

	relPath, err := uc.repo.SaveRelease(ctx, req.Version, req.Platform, payload)
	if err != nil {
		return nil, err
	}

	manifest.PreviousVersion = manifest.CurrentVersion
	manifest.PreviousPath = manifest.CurrentPath
	manifest.CurrentVersion = req.Version
	manifest.CurrentPath = relPath
	if err = uc.repo.SaveManifest(ctx, manifest); err != nil {
		return nil, err
	}

	return &domain.OTAReleaseResult{
		CurrentVersion:  manifest.CurrentVersion,
		PreviousVersion: manifest.PreviousVersion,
		RolledBack:      false,
		Message:         "release published",
	}, nil
}

// ReportBootResult processes boot status and triggers rollback on failure.
func (uc *OTAUpdateUseCase) ReportBootResult(ctx context.Context, req domain.OTABootReportRequest) (*domain.OTAReleaseResult, error) {
	manifest, err := uc.repo.LoadManifest(ctx)
	if err != nil {
		return nil, err
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "ok" {
		return &domain.OTAReleaseResult{
			CurrentVersion:  manifest.CurrentVersion,
			PreviousVersion: manifest.PreviousVersion,
			RolledBack:      false,
			Message:         "boot success recorded",
		}, nil
	}
	if status != "failed" {
		return nil, fmt.Errorf("status must be 'ok' or 'failed'")
	}
	if manifest.PreviousVersion == "" || manifest.PreviousPath == "" {
		return nil, fmt.Errorf("rollback unavailable: no previous release")
	}

	manifest.CurrentVersion, manifest.PreviousVersion = manifest.PreviousVersion, manifest.CurrentVersion
	manifest.CurrentPath, manifest.PreviousPath = manifest.PreviousPath, manifest.CurrentPath
	if err = uc.repo.SaveManifest(ctx, manifest); err != nil {
		return nil, err
	}

	return &domain.OTAReleaseResult{
		CurrentVersion:  manifest.CurrentVersion,
		PreviousVersion: manifest.PreviousVersion,
		RolledBack:      true,
		Message:         "rollback executed after boot failure",
	}, nil
}

// GetManifest returns the current OTA manifest.
func (uc *OTAUpdateUseCase) GetManifest(ctx context.Context) (*domain.OTAManifest, error) {
	return uc.repo.LoadManifest(ctx)
}

func verifySHA256(payload []byte, expectedHex string) error {
	h := sha256.Sum256(payload)
	actual := hex.EncodeToString(h[:])
	expected := strings.ToLower(strings.TrimSpace(expectedHex))
	if actual != expected {
		return fmt.Errorf("sha256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}
