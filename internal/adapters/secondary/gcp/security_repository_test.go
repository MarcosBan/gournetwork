package gcp

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/api/compute/v1"

	"gournetwork/internal/domain/security"
)

// --- mock ---

type mockGCPFirewallClient struct {
	getFirewallOut    *compute.Firewall
	getFirewallErr    error
	listFirewallsOut  []*compute.Firewall
	listFirewallsErr  error
	patchFirewallErr  error
	deleteFirewallErr error
}

func (m *mockGCPFirewallClient) GetFirewall(project, firewall string) (*compute.Firewall, error) {
	return m.getFirewallOut, m.getFirewallErr
}
func (m *mockGCPFirewallClient) ListFirewalls(project, network string) ([]*compute.Firewall, error) {
	return m.listFirewallsOut, m.listFirewallsErr
}
func (m *mockGCPFirewallClient) PatchFirewall(project, firewall string, rb *compute.Firewall) error {
	return m.patchFirewallErr
}
func (m *mockGCPFirewallClient) DeleteFirewall(project, firewall string) error {
	return m.deleteFirewallErr
}

// --- auth tests ---

func TestNewGCPSecurityRepository_MissingProject(t *testing.T) {
	ctx := context.Background()
	_, err := NewGCPSecurityRepository(ctx, "")
	if err == nil {
		t.Fatal("expected error with empty project, got nil")
	}
}

func TestNewGCPSecurityRepository_InvalidCredentialsJSON(t *testing.T) {
	ctx := context.Background()
	_, err := NewGCPSecurityRepositoryWithCredentialsJSON(ctx, "my-project", []byte(`not-json`))
	if err == nil {
		t.Fatal("expected error with invalid credentials JSON, got nil")
	}
}

func TestGCPSecurityRepository_GetSecurityGroup_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 401: Request had invalid authentication credentials")
	repo := newGCPSecurityRepositoryWithClient(&mockGCPFirewallClient{getFirewallErr: authErr}, "my-project")
	_, err := repo.GetSecurityGroup(context.Background(), "gcp", "us-central1", "my-firewall")
	if err == nil {
		t.Fatal("expected auth error to be propagated, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

func TestGCPSecurityRepository_ListSecurityGroups_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 403: Permission denied")
	repo := newGCPSecurityRepositoryWithClient(&mockGCPFirewallClient{listFirewallsErr: authErr}, "my-project")
	_, err := repo.ListSecurityGroups(context.Background(), "gcp", "us-central1", "prod-network")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// --- GetSecurityGroup ---

func TestGCPSecurityRepository_GetSecurityGroup(t *testing.T) {
	mock := &mockGCPFirewallClient{
		getFirewallOut: &compute.Firewall{
			Name:         "allow-http",
			Network:      "https://...networks/prod-network",
			Direction:    "INGRESS",
			SourceRanges: []string{"0.0.0.0/0"},
			Priority:     1000,
			Allowed: []*compute.FirewallAllowed{
				{IPProtocol: "tcp", Ports: []string{"80"}},
			},
		},
	}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	sg, err := repo.GetSecurityGroup(context.Background(), "gcp", "us-central1", "allow-http")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sg.ID != "allow-http" {
		t.Errorf("expected ID allow-http, got %q", sg.ID)
	}
	if sg.Provider != "gcp" {
		t.Errorf("expected provider gcp, got %q", sg.Provider)
	}
	if len(sg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sg.Rules))
	}
	if sg.Rules[0].Direction != "ingress" {
		t.Errorf("expected direction ingress, got %q", sg.Rules[0].Direction)
	}
	if sg.Rules[0].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %q", sg.Rules[0].Protocol)
	}
	if sg.Rules[0].Action != "allow" {
		t.Errorf("expected action allow, got %q", sg.Rules[0].Action)
	}
	if sg.Rules[0].PortRange.From != 80 {
		t.Errorf("expected port from 80, got %d", sg.Rules[0].PortRange.From)
	}
}

func TestGCPSecurityRepository_GetSecurityGroup_DenyRule(t *testing.T) {
	mock := &mockGCPFirewallClient{
		getFirewallOut: &compute.Firewall{
			Name:         "deny-all",
			Direction:    "INGRESS",
			SourceRanges: []string{"0.0.0.0/0"},
			Priority:     65534,
			Denied: []*compute.FirewallDenied{
				{IPProtocol: "all"},
			},
		},
	}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	sg, err := repo.GetSecurityGroup(context.Background(), "gcp", "us-central1", "deny-all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sg.Rules))
	}
	if sg.Rules[0].Action != "deny" {
		t.Errorf("expected action deny, got %q", sg.Rules[0].Action)
	}
}

// --- ListSecurityGroups ---

func TestGCPSecurityRepository_ListSecurityGroups(t *testing.T) {
	mock := &mockGCPFirewallClient{
		listFirewallsOut: []*compute.Firewall{
			{Name: "fw-1", Direction: "INGRESS", Allowed: []*compute.FirewallAllowed{{IPProtocol: "tcp"}}},
			{Name: "fw-2", Direction: "EGRESS", Allowed: []*compute.FirewallAllowed{{IPProtocol: "udp"}}},
		},
	}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	groups, err := repo.ListSecurityGroups(context.Background(), "gcp", "us-central1", "prod-network")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 security groups, got %d", len(groups))
	}
}

// --- UpdateRule ---

func TestGCPSecurityRepository_UpdateRule(t *testing.T) {
	mock := &mockGCPFirewallClient{}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	rule := security.SecurityRule{
		Direction: "ingress",
		Protocol:  "tcp",
		PortRange: security.PortRange{From: 443, To: 443},
		Sources:   []string{"10.0.0.0/8"},
		Action:    "allow",
		Priority:  1000,
	}
	if err := repo.UpdateRule(context.Background(), "gcp", "us-central1", "allow-https", rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGCPSecurityRepository_UpdateRule_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 403: Permission denied")
	mock := &mockGCPFirewallClient{patchFirewallErr: authErr}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	err := repo.UpdateRule(context.Background(), "gcp", "us-central1", "allow-https", security.SecurityRule{})
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- DeleteRule ---

func TestGCPSecurityRepository_DeleteRule(t *testing.T) {
	mock := &mockGCPFirewallClient{}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	if err := repo.DeleteRule(context.Background(), "gcp", "us-central1", "allow-http", "rule-id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGCPSecurityRepository_DeleteRule_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 403: Permission denied")
	mock := &mockGCPFirewallClient{deleteFirewallErr: authErr}
	repo := newGCPSecurityRepositoryWithClient(mock, "my-project")
	err := repo.DeleteRule(context.Background(), "gcp", "us-central1", "allow-http", "rule-id")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// Compile-time check: mock implements the interface.
var _ gcpFirewallClient = (*mockGCPFirewallClient)(nil)
