package service_test

import (
	"context"
	"errors"
	"testing"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
	"gournetwork/internal/service"
)

// mockVPCRepo implements CloudVPCRepository for tests.
type mockVPCRepo struct {
	GetVPCFn       func(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error)
	ListVPCsFn     func(ctx context.Context, provider, account, region string) ([]vpc.VPC, error)
	UpdateRoutesFn func(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error
	ListSubnetsFn  func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error)
	ListPeeringsFn func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error)
	ListVPNsFn     func(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error)
}

func (m *mockVPCRepo) GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	if m.GetVPCFn != nil {
		return m.GetVPCFn(ctx, provider, account, region, vpcID)
	}
	return nil, nil
}
func (m *mockVPCRepo) ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, account, region)
	}
	return nil, nil
}
func (m *mockVPCRepo) UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error {
	if m.UpdateRoutesFn != nil {
		return m.UpdateRoutesFn(ctx, provider, account, region, vpcID, routes)
	}
	return nil
}
func (m *mockVPCRepo) ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error) {
	if m.ListSubnetsFn != nil {
		return m.ListSubnetsFn(ctx, provider, account, region, vpcID)
	}
	return nil, nil
}
func (m *mockVPCRepo) ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error) {
	if m.ListPeeringsFn != nil {
		return m.ListPeeringsFn(ctx, provider, account, region, vpcID)
	}
	return nil, nil
}
func (m *mockVPCRepo) ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error) {
	if m.ListVPNsFn != nil {
		return m.ListVPNsFn(ctx, provider, account, region, vpcID)
	}
	return nil, nil
}

// mockStorage implements StorageRepository for tests.
type mockStorage struct {
	SaveVPCFn            func(ctx context.Context, v *vpc.VPC) error
	SaveSecurityGroupFn  func(ctx context.Context, sg *security.SecurityGroup) error
	ListVPCsFn           func(ctx context.Context, provider, region string) ([]vpc.VPC, error)
	ListSecurityGroupsFn func(ctx context.Context, provider, vpcID string) ([]security.SecurityGroup, error)
}

func (m *mockStorage) SaveVPC(ctx context.Context, v *vpc.VPC) error {
	if m.SaveVPCFn != nil {
		return m.SaveVPCFn(ctx, v)
	}
	return nil
}
func (m *mockStorage) SaveSecurityGroup(ctx context.Context, sg *security.SecurityGroup) error {
	if m.SaveSecurityGroupFn != nil {
		return m.SaveSecurityGroupFn(ctx, sg)
	}
	return nil
}
func (m *mockStorage) ListVPCs(ctx context.Context, provider, region string) ([]vpc.VPC, error) {
	if m.ListVPCsFn != nil {
		return m.ListVPCsFn(ctx, provider, region)
	}
	return nil, nil
}
func (m *mockStorage) ListSecurityGroups(ctx context.Context, provider, vpcID string) ([]security.SecurityGroup, error) {
	if m.ListSecurityGroupsFn != nil {
		return m.ListSecurityGroupsFn(ctx, provider, vpcID)
	}
	return nil, nil
}

func newTestRegistry(vpcRepo secondary.CloudVPCRepository) *secondary.ProviderRegistry {
	reg := secondary.NewProviderRegistry()
	if vpcRepo != nil {
		reg.RegisterVPC("aws", vpcRepo)
	}
	return reg
}

func TestVPCService_DescribeVPC_Success(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Name: "test-vpc", CIDRBlock: "10.0.0.0/16"}, nil
		},
	}
	svc := service.NewVPCService(newTestRegistry(repo), &mockStorage{})

	v, err := svc.DescribeVPC(context.Background(), "aws", "prod", "us-east-1", "vpc-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "vpc-001" {
		t.Errorf("expected vpc-001, got %q", v.ID)
	}
}

func TestVPCService_DescribeVPC_EnrichesSubnets(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001"}, nil
		},
		ListSubnetsFn: func(_ context.Context, _, _, _, _ string) ([]vpc.Subnet, error) {
			return []vpc.Subnet{{ID: "sub-001"}}, nil
		},
	}
	svc := service.NewVPCService(newTestRegistry(repo), &mockStorage{})

	v, err := svc.DescribeVPC(context.Background(), "aws", "prod", "us-east-1", "vpc-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(v.Subnets))
	}
}

func TestVPCService_DescribeVPC_ProviderNotConfigured(t *testing.T) {
	svc := service.NewVPCService(secondary.NewProviderRegistry(), &mockStorage{})

	_, err := svc.DescribeVPC(context.Background(), "azure", "prod", "us-east-1", "vpc-001")
	if err == nil {
		t.Fatal("expected error for unconfigured provider")
	}
}

func TestVPCService_InsertVPC_SavesAndReturns(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Region: "us-east-1", Provider: vpc.ProviderAWS}, nil
		},
	}
	var saved *vpc.VPC
	store := &mockStorage{
		SaveVPCFn: func(_ context.Context, v *vpc.VPC) error { saved = v; return nil },
	}
	svc := service.NewVPCService(newTestRegistry(repo), store)

	v, err := svc.InsertVPC(context.Background(), "aws", "prod", "us-east-1", "vpc-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID != "vpc-001" {
		t.Errorf("expected vpc-001, got %q", v.ID)
	}
	if saved == nil {
		t.Fatal("expected SaveVPC to be called")
	}
}

func TestVPCService_InsertVPC_StorageError(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{ID: "vpc-001", Region: "us-east-1", Provider: vpc.ProviderAWS}, nil
		},
	}
	store := &mockStorage{
		SaveVPCFn: func(_ context.Context, _ *vpc.VPC) error { return errors.New("disk full") },
	}
	svc := service.NewVPCService(newTestRegistry(repo), store)

	_, err := svc.InsertVPC(context.Background(), "aws", "prod", "us-east-1", "vpc-001")
	if err == nil {
		t.Fatal("expected error when storage fails")
	}
}
