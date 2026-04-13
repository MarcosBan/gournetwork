package service

import (
	"context"
	"fmt"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/ports/secondary"
)

// SecurityService orchestrates security group operations across providers.
type SecurityService struct {
	registry *secondary.ProviderRegistry
	store    secondary.StorageRepository
}

// NewSecurityService creates a new SecurityService.
func NewSecurityService(registry *secondary.ProviderRegistry, store secondary.StorageRepository) *SecurityService {
	return &SecurityService{registry: registry, store: store}
}

// DescribeSecurityGroup fetches a security group from the appropriate cloud provider.
func (s *SecurityService) DescribeSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	repo := s.registry.SecurityRepo(provider)
	if repo == nil {
		return nil, fmt.Errorf("provider %q not configured", provider)
	}

	group, err := repo.GetSecurityGroup(ctx, provider, account, region, groupID)
	if err != nil {
		return nil, fmt.Errorf("get security group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("security group %q not found", groupID)
	}

	return group, nil
}

// InsertSecurityGroup fetches a security group from the cloud provider and persists it to local storage.
func (s *SecurityService) InsertSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	group, err := s.DescribeSecurityGroup(ctx, provider, account, region, groupID)
	if err != nil {
		return nil, err
	}

	if err := s.store.SaveSecurityGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("save security group: %w", err)
	}

	return group, nil
}

// RemoveRule deletes a rule from a security group.
func (s *SecurityService) RemoveRule(ctx context.Context, provider, account, region, groupID, ruleID string) error {
	repo := s.registry.SecurityRepo(provider)
	if repo == nil {
		return fmt.Errorf("provider %q not configured", provider)
	}

	return repo.DeleteRule(ctx, provider, account, region, groupID, ruleID)
}
