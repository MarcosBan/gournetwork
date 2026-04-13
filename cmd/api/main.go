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
	"gournetwork/internal/ports/secondary"
	"gournetwork/internal/service"
)

func main() {
	ctx := context.Background()

	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

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

	// --- Provider Registry ---
	registry := secondary.NewProviderRegistry()

	// --- AWS credential registry ---
	awsRegistry, err := awsadapter.NewAWSClientRegistry(ctx, cfg.AWS)
	if err != nil {
		log.Fatalf("AWS credential validation failed: %v", err)
	}
	registry.RegisterVPC("aws", awsadapter.NewAWSVPCRepositoryFromRegistry(awsRegistry))
	registry.RegisterSecurity("aws", awsadapter.NewAWSSecurityRepositoryFromRegistry(awsRegistry))

	// --- GCP credential registry ---
	gcpRegistry, err := gcpadapter.NewGCPClientRegistry(ctx, cfg.GCP)
	if err != nil {
		log.Fatalf("GCP credential validation failed: %v", err)
	}
	registry.RegisterVPC("gcp", gcpadapter.NewGCPVPCRepositoryFromRegistry(gcpRegistry))
	registry.RegisterSecurity("gcp", gcpadapter.NewGCPSecurityRepositoryFromRegistry(gcpRegistry))

	// --- Services ---
	vpcSvc := service.NewVPCService(registry, store)
	secSvc := service.NewSecurityService(registry, store)
	analyseSvc := service.NewAnalyseService(registry)
	mapSvc := service.NewMapService(registry)

	// --- Handlers ---
	vpcHandler := httphandler.NewVPCHTTPHandler(vpcSvc)
	secHandler := httphandler.NewSecurityHTTPHandler(secSvc)
	analyseHandler := httphandler.NewAnalyseHTTPHandler(analyseSvc)
	mapHandler := httphandler.NewMapHTTPHandler(mapSvc)

	mux := http.NewServeMux()

	// VPC routes (unified — provider is a query param)
	mux.HandleFunc("GET /vpc/describe/{vpcID}", vpcHandler.DescribeVPC)
	mux.HandleFunc("POST /vpc/insert", vpcHandler.InsertVPC)

	// Security rules (unified — provider is a query param)
	mux.HandleFunc("GET /security-rules/describe", secHandler.DescribeSecurityGroup)
	mux.HandleFunc("POST /security-rules/insert", secHandler.InsertRule)
	mux.HandleFunc("DELETE /security-rules/remove", secHandler.RemoveRule)

	// Connectivity analysis (multicloud)
	mux.HandleFunc("POST /analyse", analyseHandler.AnalyseConnectivity)

	// Network map overview (multicloud)
	mux.HandleFunc("GET /map", mapHandler.GetNetworkMap)

	log.Printf("Starting gournetwork API server on %s", cfg.Global.Port)
	if err = http.ListenAndServe(cfg.Global.Port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
