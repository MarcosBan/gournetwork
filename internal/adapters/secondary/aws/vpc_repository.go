package aws

import (
	"context"
	"errors"

	"gournetwork/internal/domain/vpc"
)

// AWSVPCRepository is the AWS implementation of CloudVPCRepository.
type AWSVPCRepository struct{}

// NewAWSVPCRepository creates a new AWSVPCRepository.
func NewAWSVPCRepository() *AWSVPCRepository {
	return &AWSVPCRepository{}
}

// GetVPC retrieves a VPC by ID from AWS. Not yet implemented.
func (r *AWSVPCRepository) GetVPC(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
	return nil, errors.New("not implemented")
}

// ListVPCs lists all VPCs in a region from AWS. Not yet implemented.
func (r *AWSVPCRepository) ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error) {
	return nil, errors.New("not implemented")
}

// UpdateRoutes updates the routes for a VPC in AWS. Not yet implemented.
func (r *AWSVPCRepository) UpdateRoutes(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error {
	return errors.New("not implemented")
}

// ListSubnets lists all subnets for a VPC in AWS. Not yet implemented.
func (r *AWSVPCRepository) ListSubnets(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error) {
	return nil, errors.New("not implemented")
}

// ListPeerings lists all peering connections for a VPC in AWS. Not yet implemented.
func (r *AWSVPCRepository) ListPeerings(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error) {
	return nil, errors.New("not implemented")
}

// ListVPNs lists all VPN connections for a VPC in AWS. Not yet implemented.
func (r *AWSVPCRepository) ListVPNs(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error) {
	return nil, errors.New("not implemented")
}
