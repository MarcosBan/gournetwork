package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// AWSAccount holds credentials and settings for a single AWS account.
type AWSAccount struct {
	// Name is the logical identifier used in API requests (e.g., "production", "staging").
	Name string
	// AccessKeyID is the AWS access key ID.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
	// SessionToken is the optional STS session token (for temporary credentials).
	SessionToken string
	// Region is the optional default AWS region for this account.
	Region string
}

// AWSConfig holds configuration for all configured AWS accounts.
type AWSConfig struct {
	Accounts []AWSAccount
}

// GetAccount returns the account with the given logical name, or nil if not found.
func (c *AWSConfig) GetAccount(name string) *AWSAccount {
	for i := range c.Accounts {
		if c.Accounts[i].Name == name {
			return &c.Accounts[i]
		}
	}
	return nil
}

// loadAWSConfig discovers AWS accounts from:
//  1. The aws.config file (INI format, like ~/.aws/credentials) — lower priority.
//  2. Environment variables — higher priority, overrides file settings.
//
// Supported environment variable patterns for an account named "production"
// (the name is the lowercase middle segment, e.g. AWS_PRODUCTION_... → "production"):
//
//	AWS_PRODUCTION_ACCESS_KEY_ID=AKIA...
//	AWS_PRODUCTION_SECRET_ACCESS_KEY=secret...
//	AWS_PRODUCTION_SESSION_TOKEN=token...   (optional)
//	AWS_PRODUCTION_REGION=us-east-1         (optional)
//
// Alternatively, use the combined credentials form:
//
//	AWS_PRODUCTION_CREDENTIALS=AKIA...:secret...[:token...]
func loadAWSConfig() (AWSConfig, error) {
	accounts := make(map[string]*AWSAccount)

	// 1. Load from aws.config file (lower priority).
	if fileAccounts, err := loadAWSConfigFile("aws.config"); err == nil {
		for i := range fileAccounts {
			accounts[fileAccounts[i].Name] = &fileAccounts[i]
		}
	}

	// 2. Scan environment variables for account discovery.
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || !strings.HasPrefix(kv[0], "AWS_") {
			continue
		}
		key, val := kv[0], kv[1]

		// Pattern: AWS_{NAME}_ACCESS_KEY_ID
		if strings.HasSuffix(key, "_ACCESS_KEY_ID") {
			name := awsAccountName(key, "_ACCESS_KEY_ID")
			if name == "" {
				continue
			}
			if accounts[name] == nil {
				accounts[name] = &AWSAccount{Name: name}
			}
			accounts[name].AccessKeyID = val
		}

		// Pattern: AWS_{NAME}_CREDENTIALS=key:secret[:token]
		if strings.HasSuffix(key, "_CREDENTIALS") {
			name := awsAccountName(key, "_CREDENTIALS")
			if name == "" {
				continue
			}
			parts := strings.SplitN(val, ":", 3)
			if len(parts) < 2 {
				return AWSConfig{}, fmt.Errorf(
					"AWS_%s_CREDENTIALS must be formatted as accessKeyId:secretAccessKey[:sessionToken]",
					strings.ToUpper(name),
				)
			}
			if accounts[name] == nil {
				accounts[name] = &AWSAccount{Name: name}
			}
			accounts[name].AccessKeyID = parts[0]
			accounts[name].SecretAccessKey = parts[1]
			if len(parts) == 3 {
				accounts[name].SessionToken = parts[2]
			}
		}
	}

	// 3. Fill in remaining per-account fields from environment variables.
	for name, acct := range accounts {
		prefix := "AWS_" + strings.ToUpper(name) + "_"
		if v := os.Getenv(prefix + "SECRET_ACCESS_KEY"); v != "" {
			acct.SecretAccessKey = v
		}
		if v := os.Getenv(prefix + "SESSION_TOKEN"); v != "" {
			acct.SessionToken = v
		}
		if v := os.Getenv(prefix + "REGION"); v != "" {
			acct.Region = v
		}
	}

	// 4. Validate and collect.
	result := make([]AWSAccount, 0, len(accounts))
	for _, acct := range accounts {
		if acct.AccessKeyID != "" && acct.SecretAccessKey == "" {
			return AWSConfig{}, fmt.Errorf(
				"aws account %q: access key ID is set but secret access key is missing "+
					"(set AWS_%s_SECRET_ACCESS_KEY)",
				acct.Name, strings.ToUpper(acct.Name),
			)
		}
		result = append(result, *acct)
	}

	return AWSConfig{Accounts: result}, nil
}

// awsAccountName extracts the logical account name from an env var key.
// Example: ("AWS_PRODUCTION_ACCESS_KEY_ID", "_ACCESS_KEY_ID") → "production"
func awsAccountName(key, suffix string) string {
	// Strip "AWS_" prefix and the suffix.
	inner := key[len("AWS_") : len(key)-len(suffix)]
	if inner == "" {
		return ""
	}
	return strings.ToLower(inner)
}

// loadAWSConfigFile parses an INI-style credentials file (same format as ~/.aws/credentials).
// Each [section] becomes an account name; supported keys:
//
//	aws_access_key_id, aws_secret_access_key, aws_session_token, region
func loadAWSConfigFile(path string) ([]AWSAccount, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err // not present is fine; caller ignores the error
	}
	defer f.Close()

	var accounts []AWSAccount
	var current *AWSAccount
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				accounts = append(accounts, *current)
			}
			name := strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			current = &AWSAccount{Name: name}
			continue
		}
		if current == nil {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch k {
		case "aws_access_key_id":
			current.AccessKeyID = v
		case "aws_secret_access_key":
			current.SecretAccessKey = v
		case "aws_session_token":
			current.SessionToken = v
		case "region":
			current.Region = v
		}
	}
	if current != nil {
		accounts = append(accounts, *current)
	}
	return accounts, scanner.Err()
}
