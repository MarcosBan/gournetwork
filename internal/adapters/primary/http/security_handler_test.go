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
	GetSecurityGroupFn    func(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error)
	ListSecurityGroupsFn  func(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error)
	UpdateRuleFn          func(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error
	DeleteRuleFn          func(ctx context.Context, provider, region, groupID, ruleID string) error
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

func TestDescribeSecurityGroupReturns200(t *testing.T) {
	mock := &MockSecurityRepository{
		GetSecurityGroupFn: func(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error) {
			return &security.SecurityGroup{
				ID:    "sg-001",
				Name:  "web-sg",
				VPCID: "vpc-001",
			}, nil
		},
	}

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001", nil)
	w := httptest.NewRecorder()

	handler.DescribeSecurityGroup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result security.SecurityGroup
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.ID != "sg-001" {
		t.Errorf("expected group ID sg-001, got %s", result.ID)
	}
}

func TestDescribeSecurityGroupReturns404(t *testing.T) {
	mock := &MockSecurityRepository{
		GetSecurityGroupFn: func(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error) {
			return nil, errors.New("security group not found")
		},
	}

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodGet, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-999", nil)
	w := httptest.NewRecorder()

	handler.DescribeSecurityGroup(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateRuleReturns200(t *testing.T) {
	mock := &MockSecurityRepository{
		UpdateRuleFn: func(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error {
			return nil
		},
	}

	rule := security.SecurityRule{
		ID:        "rule-001",
		Direction: "ingress",
		Protocol:  "tcp",
		PortRange: security.PortRange{From: 80, To: 80},
		Action:    "allow",
	}
	body, _ := json.Marshal(rule)

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateRule(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestUpdateRuleReturns400OnInvalidBody(t *testing.T) {
	mock := &MockSecurityRepository{}

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodPost, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateRule(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDeleteRuleReturns204(t *testing.T) {
	mock := &MockSecurityRepository{
		DeleteRuleFn: func(ctx context.Context, provider, region, groupID, ruleID string) error {
			return nil
		},
	}

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001&ruleID=rule-001", nil)
	w := httptest.NewRecorder()

	handler.DeleteRule(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestDeleteRuleReturns404WhenNotFound(t *testing.T) {
	mock := &MockSecurityRepository{
		DeleteRuleFn: func(ctx context.Context, provider, region, groupID, ruleID string) error {
			return errors.New("rule not found")
		},
	}

	handler := httphandler.NewSecurityHTTPHandler(mock)
	req := httptest.NewRequest(http.MethodDelete, "/aws/security-rules/describe?provider=aws&region=us-east-1&groupID=sg-001&ruleID=rule-999", nil)
	w := httptest.NewRecorder()

	handler.DeleteRule(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
