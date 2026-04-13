package primary

import (
	"context"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// VPCService defines the use cases for VPC operations.
type VPCService interface {
	DescribeVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
	InsertVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
}

// SecurityService defines the use cases for security group operations.
type SecurityService interface {
	DescribeSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error)
	InsertSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error)
	RemoveRule(ctx context.Context, provider, account, region, groupID, ruleID string) error
}

// AnalyseService defines the use cases for connectivity analysis.
type AnalyseService interface {
	AnalyseConnectivity(ctx context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error)
}

// MapService defines the use cases for the network topology map.
type MapService interface {
	GetNetworkMap(ctx context.Context, providers []network.ProviderQuery) (*network.NetworkMap, error)
}
