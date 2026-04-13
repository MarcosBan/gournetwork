package network

import (
	"fmt"
	"net"
	"time"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// ProviderQuery identifies a provider + account + region combination to query.
type ProviderQuery struct {
	Provider string
	Account  string
	Region   string
}

// AnalyseRequest is the input for a connectivity analysis.
type AnalyseRequest struct {
	SourceProvider string
	SourceAccount  string
	SourceRegion   string
	SourceVPC      string

	DestProvider string
	DestAccount  string
	DestRegion   string
	DestCIDR     string
}

// ConnectivityResult represents the result of a connectivity analysis.
type ConnectivityResult struct {
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Connected   bool     `json:"connected"`
	Path        []string `json:"path"`
	Reason      string   `json:"reason"`
}

// Connection represents a connection between two VPCs.
type Connection struct {
	SourceVPC      string            `json:"source_vpc"`
	DestinationVPC string            `json:"destination_vpc"`
	Type           string            `json:"type"` // "peering", "vpn", or "route"
	Details        map[string]string `json:"details"`
}

// NetworkMap represents the full network topology overview.
type NetworkMap struct {
	VPCs        []vpc.VPC    `json:"vpcs"`
	Connections []Connection `json:"connections"`
	GeneratedAt time.Time    `json:"generated_at"`
}

// Node represents a vertex in the network topology graph.
type Node struct {
	VPCID    string
	Provider string
	Account  string
	Region   string
	CIDR     string
}

// Edge represents a link between two nodes in the graph.
type Edge struct {
	From    string // VPC ID
	To      string // VPC ID
	Type    string // "peering", "vpn", "route"
	State   string
	Details map[string]string
}

// Graph models the multicloud network topology as an adjacency list.
type Graph struct {
	Nodes map[string]*Node
	Edges map[string][]Edge // key = source VPC ID
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string][]Edge),
	}
}

// AddVPC adds a VPC as a node in the graph and extracts its peering/VPN edges.
func (g *Graph) AddVPC(v vpc.VPC, account string) {
	g.Nodes[v.ID] = &Node{
		VPCID:    v.ID,
		Provider: string(v.Provider),
		Account:  account,
		Region:   v.Region,
		CIDR:     v.CIDRBlock,
	}

	for _, p := range v.Peerings {
		if p.State != "active" && p.State != "ACTIVE" {
			continue
		}
		g.Edges[v.ID] = append(g.Edges[v.ID], Edge{
			From:  v.ID,
			To:    p.PeerVPC,
			Type:  "peering",
			State: p.State,
			Details: map[string]string{
				"peering_id": p.ID,
				"name":       p.Name,
			},
		})
	}

	for _, vpn := range v.VPNs {
		if vpn.State != "available" && vpn.State != "ESTABLISHED" {
			continue
		}
		g.Edges[v.ID] = append(g.Edges[v.ID], Edge{
			From:  v.ID,
			To:    vpn.RemoteGateway,
			Type:  "vpn",
			State: vpn.State,
			Details: map[string]string{
				"vpn_id":         vpn.ID,
				"name":           vpn.Name,
				"remote_gateway": vpn.RemoteGateway,
				"tunnels":        tunnelSummary(vpn.Tunnels),
			},
		})
	}
}

// Connections returns all edges as Connection structs.
func (g *Graph) Connections() []Connection {
	var conns []Connection
	for _, edges := range g.Edges {
		for _, e := range edges {
			conns = append(conns, Connection{
				SourceVPC:      e.From,
				DestinationVPC: e.To,
				Type:           e.Type,
				Details:        e.Details,
			})
		}
	}
	return conns
}

// HasRouteTo checks if a VPC has a route whose destination contains the given CIDR.
func HasRouteTo(v *vpc.VPC, destCIDR string) (vpc.Route, bool) {
	_, destNet, err := net.ParseCIDR(destCIDR)
	if err != nil {
		return vpc.Route{}, false
	}
	for _, r := range v.Routes {
		_, routeNet, err := net.ParseCIDR(r.Destination)
		if err != nil {
			continue
		}
		if routeNet.Contains(destNet.IP) || destNet.Contains(routeNet.IP) {
			return r, true
		}
	}
	return vpc.Route{}, false
}

// SecurityAllows checks if any security group allows traffic to the destination CIDR
// on the given protocol and port.
func SecurityAllows(groups []security.SecurityGroup, destCIDR, protocol string, port int) bool {
	_, destNet, err := net.ParseCIDR(destCIDR)
	if err != nil {
		return false
	}
	for _, sg := range groups {
		for _, rule := range sg.Rules {
			if rule.Direction != "egress" {
				continue
			}
			if rule.Action != "allow" {
				continue
			}
			if rule.Protocol != "-1" && rule.Protocol != protocol {
				continue
			}
			if port != 0 && (port < rule.PortRange.From || port > rule.PortRange.To) {
				if rule.PortRange.From != 0 || rule.PortRange.To != 0 {
					continue
				}
			}
			for _, dst := range rule.Destinations {
				_, ruleNet, err := net.ParseCIDR(dst)
				if err != nil {
					continue
				}
				if ruleNet.Contains(destNet.IP) {
					return true
				}
			}
			if len(rule.Destinations) == 0 {
				for _, src := range rule.Sources {
					if src == "0.0.0.0/0" {
						return true
					}
					_, ruleNet, err := net.ParseCIDR(src)
					if err != nil {
						continue
					}
					if ruleNet.Contains(destNet.IP) {
						return true
					}
				}
			}
		}
	}
	return false
}

func tunnelSummary(tunnels []vpc.VPNTunnel) string {
	up := 0
	for _, t := range tunnels {
		if t.State == "UP" || t.State == "ESTABLISHED" {
			up++
		}
	}
	if len(tunnels) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d up", up, len(tunnels))
}
