package gcp

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/api/compute/v1"

	"gournetwork/internal/domain/vpc"
)

// --- mock ---

type mockGCPVPCClient struct {
	getNetworkOut      *compute.Network
	getNetworkErr      error
	listNetworksOut    []*compute.Network
	listNetworksErr    error
	listSubnetworksOut []*compute.Subnetwork
	listSubnetworksErr error
	listRoutesOut      []*compute.Route
	listRoutesErr      error
	insertRouteErr     error
	deleteRouteErr     error
	listVpnTunnelsOut  []*compute.VpnTunnel
	listVpnTunnelsErr  error
}

func (m *mockGCPVPCClient) GetNetwork(project, network string) (*compute.Network, error) {
	return m.getNetworkOut, m.getNetworkErr
}
func (m *mockGCPVPCClient) ListNetworks(project string) ([]*compute.Network, error) {
	return m.listNetworksOut, m.listNetworksErr
}
func (m *mockGCPVPCClient) ListSubnetworks(project, region, network string) ([]*compute.Subnetwork, error) {
	return m.listSubnetworksOut, m.listSubnetworksErr
}
func (m *mockGCPVPCClient) ListRoutes(project, network string) ([]*compute.Route, error) {
	return m.listRoutesOut, m.listRoutesErr
}
func (m *mockGCPVPCClient) InsertRoute(project string, route *compute.Route) error {
	return m.insertRouteErr
}
func (m *mockGCPVPCClient) DeleteRoute(project, routeName string) error {
	return m.deleteRouteErr
}
func (m *mockGCPVPCClient) ListVpnTunnels(project, region string) ([]*compute.VpnTunnel, error) {
	return m.listVpnTunnelsOut, m.listVpnTunnelsErr
}

// Compile-time check: mock implements the interface.
var _ gcpVPCClient = (*mockGCPVPCClient)(nil)

// --- GetVPC ---

