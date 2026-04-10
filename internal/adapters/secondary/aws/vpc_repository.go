package aws

import (
	"context"
	"fmt"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"gournetwork/internal/domain/vpc"
)

// ec2VPCClient defines the EC2 operations used by AWSVPCRepository.
type ec2VPCClient interface {
	DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error)
	DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error)
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error)
	DescribeVpnConnections(ctx context.Context, params *ec2.DescribeVpnConnectionsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpnConnectionsOutput, error)
	CreateRoute(ctx context.Context, params *ec2.CreateRouteInput, optFns ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error)
	DeleteRoute(ctx context.Context, params *ec2.DeleteRouteInput, optFns ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error)
}

// AWSVPCRepository is the AWS implementation of CloudVPCRepository.
// In production it uses an AWSClientRegistry to route calls per account;
// in tests a single injected client is used instead.
type AWSVPCRepository struct {
	registry   *AWSClientRegistry
	testClient ec2VPCClient // non-nil only in test mode
}

// NewAWSVPCRepositoryFromRegistry creates a production repository backed by the given registry.
func NewAWSVPCRepositoryFromRegistry(registry *AWSClientRegistry) *AWSVPCRepository {
	return &AWSVPCRepository{registry: registry}
}

// newAWSVPCRepositoryWithClient creates a repository with an injected client (for tests).
func newAWSVPCRepositoryWithClient(client ec2VPCClient) *AWSVPCRepository {
	return &AWSVPCRepository{testClient: client}
}

// clientFor returns the EC2 client for the given account.
// In test mode the injected client is always returned.
func (r *AWSVPCRepository) clientFor(account string) (ec2VPCClient, error) {
	if r.testClient != nil {
		return r.testClient, nil
	}
	return r.registry.vpcClient(account)
}

// regionOpt returns a functional option that overrides the EC2 region for a single call.
func regionOpt(region string) func(*ec2.Options) {
	return func(o *ec2.Options) { o.Region = region }
}

// GetVPC retrieves a VPC by ID from AWS.
func (r *AWSVPCRepository) GetVPC(ctx context.Context, provider, account, region, vpcID string) (*vpc.VPC, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe vpcs: %w", err)
	}
	if len(out.Vpcs) == 0 {
		return nil, fmt.Errorf("vpc %q not found", vpcID)
	}
	v := mapAWSVPC(out.Vpcs[0])

	subnets, err := r.ListSubnets(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.Subnets = subnets

	routes, err := r.listRoutes(ctx, client, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.Routes = routes

	peerings, err := r.ListPeerings(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.Peerings = peerings

	vpns, err := r.ListVPNs(ctx, provider, account, region, vpcID)
	if err != nil {
		return nil, err
	}
	v.VPNs = vpns

	return v, nil
}

// ListVPCs lists all VPCs in a region from AWS.
func (r *AWSVPCRepository) ListVPCs(ctx context.Context, provider, account, region string) ([]vpc.VPC, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe vpcs: %w", err)
	}
	result := make([]vpc.VPC, 0, len(out.Vpcs))
	for _, v := range out.Vpcs {
		result = append(result, *mapAWSVPC(v))
	}
	return result, nil
}

// UpdateRoutes replaces the custom routes in the main route table of a VPC.
func (r *AWSVPCRepository) UpdateRoutes(ctx context.Context, provider, account, region, vpcID string, routes []vpc.Route) error {
	client, err := r.clientFor(account)
	if err != nil {
		return err
	}
	rtOut, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{Name: awslib.String("vpc-id"), Values: []string{vpcID}},
			{Name: awslib.String("association.main"), Values: []string{"true"}},
		},
	}, regionOpt(region))
	if err != nil {
		return fmt.Errorf("describe route tables: %w", err)
	}
	if len(rtOut.RouteTables) == 0 {
		return fmt.Errorf("main route table not found for vpc %q", vpcID)
	}
	rtID := awslib.ToString(rtOut.RouteTables[0].RouteTableId)

	// Delete all non-system (custom) routes.
	for _, rt := range rtOut.RouteTables[0].Routes {
		if rt.GatewayId != nil && awslib.ToString(rt.GatewayId) == "local" {
			continue
		}
		if rt.DestinationCidrBlock == nil {
			continue
		}
		if _, err = client.DeleteRoute(ctx, &ec2.DeleteRouteInput{
			RouteTableId:         awslib.String(rtID),
			DestinationCidrBlock: rt.DestinationCidrBlock,
		}, regionOpt(region)); err != nil {
			return fmt.Errorf("delete route %q: %w", awslib.ToString(rt.DestinationCidrBlock), err)
		}
	}

	// Create the desired routes.
	for _, route := range routes {
		if _, err = client.CreateRoute(ctx, &ec2.CreateRouteInput{
			RouteTableId:         awslib.String(rtID),
			DestinationCidrBlock: awslib.String(route.Destination),
			GatewayId:            awslib.String(route.NextHop),
		}, regionOpt(region)); err != nil {
			return fmt.Errorf("create route %q: %w", route.Destination, err)
		}
	}
	return nil
}

