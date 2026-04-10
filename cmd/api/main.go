package main

import (
	"context"
	"log"
	"net/http"

	httphandler "gournetwork/internal/adapters/primary/http"
	awsadapter "gournetwork/internal/adapters/secondary/aws"
	gcpadapter "gournetwork/internal/adapters/secondary/gcp"
	"gournetwork/internal/adapters/secondary/storage"
	"gournetwork/internal/config"
)

func main() {
	ctx := context.Background()

	// --- Configuration ---
	// Load global settings and per-provider credential sets from environment
	// variables and optional config files (aws.config, gcp.config).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	// Validate that at least one cloud provider is configured before binding
	// any cloud-SDK clients, so the error is surfaced immediately at startup.
	if err = cfg.Validate(ctx); err != nil {
		log.Fatalf("credential configuration error: %v\n\n"+
			"AWS accounts  — set AWS_{NAME}_ACCESS_KEY_ID + AWS_{NAME}_SECRET_ACCESS_KEY,\n"+
			"              or AWS_{NAME}_CREDENTIALS=key:secret[:token],\n"+
			"              or list them in aws.config (INI format).\n\n"+
			"GCP projects  — set GCP_{NAME}_PROJECT_ID + GCP_{NAME}_CREDENTIALS_FILE,\n"+
			"              or list them in gcp.config (INI format).\n"+
			"              Use GCP_PROJECT_ID for a single default project (uses ADC if no\n"+
			"              credentials file is set).", err)
	}

	log.Printf("Loaded %d AWS account(s) and %d GCP project(s)",
		len(cfg.AWS.Accounts), len(cfg.GCP.Projects))

	// --- Storage ---
	store := storage.NewJSONFileRepository("infra/databases/text")

	// --- AWS credential registry ---
	// One EC2 client per account; credentials validated here at startup.
	awsRegistry, err := awsadapter.NewAWSClientRegistry(ctx, cfg.AWS)
	if err != nil {
		log.Fatalf("AWS credential validation failed: %v", err)
	}
	awsVPCRepo := awsadapter.NewAWSVPCRepositoryFromRegistry(awsRegistry)
	awsSecRepo := awsadapter.NewAWSSecurityRepositoryFromRegistry(awsRegistry)

	// --- GCP credential registry ---
	// One compute.Service per project; credentials validated here at startup.
	gcpRegistry, err := gcpadapter.NewGCPClientRegistry(ctx, cfg.GCP)
	if err != nil {
		log.Fatalf("GCP credential validation failed: %v", err)
	}
	gcpVPCRepo := gcpadapter.NewGCPVPCRepositoryFromRegistry(gcpRegistry)
	gcpSecRepo := gcpadapter.NewGCPSecurityRepositoryFromRegistry(gcpRegistry)

	// --- Handlers ---
	awsVPCHandler := httphandler.NewVPCHTTPHandler(awsVPCRepo, store)
	awsSecHandler := httphandler.NewSecurityHTTPHandler(awsSecRepo, store)
	gcpVPCHandler := httphandler.NewVPCHTTPHandler(gcpVPCRepo, store)
	gcpSecHandler := httphandler.NewSecurityHTTPHandler(gcpSecRepo, store)

	analyseHandler := httphandler.NewAnalyseHTTPHandler(awsVPCRepo, awsSecRepo)
	mapHandler := httphandler.NewMapHTTPHandler(awsVPCRepo)

	mux := http.NewServeMux()

	// VPC routes — AWS
	// GET  /aws/vpc/describe/{vpcID}?provider=aws&account=<alias>&region=<r>
	// POST /aws/vpc/insert            body: {provider, account, region, vpcID}
	mux.HandleFunc("GET /aws/vpc/describe/{vpcID}", awsVPCHandler.DescribeVPC)
	mux.HandleFunc("POST /aws/vpc/insert", awsVPCHandler.InsertVPC)

	// VPC routes — GCP
	// GET  /gcp/vpc/describe/{vpcID}?provider=gcp&account=<project-alias>&region=<r>
	// POST /gcp/vpc/insert            body: {provider, account, region, vpcID}
	mux.HandleFunc("GET /gcp/vpc/describe/{vpcID}", gcpVPCHandler.DescribeVPC)
	mux.HandleFunc("POST /gcp/vpc/insert", gcpVPCHandler.InsertVPC)

	// Security rules — AWS
	mux.HandleFunc("GET /aws/security-rules/describe", awsSecHandler.DescribeSecurityGroup)
	mux.HandleFunc("POST /aws/security-rules/insert", awsSecHandler.InsertRule)
	mux.HandleFunc("DELETE /aws/security-rules/remove", awsSecHandler.RemoveRule)

	// Security rules — GCP
	mux.HandleFunc("GET /gcp/security-rules/describe", gcpSecHandler.DescribeSecurityGroup)
	mux.HandleFunc("POST /gcp/security-rules/insert", gcpSecHandler.InsertRule)
	mux.HandleFunc("DELETE /gcp/security-rules/remove", gcpSecHandler.RemoveRule)

	// Connectivity analysis
	mux.HandleFunc("POST /analyse", analyseHandler.AnalyseConnectivity)

	// Network map overview
	mux.HandleFunc("GET /map", mapHandler.GetNetworkMap)

	log.Printf("Starting gournetwork API server on %s", cfg.Global.Port)
	if err = http.ListenAndServe(cfg.Global.Port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
