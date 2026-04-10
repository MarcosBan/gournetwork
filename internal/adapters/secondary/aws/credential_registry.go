package aws

import (
	"context"
	"fmt"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appconfig "gournetwork/internal/config"
)

// AWSClientRegistry manages one EC2 client per configured AWS account.
// A single *ec2.Client per account is sufficient because the region is
// overridden per-call via regionOpt.
type AWSClientRegistry struct {
	clients map[string]*ec2.Client
}

// NewAWSClientRegistry creates an EC2 client for every account in cfg,
// validates credentials at construction time, and returns an error if any
// account fails. Call this once at startup before serving requests.
func NewAWSClientRegistry(ctx context.Context, cfg appconfig.AWSConfig) (*AWSClientRegistry, error) {
	r := &AWSClientRegistry{clients: make(map[string]*ec2.Client)}
	for _, acct := range cfg.Accounts {
		client, err := buildEC2Client(ctx, acct)
		if err != nil {
			return nil, fmt.Errorf("aws account %q: %w", acct.Name, err)
		}
		r.clients[acct.Name] = client
	}
	return r, nil
}

// vpcClient returns the EC2 client for the named account as an ec2VPCClient.
func (r *AWSClientRegistry) vpcClient(account string) (ec2VPCClient, error) {
	c, ok := r.clients[account]
	if !ok {
		return nil, fmt.Errorf("aws account %q is not configured", account)
	}
	return c, nil
}

// securityClient returns the EC2 client for the named account as an ec2SecurityClient.
func (r *AWSClientRegistry) securityClient(account string) (ec2SecurityClient, error) {
	c, ok := r.clients[account]
	if !ok {
		return nil, fmt.Errorf("aws account %q is not configured", account)
	}
	return c, nil
}

// buildEC2Client constructs and validates a single EC2 client for the given account.
func buildEC2Client(ctx context.Context, acct appconfig.AWSAccount) (*ec2.Client, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if acct.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(acct.AccessKeyID, acct.SecretAccessKey, acct.SessionToken),
		))
	}
	if acct.Region != "" {
		opts = append(opts, awsconfig.WithRegion(acct.Region))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Validate credentials eagerly so failures surface at startup.
	if _, err = cfg.Credentials.Retrieve(ctx); err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}

	return ec2.NewFromConfig(cfg), nil
}
