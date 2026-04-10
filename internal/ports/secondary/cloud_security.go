package secondary

import (
	"context"

	"gournetwork/internal/domain/security"
)

// CloudSecurityRepository defines the interface for interacting with cloud security group resources.
type CloudSecurityRepository interface {
	GetSecurityGroup(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error)
	ListSecurityGroups(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error)
	UpdateRule(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error
	DeleteRule(ctx context.Context, provider, region, groupID, ruleID string) error
}
