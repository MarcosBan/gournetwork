package main

import (
	"context"
	"log"
	"net/http"
	"os"

	httphandler "gournetwork/internal/adapters/primary/http"
	awsadapter "gournetwork/internal/adapters/secondary/aws"
	gcpadapter "gournetwork/internal/adapters/secondary/gcp"
	"gournetwork/internal/adapters/secondary/storage"
)

func main() {
	ctx := context.Background()

	// --- Storage ---
	// JSON files are persisted under infra/databases/text/{provider}/{resource-type}/...
	store := storage.NewJSONFileRepository("infra/databases/text")

	// --- AWS ---
	awsVPCRepo, err := awsadapter.NewAWSVPCRepository(ctx)
	if err != nil {
		log.Fatalf("AWS VPC repository init failed: %v", err)
	}
	awsSecRepo, err := awsadapter.NewAWSSecurityRepository(ctx)
	if err != nil {
		log.Fatalf("AWS security repository init failed: %v", err)
	}

	// --- GCP ---
	gcpProject := os.Getenv("GCP_PROJECT_ID")
	if gcpProject == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}
	gcpVPCRepo, err := gcpadapter.NewGCPVPCRepository(ctx, gcpProject)
	if err != nil {
		log.Fatalf("GCP VPC repository init failed: %v", err)
	}
	gcpSecRepo, err := gcpadapter.NewGCPSecurityRepository(ctx, gcpProject)
	if err != nil {
		log.Fatalf("GCP security repository init failed: %v", err)
	}

	// --- Handlers ---
	awsVPCHandler := httphandler.NewVPCHTTPHandler(awsVPCRepo, store)
	awsSecHandler := httphandler.NewSecurityHTTPHandler(awsSecRepo, store)
	gcpVPCHandler := httphandler.NewVPCHTTPHandler(gcpVPCRepo, store)
	gcpSecHandler := httphandler.NewSecurityHTTPHandler(gcpSecRepo, store)

	analyseHandler := httphandler.NewAnalyseHTTPHandler(awsVPCRepo, awsSecRepo)
	mapHandler := httphandler.NewMapHTTPHandler(awsVPCRepo)

	mux := http.NewServeMux()

	// VPC routes — AWS
	// GET  /aws/vpc/describe/{vpcID}  — describe VPC (vpcID in path, provider/region in query)
	// POST /aws/vpc/insert            — scrape basic IDs from cloud and store JSON
	mux.HandleFunc("GET /aws/vpc/describe/{vpcID}", awsVPCHandler.DescribeVPC)
	mux.HandleFunc("POST /aws/vpc/insert", awsVPCHandler.InsertVPC)

	// VPC routes — GCP
	mux.HandleFunc("GET /gcp/vpc/describe/{vpcID}", gcpVPCHandler.DescribeVPC)
	mux.HandleFunc("POST /gcp/vpc/insert", gcpVPCHandler.InsertVPC)

	// Security rules — AWS
	// GET    /aws/security-rules/describe  — describe by groupID query param
	// POST   /aws/security-rules/insert    — scrape security group and store JSON
	// DELETE /aws/security-rules/remove    — remove rule by groupID+ruleID query params
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

	log.Println("Starting gournetwork API server on :8080")
	if err = http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
