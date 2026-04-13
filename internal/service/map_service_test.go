package service_test

import (
	"context"
	"testing"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
	"gournetwork/internal/service"
)

func TestMapService_GetNetworkMap_ListsVPCs(t *testing.T) {
	repo := &mockVPCRepo{
		ListVPCsFn: func(_ context.Context, _, _, _ string) ([]vpc.VPC, error) {
			return []vpc.VPC{
				{ID: "vpc-001", Name: "vpc-a", Region: "us-east-1", Provider: vpc.ProviderAWS},
				{ID: "vpc-002", Name: "vpc-b", Region: "us-east-1", Provider: vpc.ProviderAWS},
			}, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", repo)
	svc := service.NewMapService(reg)

	result, err := svc.GetNetworkMap(context.Background(), []network.ProviderQuery{
		{Provider: "aws", Account: "prod", Region: "us-east-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VPCs) != 2 {
		t.Errorf("expected 2 VPCs, got %d", len(result.VPCs))
	}
}

func TestMapService_GetNetworkMap_BuildsConnections(t *testing.T) {
	repo := &mockVPCRepo{
		ListVPCsFn: func(_ context.Context, _, _, _ string) ([]vpc.VPC, error) {
			return []vpc.VPC{
				{
					ID: "vpc-001", Provider: vpc.ProviderAWS,
					Peerings: []vpc.Peering{{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"}},
				},
				{ID: "vpc-002", Provider: vpc.ProviderAWS},
			}, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", repo)
	svc := service.NewMapService(reg)

	result, err := svc.GetNetworkMap(context.Background(), []network.ProviderQuery{
		{Provider: "aws", Account: "prod", Region: "us-east-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Connections) != 1 {
		t.Errorf("expected 1 connection, got %d", len(result.Connections))
	}
	if result.Connections[0].Type != "peering" {
		t.Errorf("expected peering connection, got %q", result.Connections[0].Type)
	}
}

func TestMapService_GetNetworkMap_MultipleProviders(t *testing.T) {
	awsRepo := &mockVPCRepo{
		ListVPCsFn: func(_ context.Context, _, _, _ string) ([]vpc.VPC, error) {
			return []vpc.VPC{{ID: "vpc-001", Provider: vpc.ProviderAWS}}, nil
		},
	}
	gcpRepo := &mockVPCRepo{
		ListVPCsFn: func(_ context.Context, _, _, _ string) ([]vpc.VPC, error) {
			return []vpc.VPC{{ID: "net-001", Provider: vpc.ProviderGCP}}, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", awsRepo)
	reg.RegisterVPC("gcp", gcpRepo)
	svc := service.NewMapService(reg)

	result, err := svc.GetNetworkMap(context.Background(), []network.ProviderQuery{
		{Provider: "aws", Account: "prod", Region: "us-east-1"},
		{Provider: "gcp", Account: "myproject", Region: "us-central1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VPCs) != 2 {
		t.Errorf("expected 2 VPCs from both providers, got %d", len(result.VPCs))
	}
}

func TestMapService_GetNetworkMap_Empty(t *testing.T) {
	svc := service.NewMapService(secondary.NewProviderRegistry())

	result, err := svc.GetNetworkMap(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.VPCs) != 0 {
		t.Errorf("expected 0 VPCs, got %d", len(result.VPCs))
	}
}
