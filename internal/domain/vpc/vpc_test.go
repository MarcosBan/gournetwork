package vpc_test

import (
	"net"
	"testing"

	"gournetwork/internal/domain/vpc"
)

func TestVPCHasSubnets(t *testing.T) {
	v := vpc.VPC{
		ID:   "vpc-001",
		Name: "test-vpc",
		Subnets: []vpc.Subnet{
			{ID: "sub-001", CIDRBlock: "10.0.1.0/24"},
			{ID: "sub-002", CIDRBlock: "10.0.2.0/24"},
		},
	}

	if len(v.Subnets) != 2 {
		t.Errorf("expected 2 subnets, got %d", len(v.Subnets))
	}
	if v.Subnets[0].CIDRBlock != "10.0.1.0/24" {
		t.Errorf("expected CIDR 10.0.1.0/24, got %s", v.Subnets[0].CIDRBlock)
	}
	if v.Subnets[1].CIDRBlock != "10.0.2.0/24" {
		t.Errorf("expected CIDR 10.0.2.0/24, got %s", v.Subnets[1].CIDRBlock)
	}
}

func TestVPCHasRoutes(t *testing.T) {
	v := vpc.VPC{
		ID:   "vpc-001",
		Name: "test-vpc",
		Routes: []vpc.Route{
			{Destination: "0.0.0.0/0", NextHop: "igw-001"},
			{Destination: "10.1.0.0/16", NextHop: "pcx-001"},
		},
	}

	if len(v.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(v.Routes))
	}
	if v.Routes[0].Destination != "0.0.0.0/0" {
		t.Errorf("expected destination 0.0.0.0/0, got %s", v.Routes[0].Destination)
	}
	if v.Routes[0].NextHop != "igw-001" {
		t.Errorf("expected next hop igw-001, got %s", v.Routes[0].NextHop)
	}
	if v.Routes[1].NextHop != "pcx-001" {
		t.Errorf("expected next hop pcx-001, got %s", v.Routes[1].NextHop)
	}
}

func TestVPCHasPeering(t *testing.T) {
	v := vpc.VPC{
		ID:   "vpc-001",
		Name: "test-vpc",
		Peerings: []vpc.Peering{
			{
				ID:       "pcx-001",
				Name:     "peer-to-vpc-002",
				LocalVPC: "vpc-001",
				PeerVPC:  "vpc-002",
				State:    "active",
				Provider: vpc.ProviderAWS,
			},
		},
	}

	if len(v.Peerings) != 1 {
		t.Errorf("expected 1 peering, got %d", len(v.Peerings))
	}
	if v.Peerings[0].PeerVPC != "vpc-002" {
		t.Errorf("expected peer VPC vpc-002, got %s", v.Peerings[0].PeerVPC)
	}
	if v.Peerings[0].State != "active" {
		t.Errorf("expected state active, got %s", v.Peerings[0].State)
	}
}

func TestVPCHasVPN(t *testing.T) {
	v := vpc.VPC{
		ID:   "vpc-001",
		Name: "test-vpc",
		VPNs: []vpc.VPN{
			{
				ID:            "vpn-001",
				Name:          "vpn-to-onprem",
				LocalVPC:      "vpc-001",
				RemoteGateway: "203.0.113.1",
				State:         "available",
				Provider:      vpc.ProviderAWS,
				Tunnels: []vpc.VPNTunnel{
					{
						ID:       "tunnel-001",
						LocalIP:  "169.254.0.1",
						RemoteIP: "169.254.0.2",
						State:    "UP",
					},
				},
			},
		},
	}

	if len(v.VPNs) != 1 {
		t.Errorf("expected 1 VPN, got %d", len(v.VPNs))
	}
	if v.VPNs[0].RemoteGateway != "203.0.113.1" {
		t.Errorf("expected remote gateway 203.0.113.1, got %s", v.VPNs[0].RemoteGateway)
	}
	if len(v.VPNs[0].Tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(v.VPNs[0].Tunnels))
	}
	if v.VPNs[0].Tunnels[0].State != "UP" {
		t.Errorf("expected tunnel state UP, got %s", v.VPNs[0].Tunnels[0].State)
	}
}

func TestSubnetBelongsToProvider(t *testing.T) {
	t.Run("AWS subnet", func(t *testing.T) {
		s := vpc.Subnet{
			ID:       "sub-001",
			Provider: vpc.ProviderAWS,
		}
		if s.Provider != vpc.ProviderAWS {
			t.Errorf("expected provider aws, got %s", s.Provider)
		}
	})

	t.Run("GCP subnet", func(t *testing.T) {
		s := vpc.Subnet{
			ID:       "sub-002",
			Provider: vpc.ProviderGCP,
		}
		if s.Provider != vpc.ProviderGCP {
			t.Errorf("expected provider gcp, got %s", s.Provider)
		}
	})
}

func TestRouteDestinationFormat(t *testing.T) {
	validCIDRs := []string{
		"0.0.0.0/0",
		"10.0.0.0/8",
		"192.168.1.0/24",
		"172.16.0.0/12",
	}

	for _, cidr := range validCIDRs {
		t.Run(cidr, func(t *testing.T) {
			route := vpc.Route{
				Destination: cidr,
				NextHop:     "igw-001",
			}
			_, _, err := net.ParseCIDR(route.Destination)
			if err != nil {
				t.Errorf("route destination %q is not a valid CIDR: %v", route.Destination, err)
			}
		})
	}
}
