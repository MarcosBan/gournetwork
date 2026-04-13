package storage_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gournetwork/internal/adapters/secondary/storage"
	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

func TestSaveVPC_CreatesFileWithCorrectContent(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	v := &vpc.VPC{
		ID:        "vpc-abc123",
		Name:      "prod-vpc",
		Region:    "us-east-1",
		Provider:  vpc.ProviderAWS,
		CIDRBlock: "10.0.0.0/16",
		Subnets: []vpc.Subnet{
			{ID: "subnet-1", CIDRBlock: "10.0.1.0/24", Zone: "us-east-1a"},
		},
	}

	if err := repo.SaveVPC(context.Background(), v); err != nil {
		t.Fatalf("SaveVPC returned error: %v", err)
	}

	path := filepath.Join(dir, "aws", "vpc", "us-east-1", "vpc-abc123.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", path, err)
	}

	var got vpc.VPC
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal stored JSON: %v", err)
	}
	if got.ID != "vpc-abc123" {
		t.Errorf("expected ID vpc-abc123, got %q", got.ID)
	}
	if got.CIDRBlock != "10.0.0.0/16" {
		t.Errorf("expected CIDR 10.0.0.0/16, got %q", got.CIDRBlock)
	}
	if len(got.Subnets) != 1 {
		t.Errorf("expected 1 subnet, got %d", len(got.Subnets))
	}
}

func TestSaveVPC_CreatesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	v := &vpc.VPC{
		ID:       "vpc-nested",
		Region:   "eu-west-1",
		Provider: vpc.ProviderAWS,
	}

	if err := repo.SaveVPC(context.Background(), v); err != nil {
		t.Fatalf("SaveVPC returned error: %v", err)
	}

	path := filepath.Join(dir, "aws", "vpc", "eu-west-1", "vpc-nested.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", path)
	}
}

func TestSaveVPC_GCPProvider(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	v := &vpc.VPC{
		ID:       "prod-network",
		Region:   "us-central1",
		Provider: vpc.ProviderGCP,
	}

	if err := repo.SaveVPC(context.Background(), v); err != nil {
		t.Fatalf("SaveVPC returned error: %v", err)
	}

	path := filepath.Join(dir, "gcp", "vpc", "us-central1", "prod-network.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file at %s", path)
	}
}

func TestSaveVPC_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	v := &vpc.VPC{ID: "vpc-over", Region: "us-east-1", Provider: vpc.ProviderAWS, CIDRBlock: "10.0.0.0/16"}
	if err := repo.SaveVPC(context.Background(), v); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	v.CIDRBlock = "172.16.0.0/12"
	if err := repo.SaveVPC(context.Background(), v); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	path := filepath.Join(dir, "aws", "vpc", "us-east-1", "vpc-over.json")
	data, _ := os.ReadFile(path)
	var got vpc.VPC
	json.Unmarshal(data, &got)
	if got.CIDRBlock != "172.16.0.0/12" {
		t.Errorf("expected overwritten CIDR 172.16.0.0/12, got %q", got.CIDRBlock)
	}
}

func TestSaveSecurityGroup_CreatesFileWithCorrectContent(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	sg := &security.SecurityGroup{
		ID:       "sg-web",
		Name:     "web-sg",
		Provider: "aws",
		VPCID:    "vpc-abc",
		Rules: []security.SecurityRule{
			{Direction: "ingress", Protocol: "tcp", PortRange: security.PortRange{From: 80, To: 80}, Action: "allow"},
		},
	}

	if err := repo.SaveSecurityGroup(context.Background(), sg); err != nil {
		t.Fatalf("SaveSecurityGroup returned error: %v", err)
	}

	path := filepath.Join(dir, "aws", "security", sg.VPCID, "sg-web.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", path, err)
	}

	var got security.SecurityGroup
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal stored JSON: %v", err)
	}
	if got.ID != "sg-web" {
		t.Errorf("expected ID sg-web, got %q", got.ID)
	}
	if len(got.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(got.Rules))
	}
}

