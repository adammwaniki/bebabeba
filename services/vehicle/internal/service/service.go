// services/vehicle/internal/service/service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/adammwaniki/bebabeba/services/common/utils"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/types"
	"github.com/adammwaniki/bebabeba/services/vehicle/internal/validator"
	"github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"github.com/influxdata/influxdb/v2/pkg/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type service struct {
	store types.VehicleStore
}

// NewService creates a new vehicle service instance
func NewService(store types.VehicleStore) *service {
	return &service{store: store}
}

// Vehicle CRUD operations

func (s *service) CreateVehicle(ctx context.Context, req *genproto.CreateVehicleRequest) (*genproto.CreateVehicleResponse, error) {
	// Validate the request
	if err := validator.ValidateCreateVehicleRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	vehicle := req.Vehicle

	// Verify vehicle type exists
	_, err := s.store.GetVehicleTypeByID(ctx, vehicle.VehicleTypeId)
	if err != nil {
		if errors.Is(err, types.ErrVehicleTypeNotFound) {
			return nil, status.Errorf(codes.InvalidArgument, "vehicle type not found: %s", vehicle.VehicleTypeId)
		}
		return nil, status.Errorf(codes.Internal, "failed to validate vehicle type: %v", err)
	}

	// Check for duplicate license plate
	existing, err := s.store.GetVehicleByLicensePlate(ctx, vehicle.LicensePlate)
	if err != nil && !errors.Is(err, types.ErrVehicleNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check license plate uniqueness: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "vehicle with license plate %s already exists", vehicle.LicensePlate)
	}

	// Generate unique IDs
	nodeID, err := utils.GetSnowflakeNodeID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get snowflake node ID: %v", err)
	}

	generator := snowflake.New(int(nodeID))
	internalID := generator.Next()

	externalID, err := uuid.NewV4()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate UUID: %v", err)
	}

	// Prepare vehicle data
	vehicleData := &types.VehicleData{
		VehicleTypeID:   vehicle.VehicleTypeId,
		LicensePlate:    vehicle.LicensePlate,
		Make:            vehicle.Make,
		Model:           vehicle.Model,
		Year:            vehicle.Year,
		Color:           vehicle.Color,
		SeatingCapacity: vehicle.SeatingCapacity,
		FuelType:        vehicle.FuelType,
	}

	// Handle optional fields
	if vehicle.EngineNumber != "" {
		vehicleData.EngineNumber = &vehicle.EngineNumber
	}
	if vehicle.ChassisNumber != "" {
		vehicleData.ChassisNumber = &vehicle.ChassisNumber
	}
	if vehicle.RegistrationDate != nil {
		regDateStr := vehicle.RegistrationDate.AsTime().Format("2006-01-02")
		vehicleData.RegistrationDate = &regDateStr
	}
	if vehicle.InsuranceExpiry != nil {
		expDateStr := vehicle.InsuranceExpiry.AsTime().Format("2006-01-02")
		vehicleData.InsuranceExpiry = &expDateStr
	}

	// Create vehicle in store
	if err := s.store.CreateVehicle(ctx, internalID, externalID, vehicleData); err != nil {
		if errors.Is(err, types.ErrDuplicateEntry) {
			return nil, status.Errorf(codes.AlreadyExists, "vehicle with this license plate already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create vehicle: %v", err)
	}

	// Retrieve the created vehicle
	createdVehicle, err := s.store.GetVehicleByID(ctx, externalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve created vehicle: %v", err)
	}

	return &genproto.CreateVehicleResponse{
		Vehicle: createdVehicle,
	}, nil
}

func (s *service) GetVehicle(ctx context.Context, req *genproto.GetVehicleRequest) (*genproto.GetVehicleResponse, error) {
	if req.VehicleId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vehicle ID is required")
	}

	// Parse UUID
	vehicleID, err := uuid.FromString(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle ID format: %v", err)
	}

	// Get vehicle from store
	vehicle, err := s.store.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get vehicle: %v", err)
	}

	return &genproto.GetVehicleResponse{
		Vehicle: vehicle,
	}, nil
}

