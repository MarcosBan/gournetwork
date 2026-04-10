package gcp

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/compute/v1"

	"gournetwork/internal/domain/vpc"
)

// gcpVPCClient defines the GCP Compute operations used by GCPVPCRepository.
// Wrapping the concrete compute.Service methods behind an interface enables test injection.
type gcpVPCClient interface {
	GetNetwork(project, network string) (*compute.Network, error)
	ListNetworks(project string) ([]*compute.Network, error)
	ListSubnetworks(project, region, network string) ([]*compute.Subnetwork, error)
	ListRoutes(project, network string) ([]*compute.Route, error)
	InsertRoute(project string, route *compute.Route) error
	DeleteRoute(project, routeName string) error
	ListVpnTunnels(project, region string) ([]*compute.VpnTunnel, error)
}

// computeServiceWrapper implements gcpVPCClient using the real GCP compute.Service.
type computeServiceWrapper struct {
	svc *compute.Service
}

func (w *computeServiceWrapper) GetNetwork(project, network string) (*compute.Network, error) {
	return w.svc.Networks.Get(project, network).Do()
}

func (w *computeServiceWrapper) ListNetworks(project string) ([]*compute.Network, error) {
	list, err := w.svc.Networks.List(project).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (w *computeServiceWrapper) ListSubnetworks(project, region, network string) ([]*compute.Subnetwork, error) {
	list, err := w.svc.Subnetworks.List(project, region).Filter(fmt.Sprintf("network = \"%s\"", network)).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (w *computeServiceWrapper) ListRoutes(project, network string) ([]*compute.Route, error) {
	list, err := w.svc.Routes.List(project).Filter(fmt.Sprintf("network = \"%s\"", network)).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (w *computeServiceWrapper) InsertRoute(project string, route *compute.Route) error {
	op, err := w.svc.Routes.Insert(project, route).Do()
	if err != nil {
		return err
	}
	return waitForGlobalOp(w.svc, project, op.Name)
}

func (w *computeServiceWrapper) DeleteRoute(project, routeName string) error {
	op, err := w.svc.Routes.Delete(project, routeName).Do()
	if err != nil {
		return err
	}
	return waitForGlobalOp(w.svc, project, op.Name)
}

func (w *computeServiceWrapper) ListVpnTunnels(project, region string) ([]*compute.VpnTunnel, error) {
	list, err := w.svc.VpnTunnels.List(project, region).Do()
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// waitForGlobalOp polls a global operation until it is done.
func waitForGlobalOp(svc *compute.Service, project, opName string) error {
	for {
		op, err := svc.GlobalOperations.Get(project, opName).Do()
		if err != nil {
			return err
		}
		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				return fmt.Errorf("operation failed: %s", op.Error.Errors[0].Message)
			}
			return nil
		}
	}
}

// GCPVPCRepository is the GCP implementation of CloudVPCRepository.
// In production it uses a GCPClientRegistry to route calls per project alias;
// in tests a single injected client and project ID are used instead.
type GCPVPCRepository struct {
	registry    *GCPClientRegistry
	testClient  gcpVPCClient // non-nil only in test mode
	testProject string       // used together with testClient
}

// NewGCPVPCRepositoryFromRegistry creates a production repository backed by the given registry.
func NewGCPVPCRepositoryFromRegistry(registry *GCPClientRegistry) *GCPVPCRepository {
	return &GCPVPCRepository{registry: registry}
}

// newGCPVPCRepositoryWithClient creates a repository with an injected client (for tests).
func newGCPVPCRepositoryWithClient(client gcpVPCClient, project string) *GCPVPCRepository {
	return &GCPVPCRepository{testClient: client, testProject: project}
}

// clientFor returns the GCP client and actual project ID for the given alias.
// In test mode the injected client and project are always returned.
func (r *GCPVPCRepository) clientFor(alias string) (gcpVPCClient, string, error) {
	if r.testClient != nil {
		return r.testClient, r.testProject, nil
	}
	return r.registry.vpcClient(alias)
}

// GetVPC retrieves a VPC (Network) by name from GCP.
func (r *GCPVPCRepository) GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	net, err := client.GetNetwork(project, vpcID)
	if err != nil {
		return nil, fmt.Errorf("get network: %w", err)
	}
	v := mapGCPNetwork(net)

	subnets, err := r.ListSubnets(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.Subnets = subnets

	routes, err := r.listRoutes(client, project, net.SelfLink)
	if err != nil {
		return nil, err
	}
	v.Routes = routes

	v.Peerings = mapGCPPeerings(net.Peerings)

	vpns, err := r.ListVPNs(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.VPNs = vpns

	return v, nil
}

// ListVPCs lists all networks in the GCP project.
func (r *GCPVPCRepository) ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	nets, err := client.ListNetworks(project)
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	result := make([]vpc.VPC, 0, len(nets))
	for _, n := range nets {
		result = append(result, *mapGCPNetwork(n))
	}
	return result, nil
}

// UpdateRoutes replaces all custom routes for a GCP network.
func (r *GCPVPCRepository) UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error {
	client, project, err := r.clientFor(account)
	if err != nil {
		return err
	}
	net, err := client.GetNetwork(project, vpcID)
	if err != nil {
		return fmt.Errorf("get network: %w", err)
	}
	existing, err := client.ListRoutes(project, net.SelfLink)
	if err != nil {
		return fmt.Errorf("list routes: %w", err)
	}
	for _, rt := range existing {
		if rt.NextHopGateway == "" {
			if err = client.DeleteRoute(project, rt.Name); err != nil {
				return fmt.Errorf("delete route %q: %w", rt.Name, err)
			}
		}
	}
	for i, route := range routes {
		gcpRoute := &compute.Route{
			Name:           fmt.Sprintf("%s-route-%d", vpcID, i),
			Network:        net.SelfLink,
			DestRange:      route.Destination,
			NextHopGateway: route.NextHop,
			Priority:       int64(route.Priority),
			Description:    route.Description,
		}
		if err = client.InsertRoute(project, gcpRoute); err != nil {
			return fmt.Errorf("insert route %q: %w", route.Destination, err)
		}
	}
	return nil
}

// ListSubnets lists all subnetworks for a network in a given region.
func (r *GCPVPCRepository) ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	net, err := client.GetNetwork(project, vpcID)
	if err != nil {
		return nil, fmt.Errorf("get network: %w", err)
	}
	subs, err := client.ListSubnetworks(project, region, net.SelfLink)
	if err != nil {
		return nil, fmt.Errorf("list subnetworks: %w", err)
	}
	result := make([]vpc.Subnet, 0, len(subs))
	for _, s := range subs {
		result = append(result, vpc.Subnet{
			ID:        s.Name,
			Name:      s.Name,
			CIDRBlock: s.IpCidrRange,
			Zone:      regionFromURL(s.Region),
			Provider:  vpc.ProviderGCP,
		})
	}
	return result, nil
}

// ListPeerings lists all VPC peering connections for a network in GCP.
func (r *GCPVPCRepository) ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	net, err := client.GetNetwork(project, vpcID)
	if err != nil {
		return nil, fmt.Errorf("get network: %w", err)
	}
	return mapGCPPeerings(net.Peerings), nil
}

