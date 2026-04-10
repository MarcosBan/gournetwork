package aws

import (
	"context"
	"errors"
	"testing"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"gournetwork/internal/domain/security"
)

// --- mock ---

type mockEC2SecurityClient struct {
	describeSecurityGroupsOut    *ec2.DescribeSecurityGroupsOutput
	describeSecurityGroupsErr    error
	authorizeIngressErr          error
	authorizeEgressErr           error
	revokeIngressErr             error
	revokeEgressErr              error
}

func (m *mockEC2SecurityClient) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return m.describeSecurityGroupsOut, m.describeSecurityGroupsErr
}
func (m *mockEC2SecurityClient) AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, _ ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error) {
	return nil, m.authorizeIngressErr
}
func (m *mockEC2SecurityClient) AuthorizeSecurityGroupEgress(ctx context.Context, params *ec2.AuthorizeSecurityGroupEgressInput, _ ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupEgressOutput, error) {
	return nil, m.authorizeEgressErr
}
func (m *mockEC2SecurityClient) RevokeSecurityGroupIngress(ctx context.Context, params *ec2.RevokeSecurityGroupIngressInput, _ ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error) {
	return nil, m.revokeIngressErr
}
func (m *mockEC2SecurityClient) RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, _ ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error) {
	return nil, m.revokeEgressErr
}

// --- auth tests ---

func TestNewAWSSecurityRepository_MissingCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := newAWSSecurityRepositoryWithOptions(ctx,
		config.WithCredentialsProvider(aws_errorCredentials{}),
	)
	if err == nil {
		t.Fatal("expected error when no credentials available, got nil")
	}
}

func TestNewAWSSecurityRepository_StaticCredentials(t *testing.T) {
	ctx := context.Background()
	repo, err := NewAWSSecurityRepositoryWithStaticCredentials(ctx, "AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "")
	if err != nil {
		t.Fatalf("expected no error with static credentials, got: %v", err)
	}
	if repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestNewAWSSecurityRepository_EmptyCredentials(t *testing.T) {
	ctx := context.Background()
	_, err := NewAWSSecurityRepositoryWithStaticCredentials(ctx, "", "", "")
	if err == nil {
		t.Fatal("expected error with empty credentials, got nil")
	}
}

func TestAWSSecurityRepository_GetSecurityGroup_AuthError(t *testing.T) {
	authErr := errors.New("AuthFailure: AWS was not able to validate the provided access credentials")
	repo := newAWSSecurityRepositoryWithClient(&mockEC2SecurityClient{
		describeSecurityGroupsErr: authErr,
	})
	_, err := repo.GetSecurityGroup(context.Background(), "aws", "us-east-1", "sg-123")
	if err == nil {
		t.Fatal("expected auth error to be propagated, got nil")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- GetSecurityGroup ---

func TestAWSSecurityRepository_GetSecurityGroup(t *testing.T) {
	fromPort := int32(80)
	toPort := int32(80)
	mock := &mockEC2SecurityClient{
		describeSecurityGroupsOut: &ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{
					GroupId:   awslib.String("sg-abc123"),
					GroupName: awslib.String("web-sg"),
					VpcId:     awslib.String("vpc-abc"),
					IpPermissions: []types.IpPermission{
						{
							IpProtocol: awslib.String("tcp"),
							FromPort:   &fromPort,
							ToPort:     &toPort,
							IpRanges:   []types.IpRange{{CidrIp: awslib.String("0.0.0.0/0")}},
						},
					},
				},
			},
		},
	}
	repo := newAWSSecurityRepositoryWithClient(mock)
	sg, err := repo.GetSecurityGroup(context.Background(), "aws", "us-east-1", "sg-abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sg.ID != "sg-abc123" {
		t.Errorf("expected ID sg-abc123, got %q", sg.ID)
	}
	if sg.Name != "web-sg" {
		t.Errorf("expected Name web-sg, got %q", sg.Name)
	}
	if len(sg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(sg.Rules))
	}
	if sg.Rules[0].Direction != "ingress" {
		t.Errorf("expected direction ingress, got %q", sg.Rules[0].Direction)
	}
	if sg.Rules[0].PortRange.From != 80 {
		t.Errorf("expected from port 80, got %d", sg.Rules[0].PortRange.From)
	}
}

func TestAWSSecurityRepository_GetSecurityGroup_NotFound(t *testing.T) {
	mock := &mockEC2SecurityClient{
		describeSecurityGroupsOut: &ec2.DescribeSecurityGroupsOutput{},
	}
	repo := newAWSSecurityRepositoryWithClient(mock)
	_, err := repo.GetSecurityGroup(context.Background(), "aws", "us-east-1", "sg-missing")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

// --- ListSecurityGroups ---

func TestAWSSecurityRepository_ListSecurityGroups(t *testing.T) {
	mock := &mockEC2SecurityClient{
		describeSecurityGroupsOut: &ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{GroupId: awslib.String("sg-1"), GroupName: awslib.String("sg-one")},
				{GroupId: awslib.String("sg-2"), GroupName: awslib.String("sg-two")},
			},
		},
	}
	repo := newAWSSecurityRepositoryWithClient(mock)
	groups, err := repo.ListSecurityGroups(context.Background(), "aws", "us-east-1", "vpc-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 security groups, got %d", len(groups))
	}
}

