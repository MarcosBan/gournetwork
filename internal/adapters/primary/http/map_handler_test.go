package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "gournetwork/internal/adapters/primary/http"
	"gournetwork/internal/domain/vpc"
)

// MockMapRepository is a VPC repository mock for map handler tests.
type MockMapRepository struct {
	ListVPCsFn func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error)
}

func (m *MockMapRepository) GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	return nil, nil
}
func (m *MockMapRepository) ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, account, region)
	}
	return []vpc.VPC{}, nil
}
func (m *MockMapRepository) UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error {
	return nil
}
func (m *MockMapRepository) ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error) {
	return []vpc.Subnet{}, nil
}
func (m *MockMapRepository) ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error) {
	return []vpc.Peering{}, nil
}
func (m *MockMapRepository) ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error) {
	return []vpc.VPN{}, nil
}

func TestGetNetworkMapReturns200(t *testing.T) {
	mock := &MockMapRepository{
		ListVPCsFn: func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
			return []vpc.VPC{
				{ID: "vpc-001", Name: "vpc-a", Region: "us-east-1"},
				{ID: "vpc-002", Name: "vpc-b", Region: "us-west-2"},
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

	vpcs, ok := result["VPCs"]
	if !ok {
		t.Fatal("response does not contain 'VPCs' key")
	}
	vpcsArr, ok := vpcs.([]interface{})
	if !ok {
		t.Fatal("VPCs is not an array")
	}
	if len(vpcsArr) != 2 {
		t.Errorf("expected 2 VPCs, got %d", len(vpcsArr))
	}
}

func TestGetNetworkMapEmptyReturns200(t *testing.T) {
	mock := &MockMapRepository{
		ListVPCsFn: func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
			return []vpc.VPC{}, nil
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

	vpcs, ok := result["VPCs"]
	if !ok {
		t.Fatal("response does not contain 'VPCs' key")
	}
	vpcsArr, ok := vpcs.([]interface{})
	if !ok {
		t.Fatalf("VPCs is not an array, got %T", vpcs)
	}
	if len(vpcsArr) != 0 {
		t.Errorf("expected 0 VPCs, got %d", len(vpcsArr))
	}
}

func TestGetNetworkMapResponseStructure(t *testing.T) {
	mock := &MockMapRepository{
		ListVPCsFn: func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
			return []vpc.VPC{{ID: "vpc-001"}}, nil
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

	requiredKeys := []string{"VPCs", "Connections", "GeneratedAt"}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			t.Errorf("response is missing required key %q", key)
		}
	}
}
