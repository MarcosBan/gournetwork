package secondary

import (
	"context"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// StorageRepository defines the interface for persisting scraped cloud resources locally.
type StorageRepository interface {
	SaveVPC(ctx context.Context, v *vpc.VPC) error
	SaveSecurityGroup(ctx context.Context, sg *security.SecurityGroup) error
}