func TestAWSSecurityRepository_ListSecurityGroups_AuthError(t *testing.T) {
	authErr := errors.New("UnauthorizedOperation")
	repo := newAWSSecurityRepositoryWithClient(&mockEC2SecurityClient{describeSecurityGroupsErr: authErr})
	_, err := repo.ListSecurityGroups(context.Background(), "aws", "us-east-1", "vpc-abc")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// --- UpdateRule ---

func TestAWSSecurityRepository_UpdateRule_Ingress(t *testing.T) {
	mock := &mockEC2SecurityClient{}
	repo := newAWSSecurityRepositoryWithClient(mock)
	rule := security.SecurityRule{
		Direction: "ingress",
		Protocol:  "tcp",
		PortRange: security.PortRange{From: 443, To: 443},
		Sources:   []string{"10.0.0.0/8"},
		Action:    "allow",
	}
	if err := repo.UpdateRule(context.Background(), "aws", "us-east-1", "sg-abc", rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAWSSecurityRepository_UpdateRule_Egress(t *testing.T) {
	mock := &mockEC2SecurityClient{}
	repo := newAWSSecurityRepositoryWithClient(mock)
	rule := security.SecurityRule{
		Direction: "egress",
		Protocol:  "-1",
		PortRange: security.PortRange{From: 0, To: 0},
		Sources:   []string{"0.0.0.0/0"},
		Action:    "allow",
	}
	if err := repo.UpdateRule(context.Background(), "aws", "us-east-1", "sg-abc", rule); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAWSSecurityRepository_UpdateRule_AuthError(t *testing.T) {
	authErr := errors.New("AuthFailure")
	mock := &mockEC2SecurityClient{authorizeIngressErr: authErr}
	repo := newAWSSecurityRepositoryWithClient(mock)
	rule := security.SecurityRule{Direction: "ingress", Protocol: "tcp"}
	err := repo.UpdateRule(context.Background(), "aws", "us-east-1", "sg-abc", rule)
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
	if !errors.Is(err, authErr) {
		t.Fatalf("expected original auth error in chain, got: %v", err)
	}
}

// --- DeleteRule ---

func TestAWSSecurityRepository_DeleteRule_NotFound(t *testing.T) {
	mock := &mockEC2SecurityClient{
		describeSecurityGroupsOut: &ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{GroupId: awslib.String("sg-abc"), GroupName: awslib.String("test")},
			},
		},
	}
	repo := newAWSSecurityRepositoryWithClient(mock)
	err := repo.DeleteRule(context.Background(), "aws", "us-east-1", "sg-abc", "rule-missing")
	if err == nil {
		t.Fatal("expected not-found error for missing rule")
	}
}

func TestAWSSecurityRepository_DeleteRule_AuthError(t *testing.T) {
	authErr := errors.New("AuthFailure")
	repo := newAWSSecurityRepositoryWithClient(&mockEC2SecurityClient{describeSecurityGroupsErr: authErr})
	err := repo.DeleteRule(context.Background(), "aws", "us-east-1", "sg-abc", "rule-1")
	if err == nil {
		t.Fatal("expected auth error to be propagated")
	}
}

// Compile-time check: mock implements the interface.
var _ ec2SecurityClient = (*mockEC2SecurityClient)(nil)

// Suppress unused import.
var _ = config.LoadDefaultConfig
