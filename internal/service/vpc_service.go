package service

import (
	"context"
	"fmt"

	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
)

// VPCService orchestrates VPC describe and insert operations across providers.
type VPCService struct {
	registry *secondary.ProviderRegistry
	store    secondary.StorageRepository
}

// NewVPCService creates a new VPCService.
func NewVPCService(registry *secondary.ProviderRegistry, store secondary.StorageRepository) *VPCService {
	return &VPCService{registry: registry, store: store}
}

// DescribeVPC fetches a VPC from the appropriate cloud provider, enriching it with
// subnets, peerings, and VPNs if not already populated.
func (s *VPCService) DescribeVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	repo := s.registry.VPCRepo(provider)
	if repo == nil {
		return nil, fmt.Errorf("provider %q not configured", provider)
	}

	v, err := repo.GetVPC(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, fmt.Errorf("get vpc: %w", err)
	}
	if v == nil {
		return nil, fmt.Errorf("vpc %q not found", vpcID)
	}

	// Enrich with subnets, peerings, VPNs if not already set by the repository.
	if len(v.Subnets) == 0 {
		subnets, _ := repo.ListSubnets(ctx, provider, account, region, vpcID)
		v.Subnets = subnets
	}
	if len(v.Peerings) == 0 {
		peerings, _ := repo.ListPeerings(ctx, provider, account, region, vpcID)
		v.Peerings = peerings
	}
	if len(v.VPNs) == 0 {
		vpns, _ := repo.ListVPNs(ctx, provider, account, region, vpcID)
		v.VPNs = vpns
	}

	return v, nil
}

// InsertVPC fetches a VPC from the cloud provider and persists it to local storage.
func (s *VPCService) InsertVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	v, err := s.DescribeVPC(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}

	if err := s.store.SaveVPC(ctx, v); err != nil {
		return nil, fmt.Errorf("save vpc: %w", err)
	}

	return v, nil
}
