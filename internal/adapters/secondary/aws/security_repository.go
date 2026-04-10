package aws

import (
	"context"
	"fmt"

	awslib "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"gournetwork/internal/domain/security"
)

// ec2SecurityClient defines the EC2 operations used by AWSSecurityRepository.
type ec2SecurityClient interface {
	DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error)
	AuthorizeSecurityGroupIngress(ctx context.Context, params *ec2.AuthorizeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupIngressOutput, error)
	AuthorizeSecurityGroupEgress(ctx context.Context, params *ec2.AuthorizeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.AuthorizeSecurityGroupEgressOutput, error)
	RevokeSecurityGroupIngress(ctx context.Context, params *ec2.RevokeSecurityGroupIngressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupIngressOutput, error)
	RevokeSecurityGroupEgress(ctx context.Context, params *ec2.RevokeSecurityGroupEgressInput, optFns ...func(*ec2.Options)) (*ec2.RevokeSecurityGroupEgressOutput, error)
}

// AWSSecurityRepository is the AWS implementation of CloudSecurityRepository.
// In production it uses an AWSClientRegistry to route calls per account;
// in tests a single injected client is used instead.
type AWSSecurityRepository struct {
	registry   *AWSClientRegistry
	testClient ec2SecurityClient // non-nil only in test mode
}

// NewAWSSecurityRepositoryFromRegistry creates a production repository backed by the given registry.
func NewAWSSecurityRepositoryFromRegistry(registry *AWSClientRegistry) *AWSSecurityRepository {
	return &AWSSecurityRepository{registry: registry}
}

// newAWSSecurityRepositoryWithClient creates a repository with an injected client (for tests).
func newAWSSecurityRepositoryWithClient(client ec2SecurityClient) *AWSSecurityRepository {
	return &AWSSecurityRepository{testClient: client}
}

// clientFor returns the EC2 client for the given account.
// In test mode the injected client is always returned.
func (r *AWSSecurityRepository) clientFor(account string) (ec2SecurityClient, error) {
	if r.testClient != nil {
		return r.testClient, nil
	}
	return r.registry.securityClient(account)
}

// GetSecurityGroup retrieves a security group by ID from AWS.
func (r *AWSSecurityRepository) GetSecurityGroup(ctx context.Context, provider, account, region, groupID string) (*security.SecurityGroup, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe security groups: %w", err)
	}
	if len(out.SecurityGroups) == 0 {
		return nil, fmt.Errorf("security group %q not found", groupID)
	}
	return mapAWSSecurityGroup(out.SecurityGroups[0]), nil
}

// ListSecurityGroups lists all security groups for a VPC in AWS.
func (r *AWSSecurityRepository) ListSecurityGroups(ctx context.Context, provider, account, region, vpcID string) ([]security.SecurityGroup, error) {
	client, err := r.clientFor(account)
	if err != nil {
		return nil, err
	}
	out, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{Name: awslib.String("vpc-id"), Values: []string{vpcID}},
		},
	}, regionOpt(region))
	if err != nil {
		return nil, fmt.Errorf("describe security groups: %w", err)
	}
	result := make([]security.SecurityGroup, 0, len(out.SecurityGroups))
	for _, sg := range out.SecurityGroups {
		result = append(result, *mapAWSSecurityGroup(sg))
	}
	return result, nil
}

// UpdateRule authorizes a new security rule within a group in AWS.
func (r *AWSSecurityRepository) UpdateRule(ctx context.Context, provider, account, region, groupID string, rule security.SecurityRule) error {
	client, err := r.clientFor(account)
	if err != nil {
		return err
	}
	perm := toIPPermission(rule)
	if rule.Direction == "ingress" {
		_, err := client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       awslib.String(groupID),
			IpPermissions: []types.IpPermission{perm},
		}, regionOpt(region))
		if err != nil {
			return fmt.Errorf("authorize ingress: %w", err)
		}
		return nil
	}
	_, err = client.AuthorizeSecurityGroupEgress(ctx, &ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       awslib.String(groupID),
		IpPermissions: []types.IpPermission{perm},
	}, regionOpt(region))
	if err != nil {
		return fmt.Errorf("authorize egress: %w", err)
	}
	return nil
}

// DeleteRule removes a security rule from a group in AWS.
func (r *AWSSecurityRepository) DeleteRule(ctx context.Context, provider, account, region, groupID, ruleID string) error {
	sg, err := r.GetSecurityGroup(ctx, provider, account, region, groupID)
	if err != nil {
		return err
	}
	client, err := r.clientFor(account)
	if err != nil {
		return err
	}
	for _, rule := range sg.Rules {
		if rule.ID != ruleID {
			continue
		}
		perm := toIPPermission(rule)
		if rule.Direction == "ingress" {
			if _, err = client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
				GroupId:       awslib.String(groupID),
				IpPermissions: []types.IpPermission{perm},
			}, regionOpt(region)); err != nil {
				return fmt.Errorf("revoke ingress rule %q: %w", ruleID, err)
			}
			return nil
		}
		if _, err = client.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       awslib.String(groupID),
			IpPermissions: []types.IpPermission{perm},
		}, regionOpt(region)); err != nil {
			return fmt.Errorf("revoke egress rule %q: %w", ruleID, err)
		}
		return nil
	}
	return fmt.Errorf("rule %q not found in security group %q", ruleID, groupID)
}

// mapAWSSecurityGroup converts an EC2 SecurityGroup to the domain model.
func mapAWSSecurityGroup(sg types.SecurityGroup) *security.SecurityGroup {
	var rules []security.SecurityRule
	for _, p := range sg.IpPermissions {
		rules = append(rules, mapIPPermission(p, "ingress"))
	}
	for _, p := range sg.IpPermissionsEgress {
		rules = append(rules, mapIPPermission(p, "egress"))
	}
	return &security.SecurityGroup{
		ID:       awslib.ToString(sg.GroupId),
		Name:     awslib.ToString(sg.GroupName),
		Provider: "aws",
		VPCID:    awslib.ToString(sg.VpcId),
		Rules:    rules,
	}
}

// mapIPPermission converts an EC2 IpPermission to the domain SecurityRule.
func mapIPPermission(p types.IpPermission, direction string) security.SecurityRule {
	from, to := 0, 0
	if p.FromPort != nil {
		from = int(awslib.ToInt32(p.FromPort))
	}
	if p.ToPort != nil {
		to = int(awslib.ToInt32(p.ToPort))
	}
	var sources []string
	for _, r := range p.IpRanges {
		sources = append(sources, awslib.ToString(r.CidrIp))
	}
	return security.SecurityRule{
		Direction: direction,
		Protocol:  awslib.ToString(p.IpProtocol),
		PortRange: security.PortRange{From: from, To: to},
		Sources:   sources,
		Action:    "allow",
	}
}

// toIPPermission converts a domain SecurityRule to an EC2 IpPermission.
func toIPPermission(rule security.SecurityRule) types.IpPermission {
	fromPort := int32(rule.PortRange.From)
	toPort := int32(rule.PortRange.To)
	perm := types.IpPermission{
		IpProtocol: awslib.String(rule.Protocol),
		FromPort:   &fromPort,
		ToPort:     &toPort,
	}
	for _, src := range rule.Sources {
		perm.IpRanges = append(perm.IpRanges, types.IpRange{CidrIp: awslib.String(src)})
	}
	return perm
}
