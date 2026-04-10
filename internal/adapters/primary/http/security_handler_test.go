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

// MockSecurityRepository is a test double implementing CloudSecurityRepository.
type MockSecurityRepository struct {
	GetSecurityGroupFn   func(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error)
	ListSecurityGroupsFn func(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error)
	UpdateRuleFn         func(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error
	DeleteRuleFn         func(ctx context.Context, provider, region, groupID, ruleID string) error
}

func (m *MockSecurityRepository) GetSecurityGroup(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error) {
	if m.GetSecurityGroupFn != nil {
		return m.GetSecurityGroupFn(ctx, provider, region, groupID)
	}
	return nil, nil
}
func (m *MockSecurityRepository) ListSecurityGroups(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error) {
	if m.ListSecurityGroupsFn != nil {
		return m.ListSecurityGroupsFn(ctx, provider, region, vpcID)
	}
	return []security.SecurityGroup{}, nil
}
func (m *MockSecurityRepository) UpdateRule(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error {
	if m.UpdateRuleFn != nil {
		return m.UpdateRuleFn(ctx, provider, region, groupID, rule)
	}
	return nil
}
func (m *MockSecurityRepository) DeleteRule(ctx context.Context, provider, region, groupID, ruleID string) error {
	if m.DeleteRuleFn != nil {
		return m.DeleteRuleFn(ctx, provider, region, groupID, ruleID)
	}
	return nil
}

// newSecurityTestMux registers the handler under the real route patterns (AWS variant).
func newSecurityTestMux(h *httphandler.SecurityHTTPHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /aws/security-rules/describe", h.DescribeSecurityGroup)
	mux.HandleFunc("POST /aws/security-rules/insert", h.InsertRule)
	mux.HandleFunc("DELETE /aws/security-rules/remove", h.RemoveRule)
	return mux
}

// --- DescribeSecurityGroup tests ---

func TestDescribeSecurityGroupReturns200(t *testing.T) {
	mock := &MockSecurityRepository{
		GetSecurityGroupFn: func(_ context.Context, _, _, _ string) (*security.SecurityGroup, error) {
			return &security.SecurityGroup{ID: "sg-001", Name: "web-sg", VPCID: "vpc-001"}, nil
		},
	}
	h := httphandler.NewSecurityHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001", nil)
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
	mock := &MockSecurityRepository{
		GetSecurityGroupFn: func(_ context.Context, _, _, _ string) (*security.SecurityGroup, error) {
			return nil, errors.New("not found")
		},
	}
	h := httphandler.NewSecurityHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodGet, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-999", nil)
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
	secMock := &MockSecurityRepository{
		GetSecurityGroupFn: func(_ context.Context, _, _, _ string) (*security.SecurityGroup, error) {
			return scraped, nil
		},
	}
	var stored *security.SecurityGroup
	storeMock := &MockStorageRepository{
		SaveSecurityGroupFn: func(_ context.Context, sg *security.SecurityGroup) error {
			stored = sg
			return nil
		},
	}

	body, _ := json.Marshal(map[string]string{"provider": "aws", "region": "us-east-1", "groupID": "sg-001"})
	h := httphandler.NewSecurityHTTPHandler(secMock, storeMock)
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if stored == nil {
		t.Fatal("expected SaveSecurityGroup to be called, but it was not")
	}
	if stored.ID != "sg-001" {
		t.Errorf("expected stored group ID sg-001, got %q", stored.ID)
	}

	var result security.SecurityGroup
	json.NewDecoder(w.Body).Decode(&result)
	if result.Name != "web-sg" {
		t.Errorf("expected Name in response, got %q", result.Name)
	}
}

func TestInsertRuleReturns400OnMissingFields(t *testing.T) {
	tests := []map[string]string{
		{"provider": "aws", "region": "us-east-1"},  // missing groupID
		{"provider": "aws", "groupID": "sg-001"},    // missing region
		{"region": "us-east-1", "groupID": "sg-001"}, // missing provider
		{},
	}
	for _, body := range tests {
		b, _ := json.Marshal(body)
		h := httphandler.NewSecurityHTTPHandler(&MockSecurityRepository{}, &MockStorageRepository{})
		req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/insert", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		newSecurityTestMux(h).ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %v: expected 400, got %d", body, w.Code)
		}
	}
}

func TestInsertRuleReturns400OnInvalidJSON(t *testing.T) {
	h := httphandler.NewSecurityHTTPHandler(&MockSecurityRepository{}, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/insert", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestInsertRuleReturns500WhenCloudFetchFails(t *testing.T) {
	secMock := &MockSecurityRepository{
		GetSecurityGroupFn: func(_ context.Context, _, _, _ string) (*security.SecurityGroup, error) {
			return nil, errors.New("cloud error")
		},
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "region": "us-east-1", "groupID": "sg-001"})
	h := httphandler.NewSecurityHTTPHandler(secMock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestInsertRuleReturns500WhenStorageFails(t *testing.T) {
	secMock := &MockSecurityRepository{
		GetSecurityGroupFn: func(_ context.Context, _, _, _ string) (*security.SecurityGroup, error) {
			return &security.SecurityGroup{ID: "sg-001", Provider: "aws", VPCID: "vpc-001"}, nil
		},
	}
	storeMock := &MockStorageRepository{
		SaveSecurityGroupFn: func(_ context.Context, _ *security.SecurityGroup) error {
			return errors.New("disk full")
		},
	}
	body, _ := json.Marshal(map[string]string{"provider": "aws", "region": "us-east-1", "groupID": "sg-001"})
	h := httphandler.NewSecurityHTTPHandler(secMock, storeMock)
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/insert", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// --- RemoveRule tests ---

func TestRemoveRuleReturns204(t *testing.T) {
	mock := &MockSecurityRepository{
		DeleteRuleFn: func(_ context.Context, _, _, _, _ string) error { return nil },
	}
	h := httphandler.NewSecurityHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/aws/security-rules/remove?provider=aws&region=us-east-1&groupID=sg-001&ruleID=rule-001", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestRemoveRuleReturns404WhenNotFound(t *testing.T) {
	mock := &MockSecurityRepository{
		DeleteRuleFn: func(_ context.Context, _, _, _, _ string) error { return errors.New("not found") },
	}
	h := httphandler.NewSecurityHTTPHandler(mock, &MockStorageRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/aws/security-rules/remove?provider=aws&region=us-east-1&groupID=sg-001&ruleID=rule-999", nil)
	w := httptest.NewRecorder()
	newSecurityTestMux(h).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