// ListSubnets lists all subnets for a VPC in AWS.
func (r *AWSVPCRepository) ListSubnets(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Subnet, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{Name: awslib.String("vpc-id"), Values: []string{vpcID}},
		},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe subnets: %w", err)
	}
	result := make([]vpc.Subnet, 0, len(out.Subnets))
	for _, s := range out.Subnets {
		result = append(result, vpc.Subnet{
			ID:        awslib.ToString(s.SubnetId),
			Name:      tagName(s.Tags),
			CIDRBlock: awslib.ToString(s.CidrBlock),
			Zone:      awslib.ToString(s.AvailabilityZone),
			Provider:  vpc.ProviderAWS,
		})
	}
	return result, nil
}

// ListPeerings lists all VPC peering connections for a VPC in AWS.
func (r *AWSVPCRepository) ListPeerings(ctx context.Context, provider, account, region, vpcID string) ([]vpc.Peering, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeVpcPeeringConnections(ctx, &ec2.DescribeVpcPeeringConnectionsInput{
		Filters: []types.Filter{
			{Name: awslib.String("requester-vpc-info.vpc-id"), Values: []string{vpcID}},
		},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe vpc peering connections: %w", err)
	}
	result := make([]vpc.Peering, 0, len(out.VpcPeeringConnections))
	for _, p := range out.VpcPeeringConnections {
		state := ""
		if p.Status != nil {
			state = string(p.Status.Code)
		}
		localVPC, peerVPC := "", ""
		if p.RequesterVpcInfo != nil {
			localVPC = awslib.ToString(p.RequesterVpcInfo.VpcId)
		}
		if p.AccepterVpcInfo != nil {
			peerVPC = awslib.ToString(p.AccepterVpcInfo.VpcId)
		}
		result = append(result, vpc.Peering{
			ID:       awslib.ToString(p.VpcPeeringConnectionId),
			Name:     tagName(p.Tags),
			LocalVPC: localVPC,
			PeerVPC:  peerVPC,
			State:    state,
			Provider: vpc.ProviderAWS,
		})
	}
	return result, nil
}

// ListVPNs lists all VPN connections for a VPC in AWS.
func (r *AWSVPCRepository) ListVPNs(ctx context.Context, provider, account, region, vpcID string) ([]vpc.VPN, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeVpnConnections(ctx, &ec2.DescribeVpnConnectionsInput{
		Filters: []types.Filter{
			{Name: awslib.String("vpc-id"), Values: []string{vpcID}},
		},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe vpn connections: %w", err)
	}
	result := make([]vpc.VPN, 0, len(out.VpnConnections))
	for _, v := range out.VpnConnections {
		tunnels := make([]vpc.VPNTunnel, 0, len(v.VgwTelemetry))
		for _, t := range v.VgwTelemetry {
			tunnels = append(tunnels, vpc.VPNTunnel{
				ID:       awslib.ToString(t.OutsideIpAddress),
				RemoteIP: awslib.ToString(t.OutsideIpAddress),
				State:    string(t.Status),
			})
		}
		result = append(result, vpc.VPN{
			ID:            awslib.ToString(v.VpnConnectionId),
			Name:          tagName(v.Tags),
			LocalVPC:      vpcID,
			RemoteGateway: awslib.ToString(v.VpnGatewayId),
			Tunnels:       tunnels,
			State:         string(v.State),
			Provider:      vpc.ProviderAWS,
		})
	}
	return result, nil
}

// listRoutes retrieves routes from the main route table of a VPC.
func (r *AWSVPCRepository) listRoutes(ctx context.Context, client ec2VPCClient, region, vpcID string) ([]vpc.Route, error) {
	out, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{Name: awslib.String("vpc-id"), Values: []string{vpcID}},
			{Name: awslib.String("association.main"), Values: []string{"true"}},
		},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe route tables: %w", err)
	}
	if len(out.RouteTables) == 0 {
		return nil, nil
	}
	var routes []vpc.Route
	for _, rt := range out.RouteTables[0].Routes {
		nextHop := ""
		if rt.GatewayId != nil {
			nextHop = awslib.ToString(rt.GatewayId)
		} else if rt.NatGatewayId != nil {
			nextHop = awslib.ToString(rt.NatGatewayId)
		} else if rt.TransitGatewayId != nil {
			nextHop = awslib.ToString(rt.TransitGatewayId)
		} else if rt.VpcPeeringConnectionId != nil {
			nextHop = awslib.ToString(rt.VpcPeeringConnectionId)
		}
		if rt.DestinationCidrBlock != nil {
			routes = append(routes, vpc.Route{
				Destination: awslib.ToString(rt.DestinationCidrBlock),
				NextHop:     nextHop,
			})
		}
	}
	return routes, nil
}

// mapAWSVPC converts an EC2 VPC type to the domain VPC model.
func mapAWSVPC(v types.Vpc) *vpc.VPC {
	return &vpc.VPC{
		ID:        awslib.ToString(v.VpcId),
		Name:      tagName(v.Tags),
		CIDRBlock: awslib.ToString(v.CidrBlock),
		Provider:  vpc.ProviderAWS,
	}
}

// tagName extracts the "Name" tag value from a slice of EC2 tags.
func tagName(tags []types.Tag) string {
	for _, t := range tags {
		if awslib.ToString(t.Key) == "Name" {
			return awslib.ToString(t.Value)
		}
	}
	return ""
}
