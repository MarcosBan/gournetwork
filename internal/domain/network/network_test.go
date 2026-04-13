package network_test

import (
	"testing"
	"time"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

func TestConnectivityResultConnected(t *testing.T) {
	result := network.ConnectivityResult{
		Source:      "vpc-001",
		Destination: "10.1.0.0/16",
		Connected:   true,
		Path:        []string{"vpc-001", "pcx-001", "vpc-002"},
		Reason:      "",
	}

	if !result.Connected {
		t.Error("expected Connected to be true")
	}
	if len(result.Path) == 0 {
		t.Error("expected non-empty Path when Connected is true")
	}
}

func TestConnectivityResultNotConnected(t *testing.T) {
	result := network.ConnectivityResult{
		Source:      "vpc-001",
		Destination: "10.99.0.0/16",
		Connected:   false,
		Path:        []string{},
		Reason:      "no route to destination",
	}

	if result.Connected {
		t.Error("expected Connected to be false")
	}
	if result.Reason == "" {
		t.Error("expected non-empty Reason when Connected is false")
	}
}

func TestNetworkMapContainsVPCs(t *testing.T) {
	vpcs := []vpc.VPC{
		{ID: "vpc-001", Name: "vpc-a", Region: "us-east-1", Provider: vpc.ProviderAWS},
		{ID: "vpc-002", Name: "vpc-b", Region: "us-west-2", Provider: vpc.ProviderAWS},
	}

	nm := network.NetworkMap{
		VPCs:        vpcs,
		Connections: []network.Connection{},
		GeneratedAt: time.Now(),
	}

	if len(nm.VPCs) != 2 {
		t.Errorf("expected 2 VPCs in network map, got %d", len(nm.VPCs))
	}
	if nm.VPCs[0].ID != "vpc-001" {
		t.Errorf("expected first VPC ID vpc-001, got %s", nm.VPCs[0].ID)
	}
}

func TestNetworkMapConnection(t *testing.T) {
	conn := network.Connection{
		SourceVPC:      "vpc-001",
		DestinationVPC: "vpc-002",
		Type:           "peering",
		Details:        map[string]string{"peering_id": "pcx-001"},
	}

	if conn.SourceVPC != "vpc-001" {
		t.Errorf("expected SourceVPC vpc-001, got %s", conn.SourceVPC)
	}
	if conn.DestinationVPC != "vpc-002" {
		t.Errorf("expected DestinationVPC vpc-002, got %s", conn.DestinationVPC)
	}
}

func TestConnectionTypes(t *testing.T) {
	validTypes := []string{"peering", "vpn", "route"}

	for _, connType := range validTypes {
		t.Run(connType, func(t *testing.T) {
			conn := network.Connection{
				SourceVPC:      "vpc-001",
				DestinationVPC: "vpc-002",
				Type:           connType,
			}
			found := false
			for _, valid := range validTypes {
				if conn.Type == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("connection type %q is not a valid type", conn.Type)
			}
		})
	}
}

// --- Graph tests ---

func TestGraph_AddVPC_CreatesNode(t *testing.T) {
	g := network.NewGraph()
	g.AddVPC(vpc.VPC{
		ID:        "vpc-001",
		Provider:  vpc.ProviderAWS,
		Region:    "us-east-1",
		CIDRBlock: "10.0.0.0/16",
	}, "prod")

	if _, ok := g.Nodes["vpc-001"]; !ok {
		t.Error("expected node vpc-001 to be added")
	}
}

func TestGraph_AddVPC_CreatesPeeringEdge(t *testing.T) {
	g := network.NewGraph()
	g.AddVPC(vpc.VPC{
		ID:       "vpc-001",
		Provider: vpc.ProviderAWS,
		Peerings: []vpc.Peering{
			{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"},
		},
	}, "prod")

	edges := g.Edges["vpc-001"]
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Type != "peering" {
		t.Errorf("expected peering edge, got %q", edges[0].Type)
	}
	if edges[0].To != "vpc-002" {
		t.Errorf("expected edge to vpc-002, got %q", edges[0].To)
	}
}

func TestGraph_AddVPC_IgnoresInactivePeering(t *testing.T) {
	g := network.NewGraph()
	g.AddVPC(vpc.VPC{
		ID:       "vpc-001",
		Provider: vpc.ProviderAWS,
		Peerings: []vpc.Peering{
			{ID: "pcx-001", PeerVPC: "vpc-002", State: "deleted"},
		},
	}, "prod")

	if len(g.Edges["vpc-001"]) != 0 {
		t.Error("expected no edges for inactive peering")
	}
}

func TestGraph_AddVPC_CreatesVPNEdge(t *testing.T) {
	g := network.NewGraph()
	g.AddVPC(vpc.VPC{
		ID:       "vpc-001",
		Provider: vpc.ProviderAWS,
		VPNs: []vpc.VPN{
			{ID: "vpn-001", RemoteGateway: "203.0.113.1", State: "available"},
		},
	}, "prod")

	edges := g.Edges["vpc-001"]
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Type != "vpn" {
		t.Errorf("expected vpn edge, got %q", edges[0].Type)
	}
}

func TestGraph_Connections_ReturnsFlatList(t *testing.T) {
	g := network.NewGraph()
	g.AddVPC(vpc.VPC{
		ID:       "vpc-001",
		Provider: vpc.ProviderAWS,
		Peerings: []vpc.Peering{
			{ID: "pcx-001", PeerVPC: "vpc-002", State: "active"},
			{ID: "pcx-002", PeerVPC: "vpc-003", State: "active"},
		},
	}, "prod")

	conns := g.Connections()
	if len(conns) != 2 {
		t.Errorf("expected 2 connections, got %d", len(conns))
	}
}

// --- HasRouteTo tests ---

func TestHasRouteTo_MatchingRoute(t *testing.T) {
	v := &vpc.VPC{
		Routes: []vpc.Route{
			{Destination: "10.1.0.0/16", NextHop: "pcx-001"},
		},
	}
	route, found := network.HasRouteTo(v, "10.1.0.0/24")
	if !found {
		t.Error("expected route to be found")
	}
	if route.NextHop != "pcx-001" {
		t.Errorf("expected next hop pcx-001, got %q", route.NextHop)
	}
}

func TestHasRouteTo_NoMatch(t *testing.T) {
	v := &vpc.VPC{
		Routes: []vpc.Route{
			{Destination: "10.0.0.0/16", NextHop: "local"},
		},
	}
	_, found := network.HasRouteTo(v, "172.16.0.0/16")
	if found {
		t.Error("expected no route match")
	}
}

func TestHasRouteTo_DefaultRoute(t *testing.T) {
	v := &vpc.VPC{
		Routes: []vpc.Route{
			{Destination: "0.0.0.0/0", NextHop: "igw-001"},
		},
	}
	_, found := network.HasRouteTo(v, "10.1.0.0/16")
	if !found {
		t.Error("expected default route to match any destination")
	}
}

// --- SecurityAllows tests ---

func TestSecurityAllows_EgressAllowed(t *testing.T) {
	groups := []security.SecurityGroup{
		{
			Rules: []security.SecurityRule{
				{
					Direction: "egress",
					Protocol:  "-1",
					Action:    "allow",
					Sources:   []string{"0.0.0.0/0"},
				},
			},
		},
	}
	if !network.SecurityAllows(groups, "10.1.0.0/16", "-1", 0) {
		t.Error("expected egress to be allowed with 0.0.0.0/0")
	}
}

func TestSecurityAllows_EgressBlocked(t *testing.T) {
	groups := []security.SecurityGroup{
		{
			Rules: []security.SecurityRule{
				{
					Direction:    "egress",
					Protocol:     "tcp",
					Action:       "allow",
					PortRange:    security.PortRange{From: 80, To: 80},
					Destinations: []string{"192.168.0.0/16"},
				},
			},
		},
	}
	if network.SecurityAllows(groups, "10.1.0.0/16", "tcp", 443) {
		t.Error("expected egress to be blocked — wrong port and wrong CIDR")
	}
}
