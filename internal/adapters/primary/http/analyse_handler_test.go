package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "gournetwork/internal/adapters/primary/http"
	"gournetwork/internal/domain/network"
)

// MockAnalyseService is a test double implementing primary.AnalyseService.
type MockAnalyseService struct {
	AnalyseConnectivityFn func(ctx context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error)
}

func (m *MockAnalyseService) AnalyseConnectivity(ctx context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error) {
	if m.AnalyseConnectivityFn != nil {
		return m.AnalyseConnectivityFn(ctx, req)
	}
	return &network.ConnectivityResult{}, nil
}

func TestAnalyseConnectivityReturns200WithResult(t *testing.T) {
	mock := &MockAnalyseService{
		AnalyseConnectivityFn: func(_ context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error) {
			return &network.ConnectivityResult{
				Source:      req.SourceVPC,
				Destination: req.DestCIDR,
				Connected:   true,
				Path:        []string{req.SourceVPC, req.DestCIDR},
			}, nil
		},
	}

	handler := httphandler.NewAnalyseHTTPHandler(mock)

	body := `{"source_provider":"aws","source_account":"production","source_region":"us-east-1","source_vpc":"vpc-001","destination_cidr":"10.1.0.0/16"}`
	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result network.ConnectivityResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !result.Connected {
		t.Errorf("expected Connected to be true")
	}
}

func TestAnalyseReturns200NotConnected(t *testing.T) {
	mock := &MockAnalyseService{
		AnalyseConnectivityFn: func(_ context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error) {
			return &network.ConnectivityResult{
				Source:      req.SourceVPC,
				Destination: req.DestCIDR,
				Connected:   false,
				Reason:      "source VPC not found",
			}, nil
		},
	}

	handler := httphandler.NewAnalyseHTTPHandler(mock)

	body := `{"source_vpc":"vpc-999","destination_cidr":"10.99.0.0/16"}`
	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result network.ConnectivityResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Connected {
		t.Errorf("expected Connected to be false")
	}
	if result.Reason == "" {
		t.Errorf("expected non-empty Reason when not connected")
	}
}

func TestAnalyseReturns400OnInvalidBody(t *testing.T) {
	mock := &MockAnalyseService{}
	handler := httphandler.NewAnalyseHTTPHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAnalyseReturns400OnMissingFields(t *testing.T) {
	mock := &MockAnalyseService{}
	handler := httphandler.NewAnalyseHTTPHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