func (s *service) ListVehicles(ctx context.Context, req *genproto.ListVehiclesRequest) (*genproto.ListVehiclesResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Prepare parameters
	params := types.ListVehiclesParams{
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	}

	if req.StatusFilter != nil {
		params.StatusFilter = req.StatusFilter
	}
	if req.VehicleTypeFilter != nil && *req.VehicleTypeFilter != "" {
		params.VehicleTypeFilter = req.VehicleTypeFilter
	}
	if req.MakeFilter != nil && *req.MakeFilter != "" {
		params.MakeFilter = req.MakeFilter
	}

	// Get vehicles from store
	vehicles, nextPageToken, err := s.store.ListVehicles(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list vehicles: %v", err)
	}

	return &genproto.ListVehiclesResponse{
		Vehicles:      vehicles,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(vehicles)), // Note: This is just the current page count
	}, nil
}

func (s *service) UpdateVehicle(ctx context.Context, req *genproto.UpdateVehicleRequest) (*genproto.UpdateVehicleResponse, error) {
	// Validate the request
	if err := validator.ValidateUpdateVehicleRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Parse vehicle ID
	vehicleID, err := uuid.FromString(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle ID format: %v", err)
	}

	// Check if vehicle exists
	existingVehicle, err := s.store.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get existing vehicle: %v", err)
	}

	vehicle := req.Vehicle

	// Validate vehicle type if being updated
	if vehicle.VehicleTypeId != "" {
		_, err := s.store.GetVehicleTypeByID(ctx, vehicle.VehicleTypeId)
		if err != nil {
			if errors.Is(err, types.ErrVehicleTypeNotFound) {
				return nil, status.Errorf(codes.InvalidArgument, "vehicle type not found: %s", vehicle.VehicleTypeId)
			}
			return nil, status.Errorf(codes.Internal, "failed to validate vehicle type: %v", err)
		}
	}

	// Check license plate uniqueness if being updated
	if vehicle.LicensePlate != "" && vehicle.LicensePlate != existingVehicle.LicensePlate {
		existing, err := s.store.GetVehicleByLicensePlate(ctx, vehicle.LicensePlate)
		if err != nil && !errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.Internal, "failed to check license plate uniqueness: %v", err)
		}
		if existing != nil && existing.Id != existingVehicle.Id {
			return nil, status.Errorf(codes.AlreadyExists, "vehicle with license plate %s already exists", vehicle.LicensePlate)
		}
	}

	// Prepare update fields
	updates := types.VehicleUpdateFields{}
	if vehicle.VehicleTypeId != "" {
		updates.VehicleTypeID = &vehicle.VehicleTypeId
	}
	if vehicle.LicensePlate != "" {
		updates.LicensePlate = &vehicle.LicensePlate
	}
	if vehicle.Make != "" {
		updates.Make = &vehicle.Make
	}
	if vehicle.Model != "" {
		updates.Model = &vehicle.Model
	}
	if vehicle.Year != 0 {
		updates.Year = &vehicle.Year
	}
	if vehicle.Color != "" {
		updates.Color = &vehicle.Color
	}
	if vehicle.SeatingCapacity != 0 {
		updates.SeatingCapacity = &vehicle.SeatingCapacity
	}
	if vehicle.FuelType != genproto.FuelType_FUEL_UNSPECIFIED {
		updates.FuelType = &vehicle.FuelType
	}
	if vehicle.EngineNumber != "" {
		updates.EngineNumber = &vehicle.EngineNumber
	}
	if vehicle.ChassisNumber != "" {
		updates.ChassisNumber = &vehicle.ChassisNumber
	}
	if vehicle.RegistrationDate != nil {
		regDateStr := vehicle.RegistrationDate.AsTime().Format("2006-01-02")
		updates.RegistrationDate = &regDateStr
	}
	if vehicle.InsuranceExpiry != nil {
		expDateStr := vehicle.InsuranceExpiry.AsTime().Format("2006-01-02")
		updates.InsuranceExpiry = &expDateStr
	}

	// Update vehicle in store
	updatedVehicle, err := s.store.UpdateVehicle(ctx, vehicleID, updates, req.UpdateMask)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle not found")
		}
		if errors.Is(err, types.ErrDuplicateEntry) {
			return nil, status.Errorf(codes.AlreadyExists, "duplicate license plate")
		}
		return nil, status.Errorf(codes.Internal, "failed to update vehicle: %v", err)
	}

	return &genproto.UpdateVehicleResponse{
		Vehicle: updatedVehicle,
	}, nil
}

