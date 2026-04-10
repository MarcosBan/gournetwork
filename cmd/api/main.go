package main

import (
	"log"
	"net/http"

	httphandler "gournetwork/internal/adapters/primary/http"
	awsadapter "gournetwork/internal/adapters/secondary/aws"
	_ "gournetwork/internal/adapters/secondary/gcp" // GCP adapter available for selection based on path
)

func main() {
	// Instantiate AWS repositories (concrete implementations).
	// GCP repositories can be selected based on the request path prefix.
	awsVPCRepo := awsadapter.NewAWSVPCRepository()
	awsSecRepo := awsadapter.NewAWSSecurityRepository()

	// Wire up handlers with AWS repositories.
	// To support GCP, instantiate GCPVPCRepository and GCPSecurityRepository
	// and route /gcp/* paths to handlers constructed with those repos.
	vpcHandler := httphandler.NewVPCHTTPHandler(awsVPCRepo)
	secHandler := httphandler.NewSecurityHTTPHandler(awsSecRepo)
	analyseHandler := httphandler.NewAnalyseHTTPHandler(awsVPCRepo, awsSecRepo)
	mapHandler := httphandler.NewMapHTTPHandler(awsVPCRepo)

	mux := http.NewServeMux()

	// VPC routes — AWS
	mux.HandleFunc("GET /aws/vpc/", vpcHandler.DescribeVPC)
	mux.HandleFunc("POST /aws/vpc/", vpcHandler.UpdateRoutes)

	// VPC routes — GCP (using same handler; provider is passed as query param)
	mux.HandleFunc("GET /gcp/vpc/", vpcHandler.DescribeVPC)
	mux.HandleFunc("POST /gcp/vpc/", vpcHandler.UpdateRoutes)

	// Security rules — AWS
	mux.HandleFunc("GET /aws/security-rules/describe", secHandler.DescribeSecurityGroup)
	mux.HandleFunc("POST /aws/security-rules/describe", secHandler.UpdateRule)
	mux.HandleFunc("DELETE /aws/security-rules/describe", secHandler.DeleteRule)

	// Security rules — GCP
	mux.HandleFunc("GET /gcp/security-rules/describe", secHandler.DescribeSecurityGroup)
	mux.HandleFunc("POST /gcp/security-rules/describe", secHandler.UpdateRule)
	mux.HandleFunc("DELETE /gcp/security-rules/describe", secHandler.DeleteRule)

	// Connectivity analysis
	mux.HandleFunc("POST /analyse", analyseHandler.AnalyseConnectivity)

	// Network map overview
	mux.HandleFunc("GET /map", mapHandler.GetNetworkMap)

	log.Println("Starting gournetwork API server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