func TestGCPVPCRepository_GetVPC_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 401: Request had invalid authentication credentials")
	repo := newGCPVPCRepositoryWithClient(&mockGCPVPCClient{getNetworkErr: authErr}, "my-project")
	_, err := repo.GetVPC(context.Background(), "gcp", "test", "us-central1", "my-network")
	if err == nil {
		t.Fatal("expected auth error to be propagated, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

func TestGCPVPCRepository_GetVPC(t *testing.T) {
	mock := &mockGCPVPCClient{
		getNetworkOut: &compute.Network{
			Name:     "prod-network",
			SelfLink: "https://compute.googleapis.com/compute/v1/projects/my-project/global/networks/prod-network",
			Peerings: []*compute.NetworkPeering{},
		},
		listSubnetworksOut: []*compute.Subnetwork{
			{Name: "subnet-1", IpCidrRange: "10.0.1.0/24", Region: "https://...regions/us-central1"},
		},
		listRoutesOut: []*compute.Route{
			{DestRange: "0.0.0.0/0", NextHopGateway: "https://...default-internet-gateway", Priority: 1000},
		},
		listVpnTunnelsOut: []*compute.VpnTunnel{},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	got, err := repo.GetVPC(context.Background(), "gcp", "test", "us-central1", "prod-network")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "prod-network" {
		t.Errorf("expected ID prod-network, got %q", got.ID)
	}
	if got.Provider != vpc.ProviderGCP {
		t.Errorf("expected provider GCP, got %q", got.Provider)
	}
	if len(got.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(got.Subnets))
	}
	if len(got.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(got.Routes))
	}
}

func TestGCPVPCRepository_GetVPC_AuthError_Network(t *testing.T) {
	authErr := errors.New("googleapi: Error 403: Required 'compute.networks.get' permission")
	repo := newGCPVPCRepositoryWithClient(&mockGCPVPCClient{getNetworkErr: authErr}, "my-project")
	_, err := repo.GetVPC(context.Background(), "gcp", "test", "us-central1", "prod-network")
	if err == nil {
		t.Fatal("expected permission error to be propagated, got nil")
	}
}

// --- ListVPCs ---

func TestGCPVPCRepository_ListVPCs(t *testing.T) {
	mock := &mockGCPVPCClient{
		listNetworksOut: []*compute.Network{
			{Name: "network-1"},
			{Name: "network-2"},
		},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	vpcs, err := repo.ListVPCs(context.Background(), "gcp", "test", "us-central1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpcs) != 2 {
		t.Fatalf("expected 2 VPCs, got %d", len(vpcs))
	}
}

func TestGCPVPCRepository_ListVPCs_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 401: Invalid Credentials")
	repo := newGCPVPCRepositoryWithClient(&mockGCPVPCClient{listNetworksErr: authErr}, "my-project")
	_, err := repo.ListVPCs(context.Background(), "gcp", "test", "us-central1")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// --- ListSubnets ---

func TestGCPVPCRepository_ListSubnets(t *testing.T) {
	mock := &mockGCPVPCClient{
		getNetworkOut: &compute.Network{
			Name:     "prod-network",
			SelfLink: "https://compute.googleapis.com/compute/v1/projects/my-project/global/networks/prod-network",
		},
		listSubnetworksOut: []*compute.Subnetwork{
			{Name: "sub-a", IpCidrRange: "10.0.1.0/24", Region: "https://...regions/us-central1"},
			{Name: "sub-b", IpCidrRange: "10.0.2.0/24", Region: "https://...regions/us-central1"},
		},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	subs, err := repo.ListSubnets(context.Background(), "gcp", "test", "us-central1", "prod-network")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subnets, got %d", len(subs))
	}
	if subs[0].CIDRBlock != "10.0.1.0/24" {
		t.Errorf("expected CIDR 10.0.1.0/24, got %q", subs[0].CIDRBlock)
	}
	if subs[0].Provider != vpc.ProviderGCP {
		t.Errorf("expected provider GCP, got %q", subs[0].Provider)
	}
}

// --- ListPeerings ---

func TestGCPVPCRepository_ListPeerings(t *testing.T) {
	mock := &mockGCPVPCClient{
		getNetworkOut: &compute.Network{
			Name: "prod-network",
			Peerings: []*compute.NetworkPeering{
				{
					Name:    "peer-to-staging",
					Network: "https://...networks/staging-network",
					State:   "ACTIVE",
				},
			},
		},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	peerings, err := repo.ListPeerings(context.Background(), "gcp", "test", "us-central1", "prod-network")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peerings) != 1 {
		t.Fatalf("expected 1 peering, got %d", len(peerings))
	}
	if peerings[0].PeerVPC != "staging-network" {
		t.Errorf("expected PeerVPC staging-network, got %q", peerings[0].PeerVPC)
	}
	if peerings[0].State != "ACTIVE" {
		t.Errorf("expected State ACTIVE, got %q", peerings[0].State)
	}
}

// --- ListVPNs ---

func TestGCPVPCRepository_ListVPNs(t *testing.T) {
	mock := &mockGCPVPCClient{
		listVpnTunnelsOut: []*compute.VpnTunnel{
			{
				Name:                 "tunnel-1",
				PeerIp:               "203.0.113.1",
				Status:               "ESTABLISHED",
				LocalTrafficSelector: []string{"10.0.0.0/8"},
			},
		},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	vpns, err := repo.ListVPNs(context.Background(), "gcp", "test", "us-central1", "prod-network")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpns) != 1 {
		t.Fatalf("expected 1 VPN, got %d", len(vpns))
	}
	if vpns[0].RemoteGateway != "203.0.113.1" {
		t.Errorf("expected RemoteGateway 203.0.113.1, got %q", vpns[0].RemoteGateway)
	}
	if vpns[0].State != "ESTABLISHED" {
		t.Errorf("expected State ESTABLISHED, got %q", vpns[0].State)
	}
}

// --- UpdateRoutes ---

func TestGCPVPCRepository_UpdateRoutes(t *testing.T) {
	mock := &mockGCPVPCClient{
		getNetworkOut: &compute.Network{
			Name:     "prod-network",
			SelfLink: "https://...networks/prod-network",
		},
		listRoutesOut: []*compute.Route{
			// A custom route (no NextHopGateway = default-internet-gateway string)
			{Name: "old-route", DestRange: "10.2.0.0/16", NextHopVpnTunnel: "vpn-1"},
		},
	}
	repo := newGCPVPCRepositoryWithClient(mock, "my-project")
	routes := []vpc.Route{
		{Destination: "10.3.0.0/16", NextHop: "https://...default-internet-gateway"},
	}
	if err := repo.UpdateRoutes(context.Background(), "gcp", "test", "us-central1", "prod-network", routes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGCPVPCRepository_UpdateRoutes_AuthError(t *testing.T) {
	authErr := errors.New("googleapi: Error 403: Permission denied")
	repo := newGCPVPCRepositoryWithClient(&mockGCPVPCClient{getNetworkErr: authErr}, "my-project")
	err := repo.UpdateRoutes(context.Background(), "gcp", "test", "us-central1", "prod-network", nil)
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- registry account lookup ---

func TestGCPVPCRepository_UnknownProject(t *testing.T) {
	registry := &GCPClientRegistry{entries: map[string]*gcpClientEntry{}}
	repo := NewGCPVPCRepositoryFromRegistry(registry)
	_, err := repo.GetVPC(context.Background(), "gcp", "nonexistent", "us-central1", "my-network")
	if err == nil {
		t.Fatal("expected error for unknown project alias, got nil")
	}
}