func (s *service) DeleteVehicle(ctx context.Context, req *genproto.DeleteVehicleRequest) error {
	if req.VehicleId == "" {
		return status.Errorf(codes.InvalidArgument, "vehicle ID is required")
	}

	// Parse vehicle ID
	vehicleID, err := uuid.FromString(req.VehicleId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid vehicle ID format: %v", err)
	}

	// Check if vehicle exists and get current status
	existingVehicle, err := s.store.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return status.Errorf(codes.NotFound, "vehicle not found")
		}
		return status.Errorf(codes.Internal, "failed to get vehicle: %v", err)
	}

	// Business rule: Cannot delete assigned vehicles
	if existingVehicle.Status == genproto.VehicleStatus_ASSIGNED {
		return status.Errorf(codes.FailedPrecondition, "cannot delete assigned vehicle. Unassign vehicle first")
	}

	// Soft delete by setting status to RETIRED
	if err := s.store.DeleteVehicle(ctx, vehicleID); err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return status.Errorf(codes.NotFound, "vehicle not found")
		}
		return status.Errorf(codes.Internal, "failed to delete vehicle: %v", err)
	}

	return nil
}

// Specialized queries

func (s *service) GetVehiclesByType(ctx context.Context, req *genproto.GetVehiclesByTypeRequest) (*genproto.ListVehiclesResponse, error) {
	if req.VehicleTypeId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vehicle type ID is required")
	}

	// Verify vehicle type exists
	_, err := s.store.GetVehicleTypeByID(ctx, req.VehicleTypeId)
	if err != nil {
		if errors.Is(err, types.ErrVehicleTypeNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle type not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to validate vehicle type: %v", err)
	}

	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListVehiclesParams{
		PageSize:     pageSize,
		PageToken:    req.GetPageToken(),
		StatusFilter: req.StatusFilter,
	}

	vehicles, nextPageToken, err := s.store.GetVehiclesByType(ctx, req.VehicleTypeId, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get vehicles by type: %v", err)
	}

	return &genproto.ListVehiclesResponse{
		Vehicles:      vehicles,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(vehicles)),
	}, nil
}

func (s *service) GetAvailableVehicles(ctx context.Context, req *genproto.GetAvailableVehiclesRequest) (*genproto.ListVehiclesResponse, error) {
	// Validate vehicle type if provided
	if req.VehicleTypeId != nil && *req.VehicleTypeId != "" {
		_, err := s.store.GetVehicleTypeByID(ctx, *req.VehicleTypeId)
		if err != nil {
			if errors.Is(err, types.ErrVehicleTypeNotFound) {
				return nil, status.Errorf(codes.NotFound, "vehicle type not found")
			}
			return nil, status.Errorf(codes.Internal, "failed to validate vehicle type: %v", err)
		}
	}

	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListVehiclesParams{
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	}

	vehicles, nextPageToken, err := s.store.GetAvailableVehicles(ctx, req.VehicleTypeId, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get available vehicles: %v", err)
	}

	return &genproto.ListVehiclesResponse{
		Vehicles:      vehicles,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(vehicles)),
	}, nil
}

