// services/staff/internal/types/types.go
package types

import (
	"context"
	"errors"

	"github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Business logic interface
type StaffService interface {
	// Driver CRUD operations
	CreateDriver(ctx context.Context, req *genproto.CreateDriverRequest) (*genproto.CreateDriverResponse, error)
	GetDriver(ctx context.Context, req *genproto.GetDriverRequest) (*genproto.GetDriverResponse, error)
	GetDriverByUserID(ctx context.Context, req *genproto.GetDriverByUserIDRequest) (*genproto.GetDriverResponse, error)
	ListDrivers(ctx context.Context, req *genproto.ListDriversRequest) (*genproto.ListDriversResponse, error)
	UpdateDriver(ctx context.Context, req *genproto.UpdateDriverRequest) (*genproto.UpdateDriverResponse, error)
	DeleteDriver(ctx context.Context, req *genproto.DeleteDriverRequest) error

	// Driver status management
	UpdateDriverStatus(ctx context.Context, req *genproto.UpdateDriverStatusRequest) (*genproto.UpdateDriverStatusResponse, error)
	GetActiveDrivers(ctx context.Context, req *genproto.GetActiveDriversRequest) (*genproto.ListDriversResponse, error)

	// Driver certification management
	AddDriverCertification(ctx context.Context, req *genproto.AddDriverCertificationRequest) (*genproto.AddDriverCertificationResponse, error)
	ListDriverCertifications(ctx context.Context, req *genproto.ListDriverCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error)
	UpdateCertification(ctx context.Context, req *genproto.UpdateCertificationRequest) (*genproto.UpdateCertificationResponse, error)
	DeleteCertification(ctx context.Context, req *genproto.DeleteCertificationRequest) error

	// Driver verification and compliance
	VerifyDriverLicense(ctx context.Context, req *genproto.VerifyDriverLicenseRequest) (*genproto.VerifyDriverLicenseResponse, error)
	GetExpiringLicenses(ctx context.Context, req *genproto.GetExpiringLicensesRequest) (*genproto.ListDriversResponse, error)
	GetExpiredCertifications(ctx context.Context, req *genproto.GetExpiredCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error)
}

// Data store interface
type StaffStore interface {
	// Driver CRUD
	CreateDriver(ctx context.Context, internalID uint64, externalID uuid.UUID, driver *DriverData) error
	GetDriverByID(ctx context.Context, externalID uuid.UUID) (*genproto.Driver, error)
	GetDriverByUserID(ctx context.Context, userID string) (*genproto.Driver, error)
	GetDriverByLicenseNumber(ctx context.Context, licenseNumber string) (*genproto.Driver, error)
	ListDrivers(ctx context.Context, params ListDriversParams) ([]*genproto.Driver, string, error)
	UpdateDriver(ctx context.Context, externalID uuid.UUID, updates DriverUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.Driver, error)
	DeleteDriver(ctx context.Context, externalID uuid.UUID) error

	// Driver status management
	UpdateDriverStatus(ctx context.Context, externalID uuid.UUID, status genproto.DriverStatus, reason string) (*genproto.Driver, error)
	GetActiveDrivers(ctx context.Context, params ListDriversParams) ([]*genproto.Driver, string, error)

	// Driver certification management
	AddDriverCertification(ctx context.Context, certID uint64, driverID uuid.UUID, cert *CertificationData) (*genproto.DriverCertification, error)
	GetDriverCertifications(ctx context.Context, driverID uuid.UUID, params ListCertificationsParams) ([]*genproto.DriverCertification, string, error)
	UpdateCertification(ctx context.Context, certID uint64, updates CertificationUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.DriverCertification, error)
	DeleteCertification(ctx context.Context, certID uint64) error

	// Compliance queries
	GetExpiringLicenses(ctx context.Context, daysAhead int32, params ListDriversParams) ([]*genproto.Driver, string, error)
	GetExpiredCertifications(ctx context.Context, expiredSinceDays *int32, params ListCertificationsParams) ([]*genproto.DriverCertification, string, error)
}

// DriverData represents the data needed to create a driver
type DriverData struct {
	UserID                 string
	LicenseNumber          string
	LicenseClass           genproto.LicenseClass
	LicenseExpiry          string // ISO date string
	ExperienceYears        int32
	PhoneNumber            string
	EmergencyContactName   string
	EmergencyContactPhone  string
	HireDate               *string // ISO date string, optional
}

// DriverUpdateFields represents fields that can be updated
type DriverUpdateFields struct {
	UserID                 *string
	LicenseNumber          *string
	LicenseClass           *genproto.LicenseClass
	LicenseExpiry          *string
	ExperienceYears        *int32
	PhoneNumber            *string
	EmergencyContactName   *string
	EmergencyContactPhone  *string
	HireDate               *string
}

// CertificationData represents certification information
type CertificationData struct {
	CertificationName string
	IssuedBy          string
	IssueDate         string // ISO date string
	ExpiryDate        string // ISO date string
}

// CertificationUpdateFields represents certification fields that can be updated
type CertificationUpdateFields struct {
	CertificationName *string
	IssuedBy          *string
	IssueDate         *string
	ExpiryDate        *string
}

// ListDriversParams encapsulates list parameters for drivers
type ListDriversParams struct {
	PageSize              int32
	PageToken             string
	StatusFilter          *genproto.DriverStatus
	LicenseClassFilter    *genproto.LicenseClass
	LicenseExpiringSoon   *bool
}

// ListCertificationsParams encapsulates list parameters for certifications
type ListCertificationsParams struct {
	PageSize      int32
	PageToken     string
	StatusFilter  *genproto.CertificationStatus
	ExpiringSoon  *bool
}

// Error types
var (
	ErrDriverNotFound        = errors.New("driver not found")
	ErrCertificationNotFound = errors.New("certification not found")
	ErrDuplicateEntry        = errors.New("duplicate entry")
	ErrInvalidStatus         = errors.New("invalid status transition")
	ErrDriverHasAssignments  = errors.New("driver has active vehicle assignments")
	ErrLicenseExpired        = errors.New("driver license is expired")
)

// Driver status transition rules
var ValidDriverStatusTransitions = map[genproto.DriverStatus][]genproto.DriverStatus{
	genproto.DriverStatus_PENDING_VERIFICATION: {
		genproto.DriverStatus_ACTIVE,
		genproto.DriverStatus_INACTIVE,
	},
	genproto.DriverStatus_ACTIVE: {
		genproto.DriverStatus_SUSPENDED,
		genproto.DriverStatus_INACTIVE,
	},
	genproto.DriverStatus_SUSPENDED: {
		genproto.DriverStatus_ACTIVE,
		genproto.DriverStatus_INACTIVE,
	},
	genproto.DriverStatus_INACTIVE: {
		genproto.DriverStatus_ACTIVE,
		genproto.DriverStatus_PENDING_VERIFICATION,
	},
}

// IsValidDriverStatusTransition checks if a status transition is allowed
func IsValidDriverStatusTransition(from, to genproto.DriverStatus) bool {
	allowedTransitions, exists := ValidDriverStatusTransitions[from]
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

// Common certification types for drivers
var StandardCertifications = []struct {
	Name     string
	IssuedBy string
}{
	{"Defensive Driving Certificate", "AA Kenya"},
	{"First Aid Certificate", "Kenya Red Cross Society"},
	{"Customer Service Training", "Kenya Association of Tour Operators"},
	{"Vehicle Maintenance Training", "Kenya Industrial Training Institute"},
	{"Road Safety Training", "National Transport and Safety Authority"},
	{"Commercial Vehicle Operation Certificate", "Ministry of Transport"},
}