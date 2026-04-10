package aws

import (
	"context"
	"errors"
	"testing"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"gournetwork/internal/domain/vpc"
)

// --- mock ---

type mockEC2VPCClient struct {
	describeVpcsOut           *ec2.DescribeVpcsOutput
	describeVpcsErr           error
	describeSubnetsOut        *ec2.DescribeSubnetsOutput
	describeSubnetsErr        error
	describeRouteTablesOut    *ec2.DescribeRouteTablesOutput
	describeRouteTablesErr    error
	describeVpcPeeringsOut    *ec2.DescribeVpcPeeringConnectionsOutput
	describeVpcPeeringsErr    error
	describeVpnConnectionsOut *ec2.DescribeVpnConnectionsOutput
	describeVpnConnectionsErr error
	createRouteErr            error
	deleteRouteErr            error
}

func (m *mockEC2VPCClient) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	return m.describeVpcsOut, m.describeVpcsErr
}
func (m *mockEC2VPCClient) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	return m.describeSubnetsOut, m.describeSubnetsErr
}
func (m *mockEC2VPCClient) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, _ ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	return m.describeRouteTablesOut, m.describeRouteTablesErr
}
func (m *mockEC2VPCClient) DescribeVpcPeeringConnections(ctx context.Context, params *ec2.DescribeVpcPeeringConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpcPeeringConnectionsOutput, error) {
	return m.describeVpcPeeringsOut, m.describeVpcPeeringsErr
}
func (m *mockEC2VPCClient) DescribeVpnConnections(ctx context.Context, params *ec2.DescribeVpnConnectionsInput, _ ...func(*ec2.Options)) (*ec2.DescribeVpnConnectionsOutput, error) {
	return m.describeVpnConnectionsOut, m.describeVpnConnectionsErr
}
func (m *mockEC2VPCClient) CreateRoute(ctx context.Context, params *ec2.CreateRouteInput, _ ...func(*ec2.Options)) (*ec2.CreateRouteOutput, error) {
	return nil, m.createRouteErr
}
func (m *mockEC2VPCClient) DeleteRoute(ctx context.Context, params *ec2.DeleteRouteInput, _ ...func(*ec2.Options)) (*ec2.DeleteRouteOutput, error) {
	return nil, m.deleteRouteErr
}

// --- auth tests ---

// TestNewAWSVPCRepository_MissingCredentials verifies that the constructor returns an error
// when credentials cannot be retrieved from the credential chain.
func TestNewAWSVPCRepository_MissingCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := newAWSVPCRepositoryWithOptions(ctx,
		config.WithCredentialsProvider(aws_errorCredentials{}),
	)
	if err == nil {
		t.Fatal("expected error when no credentials available, got nil")
	}
}

