package secondary

import (
	"context"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// StorageRepository defines the interface for persisting and querying scraped cloud resources locally.
type StorageRepository interface {
	SaveVPC(ctx context.Context, v *vpc.VPC) error
	SaveSecurityGroup(ctx context.Context, sg *security.SecurityGroup) error
	ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error)
	ListSecurityGroups(ctx context.Context, provider, vpcID string) ([]security.SecurityGroup, error)
}
