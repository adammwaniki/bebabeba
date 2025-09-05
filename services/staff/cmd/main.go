// services/staff/cmd/main.go
package main

import (
	"log"
	"net"
	"os"

	"github.com/adammwaniki/bebabeba/services/staff/api"
	"github.com/adammwaniki/bebabeba/services/staff/internal/service"
	"github.com/adammwaniki/bebabeba/services/staff/internal/store"
	"github.com/adammwaniki/bebabeba/services/staff/internal/types"
	_ "github.com/joho/godotenv/autoload"
	"google.golang.org/grpc"
)

var (
	grpcAddr = os.Getenv("STAFF_GRPC_ADDR")
)

func main() {
	// Initialize database store
	staffStore, err := store.NewStore(os.Getenv("DRIVER_DB_DSN"))
	if err != nil {
		log.Fatal("Store initialization failed: ", err)
	}

	// Initialize service business logic
	svc := service.NewService(staffStore)

	// Start gRPC server
	startGRPCServer(svc)
}

func startGRPCServer(svc types.StaffService) {
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("gRPC listener failed: ", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer()
	api.NewGRPCHandler(grpcServer, svc)

	log.Printf("Starting Staff gRPC server on %s", grpcAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("gRPC server failed: ", err)
	}
}

