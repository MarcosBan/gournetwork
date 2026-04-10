package gcp

import (
	"context"
	"errors"

	"gournetwork/internal/domain/security"
)

// GCPSecurityRepository is the GCP implementation of CloudSecurityRepository.
type GCPSecurityRepository struct{}

// NewGCPSecurityRepository creates a new GCPSecurityRepository.
func NewGCPSecurityRepository() *GCPSecurityRepository {
	return &GCPSecurityRepository{}
}

// GetSecurityGroup retrieves a firewall rule set by ID from GCP. Not yet implemented.
func (r *GCPSecurityRepository) GetSecurityGroup(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error) {
	return nil, errors.New("not implemented")
}

// ListSecurityGroups lists all firewall rule sets for a VPC in GCP. Not yet implemented.
func (r *GCPSecurityRepository) ListSecurityGroups(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error) {
	return nil, errors.New("not implemented")
}

// UpdateRule updates a firewall rule within a group in GCP. Not yet implemented.
func (r *GCPSecurityRepository) UpdateRule(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error {
	return errors.New("not implemented")
}

// DeleteRule deletes a firewall rule from a group in GCP. Not yet implemented.
func (r *GCPSecurityRepository) DeleteRule(ctx context.Context, provider, region, groupID, ruleID string) error {
	return errors.New("not implemented")
}
