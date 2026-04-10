package secondary

import (
	"context"

	"gournetwork/internal/domain/vpc"
)

// CloudVPCRepository defines the interface for interacting with cloud VPC resources.
type CloudVPCRepository interface {
	GetVPC(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error)
	ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error)
	UpdateRoutes(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error
	ListSubnets(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error)
	ListPeerings(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error)
	ListVPNs(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error)
}
