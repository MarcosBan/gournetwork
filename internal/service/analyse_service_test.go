package service_test

import (
	"context"
	"testing"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
	"gournetwork/internal/service"
)

func TestAnalyseService_SourceVPCNotFound(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return nil, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", repo)
	svc := service.NewAnalyseService(reg)

	req := network.AnalyseRequest{
		SourceProvider: "aws",
		SourceAccount:  "prod",
		SourceRegion:   "us-east-1",
		SourceVPC:      "vpc-999",
		DestCIDR:       "10.1.0.0/16",
	}

	result, err := svc.AnalyseConnectivity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected not connected when source VPC not found")
	}
	if result.Reason == "" {
		t.Error("expected a reason for disconnection")
	}
}

func TestAnalyseService_NoRouteToDestination(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID:        "vpc-001",
				CIDRBlock: "10.0.0.0/16",
				Routes: []vpc.Route{
					{Destination: "10.0.0.0/16", NextHop: "local"},
				},
			}, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", repo)
	svc := service.NewAnalyseService(reg)

	req := network.AnalyseRequest{
		SourceProvider: "aws",
		SourceAccount:  "prod",
		SourceRegion:   "us-east-1",
		SourceVPC:      "vpc-001",
		DestCIDR:       "172.16.0.0/16",
	}

	result, err := svc.AnalyseConnectivity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected not connected when no route exists")
	}
}

func TestAnalyseService_RouteExists_Connected(t *testing.T) {
	repo := &mockVPCRepo{
		GetVPCFn: func(_ context.Context, _, _, _, _ string) (*vpc.VPC, error) {
			return &vpc.VPC{
				ID:        "vpc-001",
				CIDRBlock: "10.0.0.0/16",
				Routes: []vpc.Route{
					{Destination: "10.0.0.0/16", NextHop: "local"},
					{Destination: "10.1.0.0/16", NextHop: "pcx-001"},
				},
				Peerings: []vpc.Peering{
					{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"},
				},
			}, nil
		},
	}
	reg := secondary.NewProviderRegistry()
	reg.RegisterVPC("aws", repo)
	svc := service.NewAnalyseService(reg)

	req := network.AnalyseRequest{
		SourceProvider: "aws",
		SourceAccount:  "prod",
		SourceRegion:   "us-east-1",
		SourceVPC:      "vpc-001",
		DestCIDR:       "10.1.0.0/16",
	}

	result, err := svc.AnalyseConnectivity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Connected {
		t.Errorf("expected connected, got reason: %s", result.Reason)
	}
	if len(result.Path) == 0 {
		t.Error("expected non-empty path")
	}
}

func TestAnalyseService_ProviderNotConfigured(t *testing.T) {
	svc := service.NewAnalyseService(secondary.NewProviderRegistry())

	req := network.AnalyseRequest{
		SourceProvider: "azure",
		SourceVPC:      "vpc-001",
		DestCIDR:       "10.1.0.0/16",
	}

	result, err := svc.AnalyseConnectivity(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Connected {
		t.Error("expected not connected for unconfigured provider")
	}
}
