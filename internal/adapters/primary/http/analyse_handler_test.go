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
	"gournetwork/internal/domain/vpc"
)

// mockAnalyseVPCRepo is a VPC repository mock for analyse handler tests.
type mockAnalyseVPCRepo struct {
	GetVPCFn       func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error)
	ListVPCsFn     func(ctx context.Context, provider, region string) ([]vpc.VPC, error)
	UpdateRoutesFn func(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error
	ListSubnetsFn  func(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error)
	ListPeeringsFn func(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error)
	ListVPNsFn     func(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error)
}

func (m *mockAnalyseVPCRepo) GetVPC(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
	if m.GetVPCFn != nil {
		return m.GetVPCFn(ctx, provider, region, vpcID)
	}
	return nil, nil
}

func (m *mockAnalyseVPCRepo) ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, region)
	}
	return []vpc.VPC{}, nil
}

func (m *mockAnalyseVPCRepo) UpdateRoutes(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error {
	if m.UpdateRoutesFn != nil {
		return m.UpdateRoutesFn(ctx, provider, region, vpcID, routes)
	}
	return nil
}

func (m *mockAnalyseVPCRepo) ListSubnets(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error) {
	if m.ListSubnetsFn != nil {
		return m.ListSubnetsFn(ctx, provider, region, vpcID)
	}
	return []vpc.Subnet{}, nil
}

func (m *mockAnalyseVPCRepo) ListPeerings(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error) {
	if m.ListPeeringsFn != nil {
		return m.ListPeeringsFn(ctx, provider, region, vpcID)
	}
	return []vpc.Peering{}, nil
}

func (m *mockAnalyseVPCRepo) ListVPNs(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error) {
	if m.ListVPNsFn != nil {
		return m.ListVPNsFn(ctx, provider, region, vpcID)
	}
	return []vpc.VPN{}, nil
}

// mockAnalyseSecRepo is a security repository mock for analyse handler tests.
type mockAnalyseSecRepo struct{}

func (m *mockAnalyseSecRepo) GetSecurityGroup(ctx context.Context, provider, region, groupID string) (*network.ConnectivityResult, error) {
	return nil, nil
}

func TestAnalyseConnectivityReturns200WithResult(t *testing.T) {
	vpcMock := &mockAnalyseVPCRepo{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: vpcID, Name: "source-vpc"}, nil
		},
	}
	secMock := &MockSecurityRepository{}

	handler := httphandler.NewAnalyseHTTPHandler(vpcMock, secMock)

	body := `{"source_vpc": "vpc-001", "destination_cidr": "10.1.0.0/16"}`
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
	vpcMock := &mockAnalyseVPCRepo{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			// Return nil to simulate VPC not found, triggering not connected result.
			return nil, nil
		},
	}
	secMock := &MockSecurityRepository{}

	handler := httphandler.NewAnalyseHTTPHandler(vpcMock, secMock)

	body := `{"source_vpc": "vpc-999", "destination_cidr": "10.99.0.0/16"}`
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
	vpcMock := &mockAnalyseVPCRepo{}
	secMock := &MockSecurityRepository{}

	handler := httphandler.NewAnalyseHTTPHandler(vpcMock, secMock)

	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAnalyseReturns400OnMissingFields(t *testing.T) {
	vpcMock := &mockAnalyseVPCRepo{}
	secMock := &MockSecurityRepository{}

	handler := httphandler.NewAnalyseHTTPHandler(vpcMock, secMock)

	req := httptest.NewRequest(http.MethodPost, "/analyse", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AnalyseConnectivity(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
