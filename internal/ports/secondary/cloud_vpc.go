package secondary

import (
	"context"

	"gournetwork/internal/domain/vpc"
)

// CloudVPCRepository defines the interface for interacting with cloud VPC resources.
// The account parameter identifies the logical credential set to use:
// for AWS it is the account alias (e.g. "production"), for GCP it is the project alias.
type CloudVPCRepository interface {
	GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
	ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error)
	UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error
	ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error)
	ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error)
	ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error)
}
