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
	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// MockVPCRepository is a test double implementing CloudVPCRepository.
type MockVPCRepository struct {
	GetVPCFn       func(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
	ListVPCsFn     func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error)
	UpdateRoutesFn func(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error
	ListSubnetsFn  func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error)
	ListPeeringsFn func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error)
	ListVPNsFn     func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error)
}

func (m *MockVPCRepository) GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	if m.GetVPCFn != nil {
		return m.GetVPCFn(ctx, provider, account, region, vpcID)
	}
	return nil, nil
}
func (m *MockVPCRepository) ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, account, region)
	}
	return []vpc.VPC{}, nil
}
func (m *MockVPCRepository) UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error {
	if m.UpdateRoutesFn != nil {
		return m.UpdateRoutesFn(ctx, provider, account, region, vpcID, routes)
	}
	return nil
}
func (m *MockVPCRepository) ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error) {
	if m.ListSubnetsFn != nil {
		return m.ListSubnetsFn(ctx, provider, account, region, vpcID)
	}
	return []vpc.Subnet{}, nil
}
func (m *MockVPCRepository) ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error) {
	if m.ListPeeringsFn != nil {
		return m.ListPeeringsFn(ctx, provider, account, region, vpcID)
	}
	return []vpc.Peering{}, nil
}
func (m *MockVPCRepository) ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error) {
	if m.ListVPNsFn != nil {
		return m.ListVPNsFn(ctx, provider, account, region, vpcID)
	}
	return []vpc.VPN{}, nil
}

// MockStorageRepository is a test double implementing StorageRepository.
type MockStorageRepository struct {
	SaveVPCFn           func(ctx context.Context, v *vpc.VPC) error
	SaveSecurityGroupFn func(ctx context.Context, sg *security.SecurityGroup) error
}

func (m *MockStorageRepository) SaveVPC(ctx context.Context, v *vpc.VPC) error {
	if m.SaveVPCFn != nil {
		return m.SaveVPCFn(ctx, v)
	}
	return nil
}
func (m *MockStorageRepository) SaveSecurityGroup(ctx context.Context, sg *security.SecurityGroup) error {
	if m.SaveSecurityGroupFn != nil {
		return m.SaveSecurityGroupFn(ctx, sg)
	}
	return nil
}

// newVPCTestMux registers the VPC handler under the real route patterns.
func newVPCTestMux(h *httphandler.VPCHTTPHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /aws/vpc/describe/{vpcID}", h.DescribeVPC)
	mux.HandleFunc("POST /aws/vpc/insert", h.InsertVPC)
	return mux
}

// --- DescribeVPC tests ---

func TestDescribeVPCReturns200(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Name: "test-vpc", Region: "us-east-1", Provider: vpc.ProviderAWS, CIDRBlock: "10.0.0.0/16"}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result vpc.VPC
	json.NewDecoder(w.Body).Decode(&result)
	if result.ID != "vpc-001" {
		t.Errorf("expected ID vpc-001, got %q", result.ID)
	}
}

func TestDescribeVPCReturns404WhenNotFound(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return nil, errors.New("vpc not found")
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/describe/vpc-999?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDescribeVPCListsSubnets(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Subnets: []vpc.Subnet{{ID: "sub-001", CIDRBlock: "10.0.1.0/24"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	subnetsArr, _ := result["Subnets"].([]interface{})
	if len(subnetsArr) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(subnetsArr))
	}
}

func TestDescribeVPCListsPeerings(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Peerings: []vpc.Peering{{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	peeringsArr, _ := result["Peerings"].([]interface{})
	if len(peeringsArr) != 1 {
		t.Errorf("expected 1 peering, got %d", len(peeringsArr))
	}
}

func TestDescribeVPCListsVPNs(t *testing.T) {
	mock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", VPNs: []vpc.VPN{{ID: "vpn-001", RemoteGateway: "203.0.113.1"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	vpnsArr, _ := result["VPNs"].([]interface{})
	if len(vpnsArr) != 1 {
		t.Errorf("expected 1 VPN, got %d", len(vpnsArr))
	}
}

// --- InsertVPC tests ---

func TestInsertVPCReturns201WithScrapedData(t *testing.T) {
	scraped := &vpc.VPC{
		ID: "vpc-001", Name: "prod-vpc", Region: "us-east-1",
		Provider: vpc.ProviderAWS, CIDRBlock: "10.0.0.0/16",
	}
	vpcMock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) { return scraped, nil },
	}
	var stored *vpc.VPC
	storeMock := &MockStorageRepository{
		SaveVPCFn: func(_ context.Context, v *vpc.VPC) error { stored = v; return nil },
	}

	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "vpcID": "vpc-001"})
	h := httphandler.NewVPCHTTPHandler(vpcMock, storeMock)
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if stored == nil {
		t.Fatal("expected SaveVPC to be called, but it was not")
	}
	if stored.ID != "vpc-001" {
		t.Errorf("expected stored VPC ID vpc-001, got %q", stored.ID)
	}

	var result vpc.VPC
	json.NewDecoder(w.Body).Decode(&result)
	if result.CIDRBlock != "10.0.0.0/16" {
		t.Errorf("expected CIDR in response, got %q", result.CIDRBlock)
	}
}

func TestInsertVPCReturns400OnMissingFields(t *testing.T) {
	tests := []map[string]string{
		{"provider": "aws", "account": "production", "region": "us-east-1"}, // missing vpcID
		{"provider": "aws", "account": "production", "vpcID": "vpc-001"},    // missing region
		{"provider": "aws", "region": "us-east-1", "vpcID": "vpc-001"},      // missing account
		{"account": "production", "region": "us-east-1", "vpcID": "vpc-001"}, // missing provider
		{}, // all missing
	}
	for _, body := range tests {
		b, _ := json.Marshal(body)
		h := httphandler.NewVPCHTTPHandler(&MockVPCRepository{}, &MockStorageRepository{})
		req := httptest.NewRequest(http.MethodPost, "/aws/vpc/insert", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newVPCTestMux(h).ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %v: expected 400, got %d", body, w.Code)
		}
	}
}

func TestInsertVPCReturns400OnInvalidJSON(t *testing.T) {
	h := httphandler.NewVPCHTTPHandler(&MockVPCRepository{}, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/insert", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInsertVPCReturns500WhenCloudFetchFails(t *testing.T) {
	vpcMock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return nil, errors.New("cloud error")
		},
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "vpcID": "vpc-001"})
	h := httphandler.NewVPCHTTPHandler(vpcMock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestInsertVPCReturns500WhenStorageFails(t *testing.T) {
	vpcMock := &MockVPCRepository{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Region: "us-east-1", Provider: vpc.ProviderAWS}, nil
		},
	}
	storeMock := &MockStorageRepository{
		SaveVPCFn: func(_ context.Context, _ *vpc.VPC) error { return errors.New("disk full") },
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "vpcID": "vpc-001"})
	h := httphandler.NewVPCHTTPHandler(vpcMock, storeMock)
	req := httptest.NewRequest(http.MethodPost, "/aws/vpc/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
