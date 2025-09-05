// services/vehicle/api/handler.go
package api

import (
	"context"
	"log"

	"github.com/adammwaniki/bebabeba/services/vehicle/internal/types"
	"github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// grpcHandler implements the genproto.VehicleServiceServer interface
type grpcHandler struct {
	genproto.UnimplementedVehicleServiceServer
	service      types.VehicleService
	healthServer *health.Server
}

// NewGRPCHandler creates and registers the gRPC vehicle service handler
func NewGRPCHandler(grpcServer *grpc.Server, service types.VehicleService) {
	handler := &grpcHandler{
		service:      service,
		healthServer: health.NewServer(),
	}

	// Register the vehicle service
	genproto.RegisterVehicleServiceServer(grpcServer, handler)

	// Register gRPC health service
	grpc_health_v1.RegisterHealthServer(grpcServer, handler.healthServer)
	handler.healthServer.SetServingStatus(
		"vehicle.VehicleService",
		grpc_health_v1.HealthCheckResponse_SERVING,
	)

	log.Println("gRPC Vehicle and Health services registered")
}

// Vehicle CRUD operations

func (h *grpcHandler) CreateVehicle(ctx context.Context, req *genproto.CreateVehicleRequest) (*genproto.CreateVehicleResponse, error) {
	log.Println("Handling CreateVehicle gRPC request")
	
	resp, err := h.service.CreateVehicle(ctx, req)
	if err != nil {
		log.Printf("CreateVehicle failed: %v", err)
		return nil, err
	}

	log.Printf("CreateVehicle successful for vehicle %s", resp.Vehicle.LicensePlate)
	return resp, nil
}

func (h *grpcHandler) GetVehicle(ctx context.Context, req *genproto.GetVehicleRequest) (*genproto.GetVehicleResponse, error) {
	log.Printf("Handling GetVehicle gRPC request for ID: %s", req.VehicleId)
	
	resp, err := h.service.GetVehicle(ctx, req)
	if err != nil {
		log.Printf("GetVehicle failed: %v", err)
		return nil, err
	}

	log.Printf("GetVehicle successful for vehicle %s", resp.Vehicle.LicensePlate)
	return resp, nil
}

func (h *grpcHandler) ListVehicles(ctx context.Context, req *genproto.ListVehiclesRequest) (*genproto.ListVehiclesResponse, error) {
	log.Println("Handling ListVehicles gRPC request")
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("ListVehicles: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.ListVehicles(ctx, req)
	if err != nil {
		log.Printf("ListVehicles failed: %v", err)
		return nil, err
	}

	log.Printf("ListVehicles successful, returned %d vehicles", len(resp.Vehicles))
	return resp, nil
}

func (h *grpcHandler) UpdateVehicle(ctx context.Context, req *genproto.UpdateVehicleRequest) (*genproto.UpdateVehicleResponse, error) {
	log.Printf("Handling UpdateVehicle gRPC request for ID: %s", req.VehicleId)
	
	resp, err := h.service.UpdateVehicle(ctx, req)
	if err != nil {
		log.Printf("UpdateVehicle failed: %v", err)
		return nil, err
	}

	log.Printf("UpdateVehicle successful for vehicle %s", resp.Vehicle.LicensePlate)
	return resp, nil
}

func (h *grpcHandler) DeleteVehicle(ctx context.Context, req *genproto.DeleteVehicleRequest) (*emptypb.Empty, error) {
	log.Printf("Handling DeleteVehicle gRPC request for ID: %s", req.VehicleId)
	
	err := h.service.DeleteVehicle(ctx, req)
	if err != nil {
		log.Printf("DeleteVehicle failed: %v", err)
		return nil, err
	}

	log.Printf("DeleteVehicle successful for vehicle ID: %s", req.VehicleId)
	return &emptypb.Empty{}, nil
}

// Specialized queries

func (h *grpcHandler) GetVehiclesByType(ctx context.Context, req *genproto.GetVehiclesByTypeRequest) (*genproto.ListVehiclesResponse, error) {
	log.Printf("Handling GetVehiclesByType gRPC request for type: %s", req.VehicleTypeId)
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("GetVehiclesByType: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.GetVehiclesByType(ctx, req)
	if err != nil {
		log.Printf("GetVehiclesByType failed: %v", err)
		return nil, err
	}

	log.Printf("GetVehiclesByType successful, returned %d vehicles", len(resp.Vehicles))
	return resp, nil
}

func (h *grpcHandler) GetAvailableVehicles(ctx context.Context, req *genproto.GetAvailableVehiclesRequest) (*genproto.ListVehiclesResponse, error) {
	log.Println("Handling GetAvailableVehicles gRPC request")
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("GetAvailableVehicles: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.GetAvailableVehicles(ctx, req)
	if err != nil {
		log.Printf("GetAvailableVehicles failed: %v", err)
		return nil, err
	}

	log.Printf("GetAvailableVehicles successful, returned %d vehicles", len(resp.Vehicles))
	return resp, nil
}

func (h *grpcHandler) UpdateVehicleStatus(ctx context.Context, req *genproto.UpdateVehicleStatusRequest) (*genproto.UpdateVehicleStatusResponse, error) {
	log.Printf("Handling UpdateVehicleStatus gRPC request for vehicle %s to status %s", 
		req.VehicleId, req.Status.String())
	
	resp, err := h.service.UpdateVehicleStatus(ctx, req)
	if err != nil {
		log.Printf("UpdateVehicleStatus failed: %v", err)
		return nil, err
	}

	log.Printf("UpdateVehicleStatus successful for vehicle %s", resp.Vehicle.LicensePlate)
	return resp, nil
}

// Vehicle type management

func (h *grpcHandler) CreateVehicleType(ctx context.Context, req *genproto.CreateVehicleTypeRequest) (*genproto.CreateVehicleTypeResponse, error) {
	log.Printf("Handling CreateVehicleType gRPC request for type: %s", req.Name)
	
	resp, err := h.service.CreateVehicleType(ctx, req)
	if err != nil {
		log.Printf("CreateVehicleType failed: %v", err)
		return nil, err
	}

	log.Printf("CreateVehicleType successful for type %s", resp.VehicleType.Name)
	return resp, nil
}

func (h *grpcHandler) ListVehicleTypes(ctx context.Context, req *genproto.ListVehicleTypesRequest) (*genproto.ListVehicleTypesResponse, error) {
	log.Println("Handling ListVehicleTypes gRPC request")
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("ListVehicleTypes: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.ListVehicleTypes(ctx, req)
	if err != nil {
		log.Printf("ListVehicleTypes failed: %v", err)
		return nil, err
	}

	log.Printf("ListVehicleTypes successful, returned %d types", len(resp.VehicleTypes))
	return resp, nil
}