func TestSaveSecurityGroup_GCPProvider(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	sg := &security.SecurityGroup{
		ID:       "allow-http",
		Provider: "gcp",
		VPCID:    "prod-network",
	}

	if err := repo.SaveSecurityGroup(context.Background(), sg); err != nil {
		t.Fatalf("SaveSecurityGroup returned error: %v", err)
	}

	path := filepath.Join(dir, "gcp", "security", "prod-network", "allow-http.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file at %s", path)
	}
}

func TestJSONFileRepository_JSONIsIndented(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	v := &vpc.VPC{ID: "vpc-fmt", Region: "us-east-1", Provider: vpc.ProviderAWS}
	repo.SaveVPC(context.Background(), v)

	path := filepath.Join(dir, "aws", "vpc", "us-east-1", "vpc-fmt.json")
	data, _ := os.ReadFile(path)
	if len(data) < 2 || data[0] != '{' || data[1] != '\n' {
		t.Errorf("expected indented JSON, got: %q", string(data[:min(20, len(data))]))
	}
}

// --- ListVPCs tests ---

func TestListVPCs_ReturnsStoredVPCs(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)
	ctx := context.Background()

	repo.SaveVPC(ctx, &vpc.VPC{ID: "vpc-001", Region: "us-east-1", Provider: vpc.ProviderAWS, CIDRBlock: "10.0.0.0/16"})
	repo.SaveVPC(ctx, &vpc.VPC{ID: "vpc-002", Region: "us-east-1", Provider: vpc.ProviderAWS, CIDRBlock: "10.1.0.0/16"})

	vpcs, err := repo.ListVPCs(ctx, "aws", "us-east-1")
	if err != nil {
		t.Fatalf("ListVPCs error: %v", err)
	}
	if len(vpcs) != 2 {
		t.Errorf("expected 2 VPCs, got %d", len(vpcs))
	}
}

func TestListVPCs_EmptyWhenNoFiles(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	vpcs, err := repo.ListVPCs(context.Background(), "aws", "us-east-1")
	if err != nil {
		t.Fatalf("ListVPCs error: %v", err)
	}
	if len(vpcs) != 0 {
		t.Errorf("expected 0 VPCs, got %d", len(vpcs))
	}
}

func TestListVPCs_AllRegions(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)
	ctx := context.Background()

	repo.SaveVPC(ctx, &vpc.VPC{ID: "vpc-001", Region: "us-east-1", Provider: vpc.ProviderAWS})
	repo.SaveVPC(ctx, &vpc.VPC{ID: "vpc-002", Region: "us-west-2", Provider: vpc.ProviderAWS})

	vpcs, err := repo.ListVPCs(ctx, "aws", "")
	if err != nil {
		t.Fatalf("ListVPCs error: %v", err)
	}
	if len(vpcs) != 2 {
		t.Errorf("expected 2 VPCs across all regions, got %d", len(vpcs))
	}
}

// --- ListSecurityGroups tests ---

func TestListSecurityGroups_ReturnsStoredGroups(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)
	ctx := context.Background()

	repo.SaveSecurityGroup(ctx, &security.SecurityGroup{ID: "sg-001", Provider: "aws", VPCID: "vpc-001"})
	repo.SaveSecurityGroup(ctx, &security.SecurityGroup{ID: "sg-002", Provider: "aws", VPCID: "vpc-001"})

	groups, err := repo.ListSecurityGroups(ctx, "aws", "vpc-001")
	if err != nil {
		t.Fatalf("ListSecurityGroups error: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2 security groups, got %d", len(groups))
	}
}

func TestListSecurityGroups_EmptyWhenNoFiles(t *testing.T) {
	dir := t.TempDir()
	repo := storage.NewJSONFileRepository(dir)

	groups, err := repo.ListSecurityGroups(context.Background(), "aws", "vpc-001")
	if err != nil {
		t.Fatalf("ListSecurityGroups error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0 security groups, got %d", len(groups))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