// ListVPNs lists all VPN tunnels for a network in a region.
func (r *GCPVPCRepository) ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error) {
	client, project, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	tunnels, err := client.ListVpnTunnels(project, region)
	if err != nil {
		return nil, fmt.Errorf("list vpn tunnels: %w", err)
	}
	result := make([]vpc.VPN, 0, len(tunnels))
	for _, t := range tunnels {
		localTraffic := ""
		if len(t.LocalTrafficSelector) > 0 {
			localTraffic = t.LocalTrafficSelector[0]
		}
		result = append(result, vpc.VPN{
			ID:            t.Name,
			Name:          t.Name,
			LocalVPC:      vpcID,
			RemoteGateway: t.PeerIp,
			State:         t.Status,
			Provider:      vpc.ProviderGCP,
			Tunnels: []vpc.VPNTunnel{
				{
					ID:       t.Name,
					LocalIP:  localTraffic,
					RemoteIP: t.PeerIp,
					State:    t.Status,
				},
			},
		})
	}
	return result, nil
}

// listRoutes retrieves all custom routes for a network identified by its self-link.
func (r *GCPVPCRepository) listRoutes(client gcpVPCClient, project, networkSelfLink string) ([]vpc.Route, error) {
	routes, err := client.ListRoutes(project, networkSelfLink)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	result := make([]vpc.Route, 0, len(routes))
	for _, rt := range routes {
		nextHop := rt.NextHopGateway
		if nextHop == "" {
			nextHop = rt.NextHopInstance
		}
		if nextHop == "" {
			nextHop = rt.NextHopVpnTunnel
		}
		result = append(result, vpc.Route{
			Destination: rt.DestRange,
			NextHop:     nextHop,
			Priority:    int(rt.Priority),
			Description: rt.Description,
		})
	}
	return result, nil
}

// mapGCPNetwork converts a GCP Network to the domain VPC model.
func mapGCPNetwork(n *compute.Network) *vpc.VPC {
	return &vpc.VPC{
		ID:       n.Name,
		Name:     n.Name,
		Provider: vpc.ProviderGCP,
	}
}

// mapGCPPeerings converts GCP network peerings to domain Peering slice.
func mapGCPPeerings(peerings []*compute.NetworkPeering) []vpc.Peering {
	result := make([]vpc.Peering, 0, len(peerings))
	for _, p := range peerings {
		result = append(result, vpc.Peering{
			ID:       p.Name,
			Name:     p.Name,
			PeerVPC:  networkNameFromURL(p.Network),
			State:    p.State,
			Provider: vpc.ProviderGCP,
		})
	}
	return result
}

// regionFromURL extracts the region name from a GCP self-link URL.
func regionFromURL(url string) string {
	parts := strings.Split(url, "/")
	for i, p := range parts {
		if p == "regions" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return url
}

// networkNameFromURL extracts the network name from a GCP self-link URL.
func networkNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}
