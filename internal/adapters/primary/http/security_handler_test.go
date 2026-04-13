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
)

// MockSecurityService is a test double implementing primary.SecurityService.
type MockSecurityService struct {
	DescribeSecurityGroupFn func(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error)
	InsertSecurityGroupFn   func(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error)
	RemoveRuleFn            func(ctx context.Context, provider, account, region, groupID, ruleID string) error
}

func (m *MockSecurityService) DescribeSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	if m.DescribeSecurityGroupFn != nil {
		return m.DescribeSecurityGroupFn(ctx, provider, account, region, groupID)
	}
	return nil, errors.New("not found")
}

func (m *MockSecurityService) InsertSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	if m.InsertSecurityGroupFn != nil {
		return m.InsertSecurityGroupFn(ctx, provider, account, region, groupID)
	}
	return nil, errors.New("not found")
}

func (m *MockSecurityService) RemoveRule(ctx context.Context, provider, account, region, groupID, ruleID string) error {
	if m.RemoveRuleFn != nil {
		return m.RemoveRuleFn(ctx, provider, account, region, groupID, ruleID)
	}
	return errors.New("not found")
}

// newSecurityTestMux registers the handler under the real route patterns.
func newSecurityTestMux(h *httphandler.SecurityHTTPHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /security-rules/describe", h.DescribeSecurityGroup)
	mux.HandleFunc("POST /security-rules/insert", h.InsertRule)
	mux.HandleFunc("DELETE /security-rules/remove", h.RemoveRule)
	return mux
}

// --- DescribeSecurityGroup tests ---

func TestDescribeSecurityGroupReturns200(t *testing.T) {
	mock := &MockSecurityService{
		DescribeSecurityGroupFn: func(_ context.Context, _, _, _, _ string) (*security.SecurityGroup, error) {
			return &security.SecurityGroup{ID: "sg-001", Name: "web-sg", VPCID: "vpc-001"}, nil
		},
	}
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/security-rules/describe?provider=aws&account=production&region=us-east-1&groupID=sg-001", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result security.SecurityGroup
	json.NewDecoder(w.Body).Decode(&result)
	if result.ID != "sg-001" {
		t.Errorf("expected ID sg-001, got %q", result.ID)
	}
}

func TestDescribeSecurityGroupReturns404(t *testing.T) {
	mock := &MockSecurityService{
		DescribeSecurityGroupFn: func(_ context.Context, _, _, _, _ string) (*security.SecurityGroup, error) {
			return nil, errors.New("not found")
		},
	}
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/security-rules/describe?provider=aws&account=production&region=us-east-1&groupID=sg-999", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- InsertRule tests ---

func TestInsertRuleReturns201WithScrapedData(t *testing.T) {
	scraped := &security.SecurityGroup{
		ID: "sg-001", Name: "web-sg", Provider: "aws", VPCID: "vpc-001",
		Rules: []security.SecurityRule{{Direction: "ingress", Protocol: "tcp", Action: "allow"}},
	}
	mock := &MockSecurityService{
		InsertSecurityGroupFn: func(_ context.Context, _, _, _, _ string) (*security.SecurityGroup, error) {
			return scraped, nil
		},
	}

	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "groupID": "sg-001"})
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/security-rules/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var result security.SecurityGroup
	json.NewDecoder(w.Body).Decode(&result)
	if result.Name != "web-sg" {
		t.Errorf("expected Name in response, got %q", result.Name)
	}
}

func TestInsertRuleReturns400OnMissingFields(t *testing.T) {
	tests := []map[string]string{
		{"provider": "aws", "account": "production", "region": "us-east-1"},  // missing groupID
		{"provider": "aws", "account": "production", "groupID": "sg-001"},    // missing region
		{"provider": "aws", "region": "us-east-1", "groupID": "sg-001"},      // missing account
		{"account": "production", "region": "us-east-1", "groupID": "sg-001"}, // missing provider
		{},
	}
	for _, body := range tests {
		b, _ := json.Marshal(body)
		h := httphandler.NewSecurityHTTPHandler(&MockSecurityService{})
		req := httptest.NewRequest(http.MethodPost, "/security-rules/insert", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newSecurityTestMux(h).ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %v: expected 400, got %d", body, w.Code)
		}
	}
}

func TestInsertRuleReturns400OnInvalidJSON(t *testing.T) {
	h := httphandler.NewSecurityHTTPHandler(&MockSecurityService{})
	req := httptest.NewRequest(http.MethodPost, "/security-rules/insert", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInsertRuleReturns500WhenServiceFails(t *testing.T) {
	mock := &MockSecurityService{
		InsertSecurityGroupFn: func(_ context.Context, _, _, _, _ string) (*security.SecurityGroup, error) {
			return nil, errors.New("cloud error")
		},
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "account": "production", "region": "us-east-1", "groupID": "sg-001"})
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/security-rules/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- RemoveRule tests ---

func TestRemoveRuleReturns204(t *testing.T) {
	mock := &MockSecurityService{
		RemoveRuleFn: func(_ context.Context, _, _, _, _, _ string) error { return nil },
	}
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/security-rules/remove?provider=aws&account=production&region=us-east-1&groupID=sg-001&ruleID=rule-001", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestRemoveRuleReturns404WhenNotFound(t *testing.T) {
	mock := &MockSecurityService{
		RemoveRuleFn: func(_ context.Context, _, _, _, _, _ string) error { return errors.New("not found") },
	}
	h := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/security-rules/remove?provider=aws&account=production&region=us-east-1&groupID=sg-001&ruleID=rule-999", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
