package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// --- GlobalSettings / Load ---

func TestLoad_DefaultGlobalSettings(t *testing.T) {
	// Unset all relevant env vars.
	for _, k := range []string{"API_PORT", "API_LOG_LEVEL", "AWS_DEFAULT_ACCOUNT", "GCP_DEFAULT_PROJECT"} {
		t.Setenv(k, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Global.Port != ":8080" {
		t.Errorf("expected default port :8080, got %q", cfg.Global.Port)
	}
	if cfg.Global.LogLevel != "info" {
		t.Errorf("expected default log level info, got %q", cfg.Global.LogLevel)
	}
}

func TestLoad_GlobalSettingsFromEnv(t *testing.T) {
	t.Setenv("API_PORT", ":9090")
	t.Setenv("API_LOG_LEVEL", "debug")
	t.Setenv("AWS_DEFAULT_ACCOUNT", "production")
	t.Setenv("GCP_DEFAULT_PROJECT", "staging")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Global.Port != ":9090" {
		t.Errorf("expected :9090, got %q", cfg.Global.Port)
	}
	if cfg.Global.LogLevel != "debug" {
		t.Errorf("expected debug, got %q", cfg.Global.LogLevel)
	}
	if cfg.Global.DefaultAWSAccount != "production" {
		t.Errorf("expected production, got %q", cfg.Global.DefaultAWSAccount)
	}
	if cfg.Global.DefaultGCPProject != "staging" {
		t.Errorf("expected staging, got %q", cfg.Global.DefaultGCPProject)
	}
}

// --- Validate ---

func TestValidate_NoCredentials(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(context.Background()); err == nil {
		t.Fatal("expected error when no credentials configured, got nil")
	}
}

func TestValidate_AWSOnly(t *testing.T) {
	cfg := &Config{
		AWS: AWSConfig{Accounts: []AWSAccount{{Name: "prod", AccessKeyID: "A", SecretAccessKey: "S"}}},
	}
	if err := cfg.Validate(context.Background()); err != nil {
		t.Fatalf("expected no error with AWS credentials, got: %v", err)
	}
}

func TestValidate_GCPOnly(t *testing.T) {
	cfg := &Config{
		GCP: GCPConfig{Projects: []GCPProject{{Name: "prod", ProjectID: "proj-123"}}},
	}
	if err := cfg.Validate(context.Background()); err != nil {
		t.Fatalf("expected no error with GCP credentials, got: %v", err)
	}
}

// --- AWS env var loading ---

func TestLoadAWSConfig_EnvVarAccessKey(t *testing.T) {
	t.Setenv("AWS_PRODUCTION_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_PRODUCTION_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_PRODUCTION_REGION", "us-east-1")

	cfg, err := loadAWSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	acct := cfg.GetAccount("production")
	if acct == nil {
		t.Fatal("expected account 'production' to be loaded")
	}
	if acct.AccessKeyID != "AKIAIOSFODNN7EXAMPLE" {
		t.Errorf("unexpected AccessKeyID: %q", acct.AccessKeyID)
	}
	if acct.SecretAccessKey != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" {
		t.Errorf("unexpected SecretAccessKey: %q", acct.SecretAccessKey)
	}
	if acct.Region != "us-east-1" {
		t.Errorf("unexpected Region: %q", acct.Region)
	}
}

func TestLoadAWSConfig_CombinedCredentials(t *testing.T) {
	t.Setenv("AWS_STAGING_CREDENTIALS", "AKID:secret:token")

	cfg, err := loadAWSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	acct := cfg.GetAccount("staging")
	if acct == nil {
		t.Fatal("expected account 'staging' to be loaded")
	}
	if acct.AccessKeyID != "AKID" {
		t.Errorf("unexpected AccessKeyID: %q", acct.AccessKeyID)
	}
	if acct.SecretAccessKey != "secret" {
		t.Errorf("unexpected SecretAccessKey: %q", acct.SecretAccessKey)
	}
	if acct.SessionToken != "token" {
		t.Errorf("unexpected SessionToken: %q", acct.SessionToken)
	}
}

func TestLoadAWSConfig_CombinedCredentials_InvalidFormat(t *testing.T) {
	t.Setenv("AWS_BAD_CREDENTIALS", "only-one-part")

	_, err := loadAWSConfig()
	if err == nil {
		t.Fatal("expected error for malformed CREDENTIALS value, got nil")
	}
}

func TestLoadAWSConfig_MissingSecret(t *testing.T) {
	t.Setenv("AWS_PROD_ACCESS_KEY_ID", "AKID")
	// Do not set AWS_PROD_SECRET_ACCESS_KEY.

	_, err := loadAWSConfig()
	if err == nil {
		t.Fatal("expected error when secret key is missing, got nil")
	}
}

func TestLoadAWSConfig_MultipleAccounts(t *testing.T) {
	t.Setenv("AWS_PROD_ACCESS_KEY_ID", "AKID1")
	t.Setenv("AWS_PROD_SECRET_ACCESS_KEY", "secret1")
	t.Setenv("AWS_DEV_ACCESS_KEY_ID", "AKID2")
	t.Setenv("AWS_DEV_SECRET_ACCESS_KEY", "secret2")

	cfg, err := loadAWSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Accounts) < 2 {
		t.Errorf("expected at least 2 accounts, got %d", len(cfg.Accounts))
	}
	if cfg.GetAccount("prod") == nil {
		t.Error("expected 'prod' account")
	}
	if cfg.GetAccount("dev") == nil {
		t.Error("expected 'dev' account")
	}
}