// TestNewAWSVPCRepository_StaticCredentials verifies that the constructor succeeds
// when valid static credentials are provided.
func TestNewAWSVPCRepository_StaticCredentials(t *testing.T) {
	ctx := context.Background()
	repo, err := NewAWSVPCRepositoryWithStaticCredentials(ctx, "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "")
	if err != nil {
		t.Fatalf("expected no error with static credentials, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

// TestNewAWSVPCRepository_EmptyCredentials verifies that empty key/secret returns an error.
func TestNewAWSVPCRepository_EmptyCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := NewAWSVPCRepositoryWithStaticCredentials(ctx, "", "", "")
	if err == nil {
		t.Fatal("expected error with empty credentials, got nil")
	}
}

// TestAWSVPCRepository_GetVPC_AuthError verifies that an authentication failure from the SDK
// is propagated as an error.
func TestAWSVPCRepository_GetVPC_AuthError(t *testing.T) {
	authErr := errors.New("AuthFailure: AWS was not able to validate the provided access credentials")
	repo := newAWSVPCRepositoryWithClient(&mockEC2VPCClient{
		describeVpcsErr: authErr,
	})
	_, err := repo.GetVPC(context.Background(), "aws", "us-east-1", "vpc-123")
	if err == nil {
		t.Fatal("expected auth error to be propagated, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- GetVPC ---

func TestAWSVPCRepository_GetVPC(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeVpcsOut: &ec2.DescribeVpcsOutput{
			Vpcs: []types.Vpc{
				{
					VpcId:     awslib.String("vpc-abc123"),
					CidrBlock: awslib.String("10.0.0.0/16"),
					Tags:      []types.Tag{{Key: awslib.String("Name"), Value: awslib.String("test-vpc")}},
				},
			},
		},
		describeSubnetsOut:     &ec2.DescribeSubnetsOutput{},
		describeRouteTablesOut: &ec2.DescribeRouteTablesOutput{},
		describeVpcPeeringsOut: &ec2.DescribeVpcPeeringConnectionsOutput{},
		describeVpnConnectionsOut: &ec2.DescribeVpnConnectionsOutput{},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	got, err := repo.GetVPC(context.Background(), "aws", "us-east-1", "vpc-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "vpc-abc123" {
		t.Errorf("expected ID vpc-abc123, got %q", got.ID)
	}
	if got.Name != "test-vpc" {
		t.Errorf("expected Name test-vpc, got %q", got.Name)
	}
	if got.CIDRBlock != "10.0.0.0/16" {
		t.Errorf("expected CIDR 10.0.0.0/16, got %q", got.CIDRBlock)
	}
	if got.Provider != vpc.ProviderAWS {
		t.Errorf("expected provider AWS, got %q", got.Provider)
	}
}

func TestAWSVPCRepository_GetVPC_NotFound(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeVpcsOut: &ec2.DescribeVpcsOutput{Vpcs: []types.Vpc{}},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	_, err := repo.GetVPC(context.Background(), "aws", "us-east-1", "vpc-missing")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

// --- ListVPCs ---

func TestAWSVPCRepository_ListVPCs(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeVpcsOut: &ec2.DescribeVpcsOutput{
			Vpcs: []types.Vpc{
				{VpcId: awslib.String("vpc-1"), CidrBlock: awslib.String("10.0.0.0/16")},
				{VpcId: awslib.String("vpc-2"), CidrBlock: awslib.String("10.1.0.0/16")},
			},
		},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	vpcs, err := repo.ListVPCs(context.Background(), "aws", "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpcs) != 2 {
		t.Fatalf("expected 2 VPCs, got %d", len(vpcs))
	}
}

func TestAWSVPCRepository_ListVPCs_AuthError(t *testing.T) {
	authErr := errors.New("UnauthorizedOperation")
	repo := newAWSVPCRepositoryWithClient(&mockEC2VPCClient{describeVpcsErr: authErr})
	_, err := repo.ListVPCs(context.Background(), "aws", "us-east-1")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// --- ListSubnets ---

func TestAWSVPCRepository_ListSubnets(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeSubnetsOut: &ec2.DescribeSubnetsOutput{
			Subnets: []types.Subnet{
				{
					SubnetId:         awslib.String("subnet-1"),
					CidrBlock:        awslib.String("10.0.1.0/24"),
					AvailabilityZone: awslib.String("us-east-1a"),
					Tags:             []types.Tag{{Key: awslib.String("Name"), Value: awslib.String("subnet-public")}},
				},
			},
		},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	subnets, err := repo.ListSubnets(context.Background(), "aws", "us-east-1", "vpc-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(subnets) != 1 {
		t.Fatalf("expected 1 subnet, got %d", len(subnets))
	}
	if subnets[0].ID != "subnet-1" {
		t.Errorf("expected ID subnet-1, got %q", subnets[0].ID)
	}
	if subnets[0].Zone != "us-east-1a" {
		t.Errorf("expected zone us-east-1a, got %q", subnets[0].Zone)
	}
}

// --- ListPeerings ---

func TestAWSVPCRepository_ListPeerings(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeVpcPeeringsOut: &ec2.DescribeVpcPeeringConnectionsOutput{
			VpcPeeringConnections: []types.VpcPeeringConnection{
				{
					VpcPeeringConnectionId: awslib.String("pcx-123"),
					Status:                 &types.VpcPeeringConnectionStateReason{Code: types.VpcPeeringConnectionStateReasonCodeActive},
					RequesterVpcInfo:       &types.VpcPeeringConnectionVpcInfo{VpcId: awslib.String("vpc-abc123")},
					AccepterVpcInfo:        &types.VpcPeeringConnectionVpcInfo{VpcId: awslib.String("vpc-peer")},
				},
			},
		},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	peerings, err := repo.ListPeerings(context.Background(), "aws", "us-east-1", "vpc-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(peerings) != 1 {
		t.Fatalf("expected 1 peering, got %d", len(peerings))
	}
	if peerings[0].ID != "pcx-123" {
		t.Errorf("expected ID pcx-123, got %q", peerings[0].ID)
	}
	if peerings[0].LocalVPC != "vpc-abc123" {
		t.Errorf("expected LocalVPC vpc-abc123, got %q", peerings[0].LocalVPC)
	}
	if peerings[0].PeerVPC != "vpc-peer" {
		t.Errorf("expected PeerVPC vpc-peer, got %q", peerings[0].PeerVPC)
	}
}

// --- ListVPNs ---

func TestAWSVPCRepository_ListVPNs(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeVpnConnectionsOut: &ec2.DescribeVpnConnectionsOutput{
			VpnConnections: []types.VpnConnection{
				{
					VpnConnectionId: awslib.String("vpn-123"),
					VpnGatewayId:    awslib.String("vgw-abc"),
					State:           types.VpnStateAvailable,
					VgwTelemetry: []types.VgwTelemetry{
						{OutsideIpAddress: awslib.String("1.2.3.4"), Status: types.TelemetryStatusUp},
					},
				},
			},
		},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	vpns, err := repo.ListVPNs(context.Background(), "aws", "us-east-1", "vpc-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vpns) != 1 {
		t.Fatalf("expected 1 VPN, got %d", len(vpns))
	}
	if vpns[0].ID != "vpn-123" {
		t.Errorf("expected ID vpn-123, got %q", vpns[0].ID)
	}
	if vpns[0].RemoteGateway != "vgw-abc" {
		t.Errorf("expected RemoteGateway vgw-abc, got %q", vpns[0].RemoteGateway)
	}
	if len(vpns[0].Tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(vpns[0].Tunnels))
	}
}

// --- UpdateRoutes ---

func TestAWSVPCRepository_UpdateRoutes(t *testing.T) {
	mock := &mockEC2VPCClient{
		describeRouteTablesOut: &ec2.DescribeRouteTablesOutput{
			RouteTables: []types.RouteTable{
				{
					RouteTableId: awslib.String("rtb-123"),
					Routes: []types.Route{
						{
							DestinationCidrBlock: awslib.String("0.0.0.0/0"),
							GatewayId:            awslib.String("igw-abc"),
						},
					},
				},
			},
		},
	}
	repo := newAWSVPCRepositoryWithClient(mock)
	routes := []vpc.Route{
		{Destination: "10.2.0.0/16", NextHop: "pcx-456"},
	}
	if err := repo.UpdateRoutes(context.Background(), "aws", "us-east-1", "vpc-abc123", routes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAWSVPCRepository_UpdateRoutes_AuthError(t *testing.T) {
	authErr := errors.New("AuthFailure")
	repo := newAWSVPCRepositoryWithClient(&mockEC2VPCClient{
		describeRouteTablesErr: authErr,
	})
	err := repo.UpdateRoutes(context.Background(), "aws", "us-east-1", "vpc-abc123", nil)
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- helpers ---

// aws_errorCredentials is a credentials provider that always returns an error.
type aws_errorCredentials struct{}

func (aws_errorCredentials) Retrieve(ctx context.Context) (awslib.Credentials, error) {
	return awslib.Credentials{}, errors.New("no credentials configured")
}

// Verify static credentials provider compiles correctly (used in constructor test).
var _ = credentials.NewStaticCredentialsProvider
