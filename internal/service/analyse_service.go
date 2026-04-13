package service

import (
	"context"
	"fmt"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
)

// AnalyseService orchestrates multicloud connectivity analysis.
type AnalyseService struct {
	registry *secondary.ProviderRegistry
}

// NewAnalyseService creates a new AnalyseService.
func NewAnalyseService(registry *secondary.ProviderRegistry) *AnalyseService {
	return &AnalyseService{registry: registry}
}

// AnalyseConnectivity checks whether the source VPC can reach the destination CIDR
// by examining routes, peerings, VPNs, and security groups along the path.
func (s *AnalyseService) AnalyseConnectivity(ctx context.Context, req network.AnalyseRequest) (*network.ConnectivityResult, error) {
	result := &network.ConnectivityResult{
		Source:      req.SourceVPC,
		Destination: req.DestCIDR,
	}

	// 1. Fetch source VPC
	vpcRepo := s.registry.VPCRepo(req.SourceProvider)
	if vpcRepo == nil {
		result.Reason = fmt.Sprintf("source provider %q not configured", req.SourceProvider)
		return result, nil
	}

	sourceVPC, err := vpcRepo.GetVPC(ctx, req.SourceProvider, req.SourceAccount, req.SourceRegion, req.SourceVPC)
	if err != nil || sourceVPC == nil {
		result.Reason = "source VPC not found"
		return result, nil
	}

	// 2. Check if the source VPC has a route to the destination CIDR
	route, hasRoute := network.HasRouteTo(sourceVPC, req.DestCIDR)
	if !hasRoute {
		result.Reason = fmt.Sprintf("no route to %s found in VPC %s", req.DestCIDR, req.SourceVPC)
		return result, nil
	}

	path := []string{req.SourceVPC}

	// 3. Determine how the route is connected (peering, VPN, gateway)
	nextHop := route.NextHop
	connectionType := classifyNextHop(nextHop, sourceVPC)

	switch connectionType {
	case "peering":
		peerVPC := findPeeringTarget(sourceVPC, nextHop)
		if peerVPC == "" {
			result.Reason = fmt.Sprintf("peering connection %s references unknown peer VPC", nextHop)
			return result, nil
		}
		path = append(path, fmt.Sprintf("peering:%s", nextHop), peerVPC)

	case "vpn":
		vpnInfo := findVPNByNextHop(sourceVPC, nextHop)
		if vpnInfo == "" {
			path = append(path, fmt.Sprintf("vpn:%s", nextHop))
		} else {
			path = append(path, fmt.Sprintf("vpn:%s", vpnInfo))
		}

	default:
		path = append(path, fmt.Sprintf("gateway:%s", nextHop))
	}

	path = append(path, req.DestCIDR)

	// 4. Check security groups allow egress to the destination
	secRepo := s.registry.SecurityRepo(req.SourceProvider)
	if secRepo != nil {
		groups, err := secRepo.ListSecurityGroups(ctx, req.SourceProvider, req.SourceAccount, req.SourceRegion, req.SourceVPC)
		if err == nil && len(groups) > 0 {
			if !network.SecurityAllows(groups, req.DestCIDR, "-1", 0) {
				result.Path = path
				result.Reason = fmt.Sprintf("security groups on VPC %s block egress to %s", req.SourceVPC, req.DestCIDR)
				return result, nil
			}
		}
		// If no security groups or cannot fetch, assume open (default AWS behavior).
	}

	result.Connected = true
	result.Path = path
	return result, nil
}

// classifyNextHop determines whether a route's next hop is a peering, VPN, or generic gateway.
func classifyNextHop(nextHop string, v *vpc.VPC) string {
	for _, p := range v.Peerings {
		if p.ID == nextHop {
			return "peering"
		}
	}
	for _, vpn := range v.VPNs {
		if vpn.ID == nextHop || vpn.RemoteGateway == nextHop {
			return "vpn"
		}
	}
	return "gateway"
}

// findPeeringTarget returns the peer VPC ID for a peering connection.
func findPeeringTarget(v *vpc.VPC, peeringID string) string {
	for _, p := range v.Peerings {
		if p.ID == peeringID {
			return p.PeerVPC
		}
	}
	return ""
}

// findVPNByNextHop returns a VPN identifier for the given next hop.
func findVPNByNextHop(v *vpc.VPC, nextHop string) string {
	for _, vpn := range v.VPNs {
		if vpn.ID == nextHop || vpn.RemoteGateway == nextHop {
			return vpn.ID
		}
	}
	return ""
}
