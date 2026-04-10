package security_test

import (
	"testing"

	"gournetwork/internal/domain/security"
)

func TestSecurityGroupHasRules(t *testing.T) {
	group := security.SecurityGroup{
		ID:   "sg-001",
		Name: "web-sg",
		Rules: []security.SecurityRule{
			{
				ID:        "rule-001",
				Direction: "ingress",
				Protocol:  "tcp",
				PortRange: security.PortRange{From: 80, To: 80},
				Sources:   []string{"0.0.0.0/0"},
				Action:    "allow",
			},
		},
	}

	if len(group.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(group.Rules))
	}
	if group.Rules[0].Direction != "ingress" {
		t.Errorf("expected direction ingress, got %s", group.Rules[0].Direction)
	}
	if group.Rules[0].Protocol != "tcp" {
		t.Errorf("expected protocol tcp, got %s", group.Rules[0].Protocol)
	}
}

func TestSecurityRulePortRange(t *testing.T) {
	t.Run("valid port range", func(t *testing.T) {
		rule := security.SecurityRule{
			PortRange: security.PortRange{From: 80, To: 443},
		}
		if rule.PortRange.From > rule.PortRange.To {
			t.Errorf("expected From (%d) <= To (%d)", rule.PortRange.From, rule.PortRange.To)
		}
	})

	t.Run("single port range", func(t *testing.T) {
		rule := security.SecurityRule{
			PortRange: security.PortRange{From: 22, To: 22},
		}
		if rule.PortRange.From > rule.PortRange.To {
			t.Errorf("expected From (%d) <= To (%d)", rule.PortRange.From, rule.PortRange.To)
		}
	})
}

func TestSecurityRuleIngressDirection(t *testing.T) {
	rule := security.SecurityRule{
		ID:        "rule-ingress",
		Direction: "ingress",
	}
	if rule.Direction != "ingress" {
		t.Errorf("expected direction ingress, got %s", rule.Direction)
	}
}

func TestSecurityRuleEgressDirection(t *testing.T) {
	rule := security.SecurityRule{
		ID:        "rule-egress",
		Direction: "egress",
	}
	if rule.Direction != "egress" {
		t.Errorf("expected direction egress, got %s", rule.Direction)
	}
}

func TestSecurityRuleActionAllow(t *testing.T) {
	rule := security.SecurityRule{
		ID:     "rule-allow",
		Action: "allow",
	}
	if rule.Action != "allow" {
		t.Errorf("expected action allow, got %s", rule.Action)
	}
}

func TestSecurityRuleActionDeny(t *testing.T) {
	rule := security.SecurityRule{
		ID:     "rule-deny",
		Action: "deny",
	}
	if rule.Action != "deny" {
		t.Errorf("expected action deny, got %s", rule.Action)
	}
}

func TestSecurityGroupBelongsToVPC(t *testing.T) {
	group := security.SecurityGroup{
		ID:    "sg-001",
		Name:  "web-sg",
		VPCID: "vpc-001",
	}
	if group.VPCID != "vpc-001" {
		t.Errorf("expected VPCID vpc-001, got %s", group.VPCID)
	}
}
