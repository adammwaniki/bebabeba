// services/vehicle/internal/types/types.go
package types

import (
	"context"
	"errors"

	"github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Business logic interface
type VehicleService interface {
	// Vehicle CRUD operations
	CreateVehicle(ctx context.Context, req *genproto.CreateVehicleRequest) (*genproto.CreateVehicleResponse, error)
	GetVehicle(ctx context.Context, req *genproto.GetVehicleRequest) (*genproto.GetVehicleResponse, error)
	ListVehicles(ctx context.Context, req *genproto.ListVehiclesRequest) (*genproto.ListVehiclesResponse, error)
	UpdateVehicle(ctx context.Context, req *genproto.UpdateVehicleRequest) (*genproto.UpdateVehicleResponse, error)
	DeleteVehicle(ctx context.Context, req *genproto.DeleteVehicleRequest) error

	// Specialized queries
	GetVehiclesByType(ctx context.Context, req *genproto.GetVehiclesByTypeRequest) (*genproto.ListVehiclesResponse, error)
	GetAvailableVehicles(ctx context.Context, req *genproto.GetAvailableVehiclesRequest) (*genproto.ListVehiclesResponse, error)
	UpdateVehicleStatus(ctx context.Context, req *genproto.UpdateVehicleStatusRequest) (*genproto.UpdateVehicleStatusResponse, error)

	// Vehicle type management
	CreateVehicleType(ctx context.Context, req *genproto.CreateVehicleTypeRequest) (*genproto.CreateVehicleTypeResponse, error)
	ListVehicleTypes(ctx context.Context, req *genproto.ListVehicleTypesRequest) (*genproto.ListVehicleTypesResponse, error)
}

// Data store interface
type VehicleStore interface {
	// Vehicle CRUD
	CreateVehicle(ctx context.Context, internalID uint64, externalID uuid.UUID, vehicle *VehicleData) error
	GetVehicleByID(ctx context.Context, externalID uuid.UUID) (*genproto.Vehicle, error)
	GetVehicleByLicensePlate(ctx context.Context, licensePlate string) (*genproto.Vehicle, error)
	ListVehicles(ctx context.Context, params ListVehiclesParams) ([]*genproto.Vehicle, string, error)
	UpdateVehicle(ctx context.Context, externalID uuid.UUID, updates VehicleUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.Vehicle, error)
	DeleteVehicle(ctx context.Context, externalID uuid.UUID) error

	// Specialized queries
	GetVehiclesByType(ctx context.Context, vehicleTypeID string, params ListVehiclesParams) ([]*genproto.Vehicle, string, error)
	GetAvailableVehicles(ctx context.Context, vehicleTypeID *string, params ListVehiclesParams) ([]*genproto.Vehicle, string, error)
	UpdateVehicleStatus(ctx context.Context, externalID uuid.UUID, status genproto.VehicleStatus) (*genproto.Vehicle, error)

	// Vehicle type management
	CreateVehicleType(ctx context.Context, name, description string) (*genproto.VehicleType, error)
	GetVehicleTypeByID(ctx context.Context, typeID string) (*genproto.VehicleType, error)
	GetVehicleTypeByName(ctx context.Context, name string) (*genproto.VehicleType, error)
	ListVehicleTypes(ctx context.Context, pageSize int32, pageToken string) ([]*genproto.VehicleType, string, error)
}

// VehicleData represents the data needed to create a vehicle
type VehicleData struct {
	VehicleTypeID    string
	LicensePlate     string
	Make             string
	Model            string
	Year             int32
	Color            string
	SeatingCapacity  int32
	FuelType         genproto.FuelType
	EngineNumber     *string // Optional
	ChassisNumber    *string // Optional
	RegistrationDate *string // ISO date string, optional
	InsuranceExpiry  *string // ISO date string, optional
}

// VehicleUpdateFields represents fields that can be updated
type VehicleUpdateFields struct {
	VehicleTypeID    *string
	LicensePlate     *string
	Make             *string
	Model            *string
	Year             *int32
	Color            *string
	SeatingCapacity  *int32
	FuelType         *genproto.FuelType
	EngineNumber     *string
	ChassisNumber    *string
	RegistrationDate *string
	InsuranceExpiry  *string
}

// ListVehiclesParams encapsulates list parameters
type ListVehiclesParams struct {
	PageSize         int32
	PageToken        string
	StatusFilter     *genproto.VehicleStatus
	VehicleTypeFilter *string
	MakeFilter       *string
}

// Error types
var (
	ErrVehicleNotFound     = errors.New("vehicle not found")
	ErrDuplicateEntry      = errors.New("duplicate entry")
	ErrVehicleTypeNotFound = errors.New("vehicle type not found")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrVehicleInUse        = errors.New("vehicle is currently in use")
)

// Vehicle status transition rules
var ValidStatusTransitions = map[genproto.VehicleStatus][]genproto.VehicleStatus{
	genproto.VehicleStatus_ACTIVE: {
		genproto.VehicleStatus_ASSIGNED,
		genproto.VehicleStatus_MAINTENANCE,
		genproto.VehicleStatus_RETIRED,
	},
	genproto.VehicleStatus_ASSIGNED: {
		genproto.VehicleStatus_ACTIVE,
		genproto.VehicleStatus_MAINTENANCE,
	},
	genproto.VehicleStatus_MAINTENANCE: {
		genproto.VehicleStatus_ACTIVE,
		genproto.VehicleStatus_RETIRED,
	},
	genproto.VehicleStatus_RETIRED: {
		// No transitions from retired status
	},
}

// IsValidStatusTransition checks if a status transition is allowed
func IsValidStatusTransition(from, to genproto.VehicleStatus) bool {
	allowedTransitions, exists := ValidStatusTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}
	return false
}

// Common vehicle types for SACCO
var StandardVehicleTypes = []struct {
	Name        string
	Description string
}{
	{"cab", "Taxi cabs for individual passenger transport"},
	{"bus", "Large passenger buses for city-to-city routes"},
	{"matatu", "Shared taxis for local and regional routes"},
	{"bodaboda", "Motorcycle taxis for short distance transport"},
	{"truck", "Cargo vehicles for goods transport"},
	{"van", "Small passenger or cargo vans"},
	{"pickup", "Pickup trucks for light cargo transport"},
}