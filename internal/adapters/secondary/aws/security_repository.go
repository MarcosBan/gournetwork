package aws

import (
	"context"
	"errors"

	"gournetwork/internal/domain/security"
)

// AWSSecurityRepository is the AWS implementation of CloudSecurityRepository.
type AWSSecurityRepository struct{}

// NewAWSSecurityRepository creates a new AWSSecurityRepository.
func NewAWSSecurityRepository() *AWSSecurityRepository {
	return &AWSSecurityRepository{}
}

// GetSecurityGroup retrieves a security group by ID from AWS. Not yet implemented.
func (r *AWSSecurityRepository) GetSecurityGroup(ctx context.Context, provider, region, groupID string) (*security.SecurityGroup, error) {
	return nil, errors.New("not implemented")
}

// ListSecurityGroups lists all security groups for a VPC in AWS. Not yet implemented.
func (r *AWSSecurityRepository) ListSecurityGroups(ctx context.Context, provider, region, vpcID string) ([]security.SecurityGroup, error) {
	return nil, errors.New("not implemented")
}

// UpdateRule updates a security rule within a group in AWS. Not yet implemented.
func (r *AWSSecurityRepository) UpdateRule(ctx context.Context, provider, region, groupID string, rule security.SecurityRule) error {
	return errors.New("not implemented")
}

// DeleteRule deletes a security rule from a group in AWS. Not yet implemented.
func (r *AWSSecurityRepository) DeleteRule(ctx context.Context, provider, region, groupID, ruleID string) error {
	return errors.New("not implemented")
}
