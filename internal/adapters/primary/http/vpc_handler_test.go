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

// MockVPCService is a test double implementing primary.VPCService.
type MockVPCService struct {
	DescribeVPCFn func(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
	InsertVPCFn   func(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
}

func (m *MockVPCService) DescribeVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	if m.DescribeVPCFn != nil {
		return m.DescribeVPCFn(ctx, provider, account, region, vpcID)
	}
	return nil, errors.New("not found")
}

func (m *MockVPCService) InsertVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	if m.InsertVPCFn != nil {
		return m.InsertVPCFn(ctx, provider, account, region, vpcID)
	}
	return nil, errors.New("not found")
}

// newVPCTestMux registers the VPC handler under the real route patterns.
func newVPCTestMux(h *httphandler.VPCHTTPHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /vpc/describe/{vpcID}", h.DescribeVPC)
	mux.HandleFunc("POST /vpc/insert", h.InsertVPC)
	return mux
}

// --- DescribeVPC tests ---

func TestDescribeVPCReturns200(t *testing.T) {
	mock := &MockVPCService{
		DescribeVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Name: "test-vpc", Region: "us-east-1", Provider: vpc.ProviderAWS, CIDRBlock: "10.0.0.0/16"}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
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
	mock := &MockVPCService{
		DescribeVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return nil, errors.New("vpc not found")
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/vpc/describe/vpc-999?provider=aws&account=production&region=us-east-1", nil)
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDescribeVPCListsSubnets(t *testing.T) {
	mock := &MockVPCService{
		DescribeVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Subnets: []vpc.Subnet{{ID: "sub-001", CIDRBlock: "10.0.1.0/24"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
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
	mock := &MockVPCService{
		DescribeVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Peerings: []vpc.Peering{{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
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
	mock := &MockVPCService{
		DescribeVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", VPNs: []vpc.VPN{{ID: "vpn-001", RemoteGateway: "203.0.113.1"}}}, nil
		},
	}
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/vpc/describe/vpc-001?provider=aws&account=production&region=us-east-1", nil)
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
	mock := &MockVPCService{
		InsertVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) { return scraped, nil },
	}

	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "vpcID": "vpc-001"})
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/vpc/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var result vpc.VPC
	json.NewDecoder(w.Body).Decode(&result)
	if result.CIDRBlock != "10.0.0.0/16" {
		t.Errorf("expected CIDR in response, got %q", result.CIDRBlock)
	}
}

func TestInsertVPCReturns400OnMissingFields(t *testing.T) {
	tests := []map[string]string{
		{"provider": "aws", "account": "production", "region": "us-east-1"},  // missing vpcID
		{"provider": "aws", "account": "production", "vpcID": "vpc-001"},     // missing region
		{"provider": "aws", "region": "us-east-1", "vpcID": "vpc-001"},      // missing account
		{"account": "production", "region": "us-east-1", "vpcID": "vpc-001"}, // missing provider
		{}, // all missing
	}
	for _, body := range tests {
		b, _ := json.Marshal(body)
		h := httphandler.NewVPCHTTPHandler(&MockVPCService{})
		req := httptest.NewRequest(http.MethodPost, "/vpc/insert", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newVPCTestMux(h).ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %v: expected 400, got %d", body, w.Code)
		}
	}
}

func TestInsertVPCReturns400OnInvalidJSON(t *testing.T) {
	h := httphandler.NewVPCHTTPHandler(&MockVPCService{})
	req := httptest.NewRequest(http.MethodPost, "/vpc/insert", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInsertVPCReturns500WhenServiceFails(t *testing.T) {
	mock := &MockVPCService{
		InsertVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return nil, errors.New("cloud error")
		},
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "vpcID": "vpc-001"})
	h := httphandler.NewVPCHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/vpc/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newVPCTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
