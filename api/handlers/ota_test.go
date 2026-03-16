package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

type memOTARepoHandler struct {
	manifest *domain.OTAManifest
	files    map[string][]byte
}

func (m *memOTARepoHandler) SaveRelease(ctx context.Context, version, platform string, payload []byte) (string, error) {
	if m.files == nil {
		m.files = map[string][]byte{}
	}
	path := version + "-" + platform + ".bin"
	m.files[path] = payload
	return path, nil
}

func (m *memOTARepoHandler) LoadManifest(ctx context.Context) (*domain.OTAManifest, error) {
	if m.manifest == nil {
		m.manifest = &domain.OTAManifest{}
	}
	cp := *m.manifest
	return &cp, nil
}

func (m *memOTARepoHandler) SaveManifest(ctx context.Context, manifest *domain.OTAManifest) error {
	cp := *manifest
	m.manifest = &cp
	return nil
}

func newOTAHandlerForTest() (*OTAHandler, *memOTARepoHandler) {
	repo := &memOTARepoHandler{}
	uc := usecase.NewOTAUpdateUseCase(repo)
	return NewOTAHandler(uc), repo
}

func TestOTAHandler_PublishRelease_Success(t *testing.T) {
	h, _ := newOTAHandlerForTest()
	payload := []byte("firmware-v1")
	hash := sha256.Sum256(payload)

	body := map[string]string{
		"version":       "1.0.0",
		"platform":      "esp32",
		"binary_base64": base64.StdEncoding.EncodeToString(payload),
		"sha256":        hex.EncodeToString(hash[:]),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/release", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.PublishRelease(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOTAHandler_PublishRelease_InvalidChecksum(t *testing.T) {
	h, _ := newOTAHandlerForTest()
	payload := []byte("firmware-v1")

	body := map[string]string{
		"version":       "1.0.0",
		"platform":      "esp32",
		"binary_base64": base64.StdEncoding.EncodeToString(payload),
		"sha256":        "deadbeef",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/release", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.PublishRelease(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestOTAHandler_ReportBoot_Rollback(t *testing.T) {
	h, repo := newOTAHandlerForTest()
	repo.manifest = &domain.OTAManifest{
		CurrentVersion:  "2.0.0",
		PreviousVersion: "1.0.0",
		CurrentPath:     "2.bin",
		PreviousPath:    "1.bin",
	}

	body := map[string]string{"status": "failed", "device_id": "d1", "version": "2.0.0"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/report-boot", bytes.NewReader(b))
	rr := httptest.NewRecorder()

	h.ReportBoot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if repo.manifest.CurrentVersion != "1.0.0" {
		t.Fatalf("expected rollback to 1.0.0, got %s", repo.manifest.CurrentVersion)
	}
}

func TestOTAHandler_Manifest(t *testing.T) {
	h, repo := newOTAHandlerForTest()
	repo.manifest = &domain.OTAManifest{CurrentVersion: "1.2.3"}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/manifest", nil)
	rr := httptest.NewRecorder()

	h.Manifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var got domain.OTAManifest
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.CurrentVersion != "1.2.3" {
		t.Fatalf("expected current_version 1.2.3, got %s", got.CurrentVersion)
	}
}
