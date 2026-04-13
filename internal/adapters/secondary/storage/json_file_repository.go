package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// JSONFileRepository persists cloud resources as indented JSON files under a base directory.
//
// Directory layout:
//
//	{baseDir}/{provider}/vpc/{region}/{vpcID}.json
//	{baseDir}/{provider}/security/{vpcID}/{groupID}.json
type JSONFileRepository struct {
	baseDir string
}

// NewJSONFileRepository creates a new JSONFileRepository rooted at baseDir.
func NewJSONFileRepository(baseDir string) *JSONFileRepository {
	return &JSONFileRepository{baseDir: baseDir}
}

// SaveVPC serialises v as indented JSON and writes it to
// {baseDir}/{provider}/vpc/{region}/{vpcID}.json, creating directories as needed.
func (r *JSONFileRepository) SaveVPC(_ context.Context, v *vpc.VPC) error {
	dir := filepath.Join(r.baseDir, string(v.Provider), "vpc", v.Region)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, v.ID+".json")
	return writeJSON(path, v)
}

// SaveSecurityGroup serialises sg as indented JSON and writes it to
// {baseDir}/{provider}/security/{vpcID}/{groupID}.json, creating directories as needed.
func (r *JSONFileRepository) SaveSecurityGroup(_ context.Context, sg *security.SecurityGroup) error {
	dir := filepath.Join(r.baseDir, sg.Provider, "security", sg.VPCID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, sg.ID+".json")
	return writeJSON(path, sg)
}

// ListVPCs reads all stored VPC JSON files for a given provider and optional region.
// If region is empty, all regions are returned.
func (r *JSONFileRepository) ListVPCs(_ context.Context, provider, region string) ([]vpc.VPC, error) {
	var pattern string
	if region != "" {
		pattern = filepath.Join(r.baseDir, provider, "vpc", region, "*.json")
	} else {
		pattern = filepath.Join(r.baseDir, provider, "vpc", "*", "*.json")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var vpcs []vpc.VPC
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var v vpc.VPC
		if err := json.Unmarshal(data, &v); err != nil {
			continue
		}
		vpcs = append(vpcs, v)
	}
	return vpcs, nil
}

// ListSecurityGroups reads all stored security group JSON files for a given provider and VPC.
// If vpcID is empty, all security groups for the provider are returned.
func (r *JSONFileRepository) ListSecurityGroups(_ context.Context, provider, vpcID string) ([]security.SecurityGroup, error) {
	var pattern string
	if vpcID != "" {
		pattern = filepath.Join(r.baseDir, provider, "security", vpcID, "*.json")
	} else {
		pattern = filepath.Join(r.baseDir, provider, "security", "*", "*.json")
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob %s: %w", pattern, err)
	}

	var groups []security.SecurityGroup
	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var sg security.SecurityGroup
		if err := json.Unmarshal(data, &sg); err != nil {
			continue
		}
		groups = append(groups, sg)
	}
	return groups, nil
}

// writeJSON marshals v as indented JSON and writes it to path.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
