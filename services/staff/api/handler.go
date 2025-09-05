// services/staff/api/handler.go
package api

import (
	"context"
	"log"

	"github.com/adammwaniki/bebabeba/services/staff/internal/types"
	"github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// grpcHandler implements the genproto.StaffServiceServer interface
type grpcHandler struct {
	genproto.UnimplementedStaffServiceServer
	service      types.StaffService
	healthServer *health.Server
}

// NewGRPCHandler creates and registers the gRPC staff service handler
func NewGRPCHandler(grpcServer *grpc.Server, service types.StaffService) {
	handler := &grpcHandler{
		service:      service,
		healthServer: health.NewServer(),
	}

	// Register the staff service
	genproto.RegisterStaffServiceServer(grpcServer, handler)

	// Register gRPC health service
	grpc_health_v1.RegisterHealthServer(grpcServer, handler.healthServer)
	handler.healthServer.SetServingStatus(
		"staff.StaffService",
		grpc_health_v1.HealthCheckResponse_SERVING,
	)

	log.Println("gRPC Staff and Health services registered")
}

// Driver CRUD operations

func (h *grpcHandler) CreateDriver(ctx context.Context, req *genproto.CreateDriverRequest) (*genproto.CreateDriverResponse, error) {
	log.Println("Handling CreateDriver gRPC request")
	
	resp, err := h.service.CreateDriver(ctx, req)
	if err != nil {
		log.Printf("CreateDriver failed: %v", err)
		return nil, err
	}

	log.Printf("CreateDriver successful for user %s with license %s", 
		resp.Driver.UserId, resp.Driver.LicenseNumber)
	return resp, nil
}

func (h *grpcHandler) GetDriver(ctx context.Context, req *genproto.GetDriverRequest) (*genproto.GetDriverResponse, error) {
	log.Printf("Handling GetDriver gRPC request for ID: %s", req.DriverId)
	
	resp, err := h.service.GetDriver(ctx, req)
	if err != nil {
		log.Printf("GetDriver failed: %v", err)
		return nil, err
	}

	log.Printf("GetDriver successful for driver %s", resp.Driver.LicenseNumber)
	return resp, nil
}

func (h *grpcHandler) GetDriverByUserID(ctx context.Context, req *genproto.GetDriverByUserIDRequest) (*genproto.GetDriverResponse, error) {
	log.Printf("Handling GetDriverByUserID gRPC request for user: %s", req.UserId)
	
	resp, err := h.service.GetDriverByUserID(ctx, req)
	if err != nil {
		log.Printf("GetDriverByUserID failed: %v", err)
		return nil, err
	}

	log.Printf("GetDriverByUserID successful for user %s", req.UserId)
	return resp, nil
}

func (h *grpcHandler) ListDrivers(ctx context.Context, req *genproto.ListDriversRequest) (*genproto.ListDriversResponse, error) {
	log.Println("Handling ListDrivers gRPC request")
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("ListDrivers: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.ListDrivers(ctx, req)
	if err != nil {
		log.Printf("ListDrivers failed: %v", err)
		return nil, err
	}

	log.Printf("ListDrivers successful, returned %d drivers", len(resp.Drivers))
	return resp, nil
}

func (h *grpcHandler) UpdateDriver(ctx context.Context, req *genproto.UpdateDriverRequest) (*genproto.UpdateDriverResponse, error) {
	log.Printf("Handling UpdateDriver gRPC request for ID: %s", req.DriverId)
	
	resp, err := h.service.UpdateDriver(ctx, req)
	if err != nil {
		log.Printf("UpdateDriver failed: %v", err)
		return nil, err
	}

	log.Printf("UpdateDriver successful for driver %s", resp.Driver.LicenseNumber)
	return resp, nil
}

func (h *grpcHandler) DeleteDriver(ctx context.Context, req *genproto.DeleteDriverRequest) (*emptypb.Empty, error) {
	log.Printf("Handling DeleteDriver gRPC request for ID: %s", req.DriverId)
	
	err := h.service.DeleteDriver(ctx, req)
	if err != nil {
		log.Printf("DeleteDriver failed: %v", err)
		return nil, err
	}

	log.Printf("DeleteDriver successful for driver ID: %s", req.DriverId)
	return &emptypb.Empty{}, nil
}

// Driver status management

func (h *grpcHandler) UpdateDriverStatus(ctx context.Context, req *genproto.UpdateDriverStatusRequest) (*genproto.UpdateDriverStatusResponse, error) {
	log.Printf("Handling UpdateDriverStatus gRPC request for driver %s to status %s", 
		req.DriverId, req.Status.String())
	
	resp, err := h.service.UpdateDriverStatus(ctx, req)
	if err != nil {
		log.Printf("UpdateDriverStatus failed: %v", err)
		return nil, err
	}

	log.Printf("UpdateDriverStatus successful for driver %s", resp.Driver.LicenseNumber)
	return resp, nil
}

func (h *grpcHandler) GetActiveDrivers(ctx context.Context, req *genproto.GetActiveDriversRequest) (*genproto.ListDriversResponse, error) {
	log.Println("Handling GetActiveDrivers gRPC request")
	
	// Validate page size
	if req.GetPageSize() > 100 {
		log.Printf("GetActiveDrivers: page size %d exceeds maximum of 100", req.GetPageSize())
		req.PageSize = 100
	}

	resp, err := h.service.GetActiveDrivers(ctx, req)
	if err != nil {
		log.Printf("GetActiveDrivers failed: %v", err)
		return nil, err
	}

	log.Printf("GetActiveDrivers successful, returned %d drivers", len(resp.Drivers))
	return resp, nil
}

// Driver certification management

func (h *grpcHandler) AddDriverCertification(ctx context.Context, req *genproto.AddDriverCertificationRequest) (*genproto.AddDriverCertificationResponse, error) {
	log.Printf("Handling AddDriverCertification gRPC request for driver %s", req.DriverId)
	
	resp, err := h.service.AddDriverCertification(ctx, req)
	if err != nil {
		log.Printf("AddDriverCertification failed: %v", err)
		return nil, err
	}

	log.Printf("AddDriverCertification successful for driver %s", req.DriverId)
	return resp, nil
}

func (h *grpcHandler) ListDriverCertifications(ctx context.Context, req *genproto.ListDriverCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error) {
	log.Printf("Handling ListDriverCertifications gRPC request for driver %s", req.DriverId)
	
	resp, err := h.service.ListDriverCertifications(ctx, req)
	if err != nil {
		log.Printf("ListDriverCertifications failed: %v", err)
		return nil, err
	}

	log.Printf("ListDriverCertifications successful for driver %s", req.DriverId)
	return resp, nil
}

func (h *grpcHandler) UpdateCertification(ctx context.Context, req *genproto.UpdateCertificationRequest) (*genproto.UpdateCertificationResponse, error) {
	log.Printf("Handling UpdateCertification gRPC request for certification %s", req.CertificationId)
	
	resp, err := h.service.UpdateCertification(ctx, req)
	if err != nil {
		log.Printf("UpdateCertification failed: %v", err)
		return nil, err
	}

	log.Printf("UpdateCertification successful for certification %s", req.CertificationId)
	return resp, nil
}

func (h *grpcHandler) DeleteCertification(ctx context.Context, req *genproto.DeleteCertificationRequest) (*emptypb.Empty, error) {
	log.Printf("Handling DeleteCertification gRPC request for certification %s", req.CertificationId)
	
	err := h.service.DeleteCertification(ctx, req)
	if err != nil {
		log.Printf("DeleteCertification failed: %v", err)
		return nil, err
	}

	log.Printf("DeleteCertification successful for certification %s", req.CertificationId)
	return &emptypb.Empty{}, nil
}

// Driver verification and compliance

func (h *grpcHandler) VerifyDriverLicense(ctx context.Context, req *genproto.VerifyDriverLicenseRequest) (*genproto.VerifyDriverLicenseResponse, error) {
	log.Printf("Handling VerifyDriverLicense gRPC request for driver %s", req.DriverId)
	
	resp, err := h.service.VerifyDriverLicense(ctx, req)
	if err != nil {
		log.Printf("VerifyDriverLicense failed: %v", err)
		return nil, err
	}

	log.Printf("VerifyDriverLicense successful for driver %s", req.DriverId)
	return resp, nil
}

func (h *grpcHandler) GetExpiringLicenses(ctx context.Context, req *genproto.GetExpiringLicensesRequest) (*genproto.ListDriversResponse, error) {
	log.Printf("Handling GetExpiringLicenses gRPC request for %d days ahead", req.DaysAhead)
	
	resp, err := h.service.GetExpiringLicenses(ctx, req)
	if err != nil {
		log.Printf("GetExpiringLicenses failed: %v", err)
		return nil, err
	}

	log.Printf("GetExpiringLicenses successful, returned %d drivers", len(resp.Drivers))
	return resp, nil
}

func (h *grpcHandler) GetExpiredCertifications(ctx context.Context, req *genproto.GetExpiredCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error) {
	log.Println("Handling GetExpiredCertifications gRPC request")
	
	resp, err := h.service.GetExpiredCertifications(ctx, req)
	if err != nil {
		log.Printf("GetExpiredCertifications failed: %v", err)
		return nil, err
	}

	log.Printf("GetExpiredCertifications successful, returned %d certifications", len(resp.Certifications))
	return resp, nil
}