func (s *service) UpdateVehicleStatus(ctx context.Context, req *genproto.UpdateVehicleStatusRequest) (*genproto.UpdateVehicleStatusResponse, error) {
	if req.VehicleId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vehicle ID is required")
	}

	// Validate status
	if err := validator.ValidateVehicleStatus("status", req.Status); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Parse vehicle ID
	vehicleID, err := uuid.FromString(req.VehicleId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid vehicle ID format: %v", err)
	}

	// Get current vehicle to check status transition
	currentVehicle, err := s.store.GetVehicleByID(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get current vehicle: %v", err)
	}

	// Check if status transition is valid
	if !types.IsValidStatusTransition(currentVehicle.Status, req.Status) {
		return nil, status.Errorf(codes.InvalidArgument, 
			"invalid status transition from %s to %s", 
			currentVehicle.Status.String(), req.Status.String())
	}

	// Update status
	updatedVehicle, err := s.store.UpdateVehicleStatus(ctx, vehicleID, req.Status)
	if err != nil {
		if errors.Is(err, types.ErrVehicleNotFound) {
			return nil, status.Errorf(codes.NotFound, "vehicle not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update vehicle status: %v", err)
	}

	log.Printf("Vehicle %s status updated from %s to %s", 
		req.VehicleId, currentVehicle.Status.String(), req.Status.String())

	return &genproto.UpdateVehicleStatusResponse{
		Vehicle: updatedVehicle,
	}, nil
}

// Vehicle type management

func (s *service) CreateVehicleType(ctx context.Context, req *genproto.CreateVehicleTypeRequest) (*genproto.CreateVehicleTypeResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "vehicle type name is required")
	}

	// Validate vehicle type name
	if err := validator.ValidateVehicleTypeName("name", req.Name); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Check if vehicle type already exists
	existing, err := s.store.GetVehicleTypeByName(ctx, req.Name)
	if err != nil && !errors.Is(err, types.ErrVehicleTypeNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check vehicle type uniqueness: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "vehicle type %s already exists", req.Name)
	}

	// Create vehicle type
	vehicleType, err := s.store.CreateVehicleType(ctx, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, types.ErrDuplicateEntry) {
			return nil, status.Errorf(codes.AlreadyExists, "vehicle type %s already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "failed to create vehicle type: %v", err)
	}

	return &genproto.CreateVehicleTypeResponse{
		VehicleType: vehicleType,
	}, nil
}

func (s *service) ListVehicleTypes(ctx context.Context, req *genproto.ListVehicleTypesRequest) (*genproto.ListVehicleTypesResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	vehicleTypes, nextPageToken, err := s.store.ListVehicleTypes(ctx, pageSize, req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list vehicle types: %v", err)
	}

	return &genproto.ListVehicleTypesResponse{
		VehicleTypes:  vehicleTypes,
		NextPageToken: nextPageToken,
	}, nil
}

// InitializeStandardVehicleTypes creates the standard vehicle types if they don't exist
func (s *service) InitializeStandardVehicleTypes(ctx context.Context) error {
	for _, stdType := range types.StandardVehicleTypes {
		_, err := s.store.GetVehicleTypeByName(ctx, stdType.Name)
		if errors.Is(err, types.ErrVehicleTypeNotFound) {
			// Create the standard type
			_, err := s.store.CreateVehicleType(ctx, stdType.Name, stdType.Description)
			if err != nil && !errors.Is(err, types.ErrDuplicateEntry) {
				return fmt.Errorf("failed to create standard vehicle type %s: %w", stdType.Name, err)
			}
			log.Printf("Created standard vehicle type: %s", stdType.Name)
		} else if err != nil {
			return fmt.Errorf("failed to check vehicle type %s: %w", stdType.Name, err)
		}
	}
	return nil
}