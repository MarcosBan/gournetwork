package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httphandler "gournetwork/internal/adapters/primary/http"
	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/vpc"
)

// MockMapService is a test double implementing primary.MapService.
type MockMapService struct {
	GetNetworkMapFn func(ctx context.Context, providers []network.ProviderQuery) (*network.NetworkMap, error)
}

func (m *MockMapService) GetNetworkMap(ctx context.Context, providers []network.ProviderQuery) (*network.NetworkMap, error) {
	if m.GetNetworkMapFn != nil {
		return m.GetNetworkMapFn(ctx, providers)
	}
	return &network.NetworkMap{
		VPCs:        []vpc.VPC{},
		Connections: []network.Connection{},
		GeneratedAt: time.Now(),
	}, nil
}

func TestGetNetworkMapReturns200(t *testing.T) {
	mock := &MockMapService{
		GetNetworkMapFn: func(_ context.Context, _ []network.ProviderQuery) (*network.NetworkMap, error) {
			return &network.NetworkMap{
				VPCs: []vpc.VPC{
					{ID: "vpc-001", Name: "vpc-a", Region: "us-east-1"},
					{ID: "vpc-002", Name: "vpc-b", Region: "us-west-2"},
				},
				Connections: []network.Connection{},
				GeneratedAt: time.Now(),
			}, nil
		},
	}

	handler := httphandler.NewMapHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/map?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()

	handler.GetNetworkMap(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	vpcsVal, ok := result["vpcs"]
	if !ok {
		t.Fatal("response does not contain 'vpcs' key")
	}
	vpcsArr, ok := vpcsVal.([]interface{})
	if !ok {
		t.Fatal("vpcs is not an array")
	}
	if len(vpcsArr) != 2 {
		t.Errorf("expected 2 VPCs, got %d", len(vpcsArr))
	}
}

func TestGetNetworkMapEmptyReturns200(t *testing.T) {
	mock := &MockMapService{}
	handler := httphandler.NewMapHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/map", nil)
	w := httptest.NewRecorder()

	handler.GetNetworkMap(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	vpcsVal, ok := result["vpcs"]
	if !ok {
		t.Fatal("response does not contain 'vpcs' key")
	}
	vpcsArr, ok := vpcsVal.([]interface{})
	if !ok {
		t.Fatalf("vpcs is not an array, got %T", vpcsVal)
	}
	if len(vpcsArr) != 0 {
		t.Errorf("expected 0 VPCs, got %d", len(vpcsArr))
	}
}

func TestGetNetworkMapResponseStructure(t *testing.T) {
	mock := &MockMapService{
		GetNetworkMapFn: func(_ context.Context, _ []network.ProviderQuery) (*network.NetworkMap, error) {
			return &network.NetworkMap{
				VPCs:        []vpc.VPC{{ID: "vpc-001"}},
				Connections: []network.Connection{},
				GeneratedAt: time.Now(),
			}, nil
		},
	}

	handler := httphandler.NewMapHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/map", nil)
	w := httptest.NewRecorder()

	handler.GetNetworkMap(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	requiredKeys := []string{"vpcs", "connections", "generated_at"}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("response is missing required key %q", key)
		}
	}
}

func TestGetNetworkMapMultipleProviders(t *testing.T) {
	mock := &MockMapService{
		GetNetworkMapFn: func(_ context.Context, providers []network.ProviderQuery) (*network.NetworkMap, error) {
			if len(providers) != 2 {
				t.Errorf("expected 2 providers, got %d", len(providers))
			}
			return &network.NetworkMap{
				VPCs:        []vpc.VPC{{ID: "vpc-001"}, {ID: "net-001"}},
				Connections: []network.Connection{},
				GeneratedAt: time.Now(),
			}, nil
		},
	}

	handler := httphandler.NewMapHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/map?providers=aws,gcp&account=prod&region=us-east-1", nil)
	w := httptest.NewRecorder()

	handler.GetNetworkMap(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
