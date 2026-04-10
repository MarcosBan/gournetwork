package gcp

import (
	"context"
	"errors"

	"gournetwork/internal/domain/vpc"
)

// GCPVPCRepository is the GCP implementation of CloudVPCRepository.
type GCPVPCRepository struct{}

// NewGCPVPCRepository creates a new GCPVPCRepository.
func NewGCPVPCRepository() *GCPVPCRepository {
	return &GCPVPCRepository{}
}

// GetVPC retrieves a VPC by ID from GCP. Not yet implemented.
func (r *GCPVPCRepository) GetVPC(ctx context.Context, provider, region, vpcID string) (*vpc.VPC, error) {
	return nil, errors.New("not implemented")
}

// ListVPCs lists all VPCs in a region from GCP. Not yet implemented.
func (r *GCPVPCRepository) ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error) {
	return nil, errors.New("not implemented")
}

// UpdateRoutes updates the routes for a VPC in GCP. Not yet implemented.
func (r *GCPVPCRepository) UpdateRoutes(ctx context.Context, provider, region, vpcID string, routes []vpc.Route) error {
	return errors.New("not implemented")
}

// ListSubnets lists all subnets for a VPC in GCP. Not yet implemented.
func (r *GCPVPCRepository) ListSubnets(ctx context.Context, provider, region, vpcID string) ([]vpc.Subnet, error) {
	return nil, errors.New("not implemented")
}

// ListPeerings lists all peering connections for a VPC in GCP. Not yet implemented.
func (r *GCPVPCRepository) ListPeerings(ctx context.Context, provider, region, vpcID string) ([]vpc.Peering, error) {
	return nil, errors.New("not implemented")
}

// ListVPNs lists all VPN connections for a VPC in GCP. Not yet implemented.
func (r *GCPVPCRepository) ListVPNs(ctx context.Context, provider, region, vpcID string) ([]vpc.VPN, error) {
	return nil, errors.New("not implemented")
}
