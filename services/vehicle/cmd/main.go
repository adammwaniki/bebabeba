// services/vehicle/cmd/main.go
package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/adammwaniki/bebabeba/services/vehicle/api"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/service"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/store"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/types"
	_ "github.com/joho/godotenv/autoload"
	"google.golang.org/grpc"
)

var (
	grpcAddr = os.Getenv("VEHICLE_GRPC_ADDR")
)

func main() {
	// Initialize database store
	vehicleStore, err := store.NewStore(os.Getenv("TRANSPORT_DB_DSN"))
	if err != nil {
		log.Fatal("Store initialization failed: ", err)
	}

	// Initialize service business logic
	svc := service.NewService(vehicleStore)

	// Initialize standard vehicle types
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := svc.InitializeStandardVehicleTypes(ctx); err != nil {
		log.Printf("Warning: Failed to initialize standard vehicle types: %v", err)
	}

	// Start gRPC server
	startGRPCServer(svc)
}

func startGRPCServer(svc types.VehicleService) {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("gRPC listener failed: ", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	api.NewGRPCHandler(grpcServer, svc)

	log.Printf("Starting Vehicle gRPC server on %s", grpcAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("gRPC server failed: ", err)
	}
}