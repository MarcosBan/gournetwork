package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httphandler "gournetwork/internal/adapters/primary/http"
	"gournetwork/internal/domain/vpc"
)

// MockVPCRepository is a test double implementing CloudVPCRepository.
type MockVPCRepository struct {
	GetVPCFn       func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error)
	ListVPCsFn     func(ctx context.Context, provider, region string) ([]vpc.VPC, error)
	UpdateRoutesFn func(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error
	ListSubnetsFn  func(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error)
	ListPeeringsFn func(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error)
	ListVPNsFn     func(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error)
}

func (m *MockVPCRepository) GetVPC(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
	if m.GetVPCFn != nil {
		return m.GetVPCFn(ctx, provider, region, vpcID)
	}
	return nil, nil
}

func (m *MockVPCRepository) ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, region)
	}
	return []vpc.VPC{}, nil
}

func (m *MockVPCRepository) UpdateRoutes(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error {
	if m.UpdateRoutesFn != nil {
		return m.UpdateRoutesFn(ctx, provider, region, vpcID, routes)
	}
	return nil
}

func (m *MockVPCRepository) ListSubnets(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error) {
	if m.ListSubnetsFn != nil {
		return m.ListSubnetsFn(ctx, provider, region, vpcID)
	}
	return []vpc.Subnet{}, nil
}

func (m *MockVPCRepository) ListPeerings(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error) {
	if m.ListPeeringsFn != nil {
		return m.ListPeeringsFn(ctx, provider, region, vpcID)
	}
	return []vpc.Peering{}, nil
}

func (m *MockVPCRepository) ListVPNs(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error) {
	if m.ListVPNsFn != nil {
		return m.ListVPNsFn(ctx, provider, region, vpcID)
	}
	return []vpc.VPN{}, nil
}

func TestDescribeVPCReturns200(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID:        "vpc-001",
				Name:      "test-vpc",
				Region:    "us-east-1",
				Provider:  vpc.ProviderAWS,
				CIDRBlock: "10.0.0.0/16",
			}, nil
		},
	}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", nil)
	w := httptest.NewRecorder()

	handler.DescribeVPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result vpc.VPC
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "vpc-001" {
		t.Errorf("expected VPC ID vpc-001, got %s", result.ID)
	}
}

func TestDescribeVPCReturns404WhenNotFound(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return nil, errors.New("vpc not found")
		},
	}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-999", nil)
	w := httptest.NewRecorder()

	handler.DescribeVPC(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateRoutesReturns200(t *testing.T) {
	mock := &MockVPCRepository{
		UpdateRoutesFn: func(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error {
			return nil
		},
	}

	routes := []vpc.Route{
		{Destination: "0.0.0.0/0", NextHop: "igw-001"},
	}
	body, _ := json.Marshal(routes)

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateRoutes(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestUpdateRoutesReturns400OnInvalidBody(t *testing.T) {
	mock := &MockVPCRepository{}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateRoutes(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDescribeVPCListsSubnets(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID:      "vpc-001",
				Subnets: []vpc.Subnet{{ID: "sub-001", CIDRBlock: "10.0.1.0/24"}},
			}, nil
		},
	}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", nil)
	w := httptest.NewRecorder()

	handler.DescribeVPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	subnets, ok := result["Subnets"]
	if !ok {
		t.Fatal("response body does not contain 'Subnets' key")
	}
	subnetsArr, ok := subnets.([]interface{})
	if !ok {
		t.Fatal("Subnets is not an array")
	}
	if len(subnetsArr) != 1 {
		t.Errorf("expected 1 subnet in response, got %d", len(subnetsArr))
	}
}

func TestDescribeVPCListsPeerings(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID: "vpc-001",
				Peerings: []vpc.Peering{
					{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"},
				},
			}, nil
		},
	}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", nil)
	w := httptest.NewRecorder()

	handler.DescribeVPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	peerings, ok := result["Peerings"]
	if !ok {
		t.Fatal("response body does not contain 'Peerings' key")
	}
	peeringsArr, ok := peerings.([]interface{})
	if !ok {
		t.Fatal("Peerings is not an array")
	}
	if len(peeringsArr) != 1 {
		t.Errorf("expected 1 peering in response, got %d", len(peeringsArr))
	}
}

func TestDescribeVPCListsVPNs(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID: "vpc-001",
				VPNs: []vpc.VPN{
					{ID: "vpn-001", RemoteGateway: "203.0.113.1", State: "available"},
				},
			}, nil
		},
	}

	handler := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/?provider=aws&region=us-east-1&vpcID=vpc-001", nil)
	w := httptest.NewRecorder()

	handler.DescribeVPC(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	vpns, ok := result["VPNs"]
	if !ok {
		t.Fatal("response body does not contain 'VPNs' key")
	}
	vpnsArr, ok := vpns.([]interface{})
	if !ok {
		t.Fatal("VPNs is not an array")
	}
	if len(vpnsArr) != 1 {
		t.Errorf("expected 1 VPN in response, got %d", len(vpnsArr))
	}
}