// --- AWS config file loading ---

func TestLoadAWSConfigFile(t *testing.T) {
	dir := t.TempDir()
	content := `
[production]
aws_access_key_id = AKIA_PROD
aws_secret_access_key = secret_prod
region = us-east-1

[staging]
aws_access_key_id = AKIA_STG
aws_secret_access_key = secret_stg
aws_session_token = token_stg
region = us-west-2
`
	path := filepath.Join(dir, "aws.config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	accounts, err := loadAWSConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts, got %d", len(accounts))
	}

	prod := accounts[0]
	if prod.Name != "production" {
		t.Errorf("expected name 'production', got %q", prod.Name)
	}
	if prod.AccessKeyID != "AKIA_PROD" {
		t.Errorf("unexpected AccessKeyID: %q", prod.AccessKeyID)
	}
	if prod.Region != "us-east-1" {
		t.Errorf("unexpected Region: %q", prod.Region)
	}

	stg := accounts[1]
	if stg.SessionToken != "token_stg" {
		t.Errorf("expected session token, got %q", stg.SessionToken)
	}
}

func TestLoadAWSConfigFile_NotFound(t *testing.T) {
	accounts, err := loadAWSConfigFile("/nonexistent/path/aws.config")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if len(accounts) != 0 {
		t.Errorf("expected empty accounts, got %d", len(accounts))
	}
}

// --- GCP env var loading ---

func TestLoadGCPConfig_SingleProject(t *testing.T) {
	t.Setenv("GCP_PROJECT_ID", "my-project-123")
	t.Setenv("GCP_CREDENTIALS_FILE", "/path/to/sa.json")

	cfg, err := loadGCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proj := cfg.GetProject("default")
	if proj == nil {
		t.Fatal("expected 'default' project")
	}
	if proj.ProjectID != "my-project-123" {
		t.Errorf("unexpected ProjectID: %q", proj.ProjectID)
	}
	if proj.CredentialsFile != "/path/to/sa.json" {
		t.Errorf("unexpected CredentialsFile: %q", proj.CredentialsFile)
	}
}

func TestLoadGCPConfig_MultipleProjects(t *testing.T) {
	t.Setenv("GCP_PROD_PROJECT_ID", "prod-123")
	t.Setenv("GCP_PROD_CREDENTIALS_FILE", "/path/prod.json")
	t.Setenv("GCP_STAGING_PROJECT_ID", "staging-456")
	t.Setenv("GCP_STAGING_CREDENTIALS_FILE", "/path/staging.json")

	cfg, err := loadGCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GetProject("prod") == nil {
		t.Error("expected 'prod' project")
	}
	if cfg.GetProject("staging") == nil {
		t.Error("expected 'staging' project")
	}
	if cfg.GetProject("prod").ProjectID != "prod-123" {
		t.Errorf("unexpected prod ProjectID: %q", cfg.GetProject("prod").ProjectID)
	}
}

func TestLoadGCPConfig_ADC(t *testing.T) {
	// GCP_PROJECT_ID without credentials file uses ADC.
	t.Setenv("GCP_PROJECT_ID", "my-project")
	// Make sure credentials vars are unset.
	os.Unsetenv("GCP_CREDENTIALS_FILE")
	os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")

	cfg, err := loadGCPConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proj := cfg.GetProject("default")
	if proj == nil {
		t.Fatal("expected 'default' project")
	}
	if proj.CredentialsFile != "" {
		t.Errorf("expected empty CredentialsFile for ADC, got %q", proj.CredentialsFile)
	}
}

// --- GCP config file loading ---

func TestLoadGCPConfigFile(t *testing.T) {
	dir := t.TempDir()
	content := `
[production]
project_id = prod-project-123
credentials_file = /path/to/prod-sa.json

[staging]
project_id = staging-project-456
credentials_file = /path/to/staging-sa.json
`
	path := filepath.Join(dir, "gcp.config")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	projects, err := loadGCPConfigFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].Name != "production" {
		t.Errorf("expected name 'production', got %q", projects[0].Name)
	}
	if projects[0].ProjectID != "prod-project-123" {
		t.Errorf("unexpected ProjectID: %q", projects[0].ProjectID)
	}
	if projects[1].CredentialsFile != "/path/to/staging-sa.json" {
		t.Errorf("unexpected CredentialsFile: %q", projects[1].CredentialsFile)
	}
}
