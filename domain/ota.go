package domain

import "context"

// OTAReleaseRequest is the payload for publishing an OTA firmware release.
type OTAReleaseRequest struct {
	Version      string
	Platform     string
	BinaryBase64 string
	SHA256       string
}

// OTABootReportRequest reports the result of a device boot after update.
type OTABootReportRequest struct {
	DeviceID string
	Version  string
	Status   string // "ok" | "failed"
}

// OTAReleaseResult summarizes OTA publish/rollback operations.
type OTAReleaseResult struct {
	CurrentVersion  string `json:"current_version"`
	PreviousVersion string `json:"previous_version,omitempty"`
	RolledBack      bool   `json:"rolled_back"`
	Message         string `json:"message"`
}

// OTAManifest tracks current and previous firmware versions.
type OTAManifest struct {
	CurrentVersion  string `json:"current_version"`
	PreviousVersion string `json:"previous_version,omitempty"`
	CurrentPath     string `json:"current_path,omitempty"`
	PreviousPath    string `json:"previous_path,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}

// OTARepository persists OTA artifacts and manifest state.
type OTARepository interface {
	SaveRelease(ctx context.Context, version, platform string, payload []byte) (string, error)
	LoadManifest(ctx context.Context) (*OTAManifest, error)
	SaveManifest(ctx context.Context, manifest *OTAManifest) error
}
