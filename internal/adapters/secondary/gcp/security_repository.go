package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"

	"gournetwork/internal/domain/security"
)

// gcpFirewallClient defines the GCP Compute firewall operations used by GCPSecurityRepository.
type gcpFirewallClient interface {
	GetFirewall(project, firewall string) (*compute.Firewall, error)
	ListFirewalls(project, network string) ([]*compute.Firewall, error)
	PatchFirewall(project, firewall string, rb *compute.Firewall) error
	DeleteFirewall(project, firewall string) error
}

// firewallServiceWrapper implements gcpFirewallClient using the real GCP compute.Service.
type firewallServiceWrapper struct {
	svc *compute.Service
}

func (w *firewallServiceWrapper) GetFirewall(project, firewall string) (*compute.Firewall, error) {
	return w.svc.Firewalls.Get(project, firewall).Do()
}

func (w *firewallServiceWrapper) ListFirewalls(project, network string) ([]*compute.Firewall, error) {
	list, err := w.svc.Firewalls.List(project).Filter(fmt.Sprintf("network = \"%s\"", network)).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (w *firewallServiceWrapper) PatchFirewall(project, firewall string, rb *compute.Firewall) error {
	op, err := w.svc.Firewalls.Patch(project, firewall, rb).Do()
	if err != nil {
		return err
	}
	return waitForGlobalOp(w.svc, project, op.Name)
}

func (w *firewallServiceWrapper) DeleteFirewall(project, firewall string) error {
	op, err := w.svc.Firewalls.Delete(project, firewall).Do()
	if err != nil {
		return err
	}
	return waitForGlobalOp(w.svc, project, op.Name)
}

// GCPSecurityRepository is the GCP implementation of CloudSecurityRepository.
// In production it uses a GCPClientRegistry to route calls per project alias;
// in tests a single injected client and project ID are used instead.
type GCPSecurityRepository struct {
	registry    *GCPClientRegistry
	testClient  gcpFirewallClient // non-nil only in test mode
	testProject string            // used together with testClient
}

// NewGCPSecurityRepositoryFromRegistry creates a production repository backed by the given registry.
func NewGCPSecurityRepositoryFromRegistry(registry *GCPClientRegistry) *GCPSecurityRepository {
	return &GCPSecurityRepository{registry: registry}
}

// newGCPSecurityRepositoryWithClient creates a repository with an injected client (for tests).
func newGCPSecurityRepositoryWithClient(client gcpFirewallClient, project string) *GCPSecurityRepository {
	return &GCPSecurityRepository{testClient: client, testProject: project}
}

// clientFor returns the GCP firewall client and actual project ID for the given alias.
// In test mode the injected client and project are always returned.
func (r *GCPSecurityRepository) clientFor(alias string) (gcpFirewallClient, string, error) {
	if r.testClient != nil {
		return r.testClient, r.testProject, nil
	}
	return r.registry.firewallClient(alias)
}

// GetSecurityGroup retrieves a GCP firewall rule set by name.
// In GCP, the groupID corresponds to the firewall rule name.
func (r *GCPSecurityRepository) GetSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	fw, err := client.GetFirewall(project, groupID)
	if err != nil {
		return nil, fmt.Errorf("get firewall: %w", err)
	}
	return mapGCPFirewall(fw), nil
}

// ListSecurityGroups lists all GCP firewall rules for a network (vpcID).
func (r *GCPSecurityRepository) ListSecurityGroups(ctx context.Context, provider, account, region, vpcID string) ([]security.SecurityGroup, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	fws, err := client.ListFirewalls(project, vpcID)
	if err != nil {
		return nil, fmt.Errorf("list firewalls: %w", err)
	}
	result := make([]security.SecurityGroup, 0, len(fws))
	for _, fw := range fws {
		result = append(result, *mapGCPFirewall(fw))
	}
	return result, nil
}

// UpdateRule updates (patches) the firewall rule identified by groupID with the new rule definition.
func (r *GCPSecurityRepository) UpdateRule(ctx context.Context, provider, account, region, groupID string, rule security.SecurityRule) error {
	client, project, err := r.clientFor(account)
	if err != nil {
		return err
	}
	fw := toGCPFirewall(rule)
	if err := client.PatchFirewall(project, groupID, fw); err != nil {
		return fmt.Errorf("patch firewall %q: %w", groupID, err)
	}
	return nil
}

// DeleteRule deletes the firewall rule identified by groupID.
// ruleID is ignored — GCP firewalls are standalone resources identified by groupID.
func (r *GCPSecurityRepository) DeleteRule(ctx context.Context, provider, account, region, groupID, ruleID string) error {
	client, project, err := r.clientFor(account)
	if err != nil {
		return err
	}
	if err := client.DeleteFirewall(project, groupID); err != nil {
		return fmt.Errorf("delete firewall %q: %w", groupID, err)
	}
	return nil
}

// mapGCPFirewall converts a GCP Firewall resource to the domain SecurityGroup model.
func mapGCPFirewall(fw *compute.Firewall) *security.SecurityGroup {
	var rules []security.SecurityRule
	direction := strings.ToLower(fw.Direction)
	for _, a := range fw.Allowed {
		from, to := 0, 65535
		if len(a.Ports) > 0 {
			fmt.Sscanf(a.Ports[0], "%d-%d", &from, &to)
			if to == 0 {
				to = from
			}
		}
		rules = append(rules, security.SecurityRule{
			Direction: direction,
			Protocol:  a.IPProtocol,
			PortRange: security.PortRange{From: from, To: to},
			Sources:   fw.SourceRanges,
			Action:    "allow",
			Priority:  int(fw.Priority),
		})
	}
	for _, d := range fw.Denied {
		from, to := 0, 65535
		if len(d.Ports) > 0 {
			fmt.Sscanf(d.Ports[0], "%d-%d", &from, &to)
			if to == 0 {
				to = from
			}
		}
		rules = append(rules, security.SecurityRule{
			Direction: direction,
			Protocol:  d.IPProtocol,
			PortRange: security.PortRange{From: from, To: to},
			Sources:   fw.SourceRanges,
			Action:    "deny",
			Priority:  int(fw.Priority),
		})
	}
	return &security.SecurityGroup{
		ID:       fw.Name,
		Name:     fw.Name,
		Provider: "gcp",
		VPCID:    networkNameFromURL(fw.Network),
		Rules:    rules,
	}
}

// toGCPFirewall converts a domain SecurityRule to a GCP Firewall patch payload.
func toGCPFirewall(rule security.SecurityRule) *compute.Firewall {
	portStr := fmt.Sprintf("%d-%d", rule.PortRange.From, rule.PortRange.To)
	if rule.PortRange.From == rule.PortRange.To {
		portStr = fmt.Sprintf("%d", rule.PortRange.From)
	}
	fw := &compute.Firewall{
		Direction:    strings.ToUpper(rule.Direction),
		SourceRanges: rule.Sources,
		Priority:     int64(rule.Priority),
	}
	if rule.Action == "allow" {
		fw.Allowed = []*compute.FirewallAllowed{{
			IPProtocol: rule.Protocol,
			Ports:      []string{portStr},
		}}
	} else {
		fw.Denied = []*compute.FirewallDenied{{
			IPProtocol: rule.Protocol,
			Ports:      []string{portStr},
		}}
	}
	return fw
}
