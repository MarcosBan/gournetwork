package secondary

import (
	"context"

	"gournetwork/internal/domain/security"
)

// CloudSecurityRepository defines the interface for interacting with cloud security group resources.
// The account parameter identifies the logical credential set to use:
// for AWS it is the account alias (e.g. "production"), for GCP it is the project alias.
type CloudSecurityRepository interface {
	GetSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error)
	ListSecurityGroups(ctx context.Context, provider, account, region, vpcID string) ([]security.SecurityGroup, error)
	UpdateRule(ctx context.Context, provider, account, region, groupID string, rule security.SecurityRule) error
	DeleteRule(ctx context.Context, provider, account, region, groupID, ruleID string) error
}
