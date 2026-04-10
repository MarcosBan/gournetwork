package network_test

import (
	"testing"
	"time"

	"gournetwork/internal/domain/network"
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
	if nm.VPCs[1].ID != "vpc-002" {
		t.Errorf("expected second VPC ID vpc-002, got %s", nm.VPCs[1].ID)
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
	if conn.Type != "peering" {
		t.Errorf("expected Type peering, got %s", conn.Type)
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
