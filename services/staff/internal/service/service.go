// services/staff/internal/service/service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/adammwaniki/bebabeba/services/common/utils"
	"github.com/adammwaniki/bebabeba/services/staff/internal/types"
	"github.com/adammwaniki/bebabeba/services/staff/internal/validator"
	"github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"github.com/influxdata/influxdb/v2/pkg/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type service struct {
	store types.StaffStore
}

// NewService creates a new staff service instance
func NewService(store types.StaffStore) *service {
	return &service{store: store}
}

// Driver CRUD operations

func (s *service) CreateDriver(ctx context.Context, req *genproto.CreateDriverRequest) (*genproto.CreateDriverResponse, error) {
	// Validate the request
	if err := validator.ValidateCreateDriverRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	driver := req.Driver

	// Check for duplicate license number
	existing, err := s.store.GetDriverByLicenseNumber(ctx, driver.LicenseNumber)
	if err != nil && !errors.Is(err, types.ErrDriverNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check license uniqueness: %v", err)
	}
	if existing != nil {
		return nil, status.Errorf(codes.AlreadyExists, "driver with license number %s already exists", driver.LicenseNumber)
	}

	// Check for duplicate user ID
	existingByUser, err := s.store.GetDriverByUserID(ctx, driver.UserId)
	if err != nil && !errors.Is(err, types.ErrDriverNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to check user ID uniqueness: %v", err)
	}
	if existingByUser != nil {
		return nil, status.Errorf(codes.AlreadyExists, "driver profile already exists for user %s", driver.UserId)
	}

	// Check if license is expired
	licenseExpiry := driver.LicenseExpiry.AsTime()
	if licenseExpiry.Before(time.Now()) {
		return nil, status.Errorf(codes.InvalidArgument, "cannot create driver with expired license")
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

	// Prepare driver data
	driverData := &types.DriverData{
		UserID:                driver.UserId,
		LicenseNumber:         driver.LicenseNumber,
		LicenseClass:          driver.LicenseClass,
		LicenseExpiry:         licenseExpiry.Format("2006-01-02"),
		ExperienceYears:       driver.ExperienceYears,
		PhoneNumber:           driver.PhoneNumber,
		EmergencyContactName:  driver.EmergencyContactName,
		EmergencyContactPhone: driver.EmergencyContactPhone,
	}

	// Handle hire date
	if driver.HireDate != nil {
		hireDateStr := driver.HireDate.AsTime().Format("2006-01-02")
		driverData.HireDate = &hireDateStr
	}

	// Create driver in store
	if err := s.store.CreateDriver(ctx, internalID, externalID, driverData); err != nil {
		if errors.Is(err, types.ErrDuplicateEntry) {
			return nil, status.Errorf(codes.AlreadyExists, "driver with this license number or user ID already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create driver: %v", err)
	}

	// Retrieve the created driver
	createdDriver, err := s.store.GetDriverByID(ctx, externalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve created driver: %v", err)
	}

	log.Printf("Driver created successfully for user %s with license %s", driver.UserId, driver.LicenseNumber)

	return &genproto.CreateDriverResponse{
		Driver: createdDriver,
	}, nil
}

func (s *service) GetDriver(ctx context.Context, req *genproto.GetDriverRequest) (*genproto.GetDriverResponse, error) {
	if req.DriverId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver ID is required")
	}

	// Parse UUID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Get driver from store
	driver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get driver: %v", err)
	}

	return &genproto.GetDriverResponse{
		Driver: driver,
	}, nil
}

func (s *service) GetDriverByUserID(ctx context.Context, req *genproto.GetDriverByUserIDRequest) (*genproto.GetDriverResponse, error) {
	if req.UserId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "user ID is required")
	}

	// Get driver from store
	driver, err := s.store.GetDriverByUserID(ctx, req.UserId)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found for user")
		}
		return nil, status.Errorf(codes.Internal, "failed to get driver by user ID: %v", err)
	}

	return &genproto.GetDriverResponse{
		Driver: driver,
	}, nil
}

func (s *service) ListDrivers(ctx context.Context, req *genproto.ListDriversRequest) (*genproto.ListDriversResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Prepare parameters
	params := types.ListDriversParams{
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	}

	if req.StatusFilter != nil {
		params.StatusFilter = req.StatusFilter
	}
	if req.LicenseClassFilter != nil {
		params.LicenseClassFilter = req.LicenseClassFilter
	}
	if req.LicenseExpiringSoon != nil {
		params.LicenseExpiringSoon = req.LicenseExpiringSoon
	}

	// Get drivers from store
	drivers, nextPageToken, err := s.store.ListDrivers(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list drivers: %v", err)
	}

	return &genproto.ListDriversResponse{
		Drivers:       drivers,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(drivers)),
	}, nil
}

func (s *service) UpdateDriverStatus(ctx context.Context, req *genproto.UpdateDriverStatusRequest) (*genproto.UpdateDriverStatusResponse, error) {
	if req.DriverId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver ID is required")
	}

	// Validate status
	if err := validator.ValidateDriverStatus("status", req.Status); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Get current driver to check status transition
	currentDriver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get current driver: %v", err)
	}

	// Check if status transition is valid
	if !types.IsValidDriverStatusTransition(currentDriver.Status, req.Status) {
		return nil, status.Errorf(codes.InvalidArgument,
			"invalid status transition from %s to %s",
			currentDriver.Status.String(), req.Status.String())
	}

	// Business rule: Cannot activate driver with expired license
	if req.Status == genproto.DriverStatus_ACTIVE && currentDriver.LicenseExpired {
		return nil, status.Errorf(codes.FailedPrecondition, "cannot activate driver with expired license")
	}

	// Update status
	updatedDriver, err := s.store.UpdateDriverStatus(ctx, driverID, req.Status, req.Reason)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update driver status: %v", err)
	}

	log.Printf("Driver %s status updated from %s to %s. Reason: %s",
		req.DriverId, currentDriver.Status.String(), req.Status.String(), req.Reason)

	return &genproto.UpdateDriverStatusResponse{
		Driver: updatedDriver,
	}, nil
}

func (s *service) GetActiveDrivers(ctx context.Context, req *genproto.GetActiveDriversRequest) (*genproto.ListDriversResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListDriversParams{
		PageSize:           pageSize,
		PageToken:          req.GetPageToken(),
		LicenseClassFilter: req.LicenseClassFilter,
	}

	drivers, nextPageToken, err := s.store.GetActiveDrivers(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get active drivers: %v", err)
	}

	return &genproto.ListDriversResponse{
		Drivers:       drivers,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(drivers)),
	}, nil
}

// Driver certification management

func (s *service) AddDriverCertification(ctx context.Context, req *genproto.AddDriverCertificationRequest) (*genproto.AddDriverCertificationResponse, error) {
	// Validate the request
	if err := validator.ValidateAddCertificationRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Verify driver exists
	_, err = s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify driver: %v", err)
	}

	// Generate certification ID
	nodeID, err := utils.GetSnowflakeNodeID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get snowflake node ID: %v", err)
	}

	generator := snowflake.New(int(nodeID))
	certID := generator.Next()

	cert := req.Certification

	// Prepare certification data
	certData := &types.CertificationData{
		CertificationName: cert.CertificationName,
		IssuedBy:          cert.IssuedBy,
		IssueDate:         cert.IssueDate.AsTime().Format("2006-01-02"),
		ExpiryDate:        cert.ExpiryDate.AsTime().Format("2006-01-02"),
	}

	// Add certification
	certification, err := s.store.AddDriverCertification(ctx, certID, driverID, certData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add certification: %v", err)
	}

	log.Printf("Certification %s added for driver %s", cert.CertificationName, req.DriverId)

	return &genproto.AddDriverCertificationResponse{
		Certification: certification,
	}, nil
}

// Verification and compliance

func (s *service) VerifyDriverLicense(ctx context.Context, req *genproto.VerifyDriverLicenseRequest) (*genproto.VerifyDriverLicenseResponse, error) {
	if req.DriverId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver ID is required")
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Get driver
	driver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get driver: %v", err)
	}

	// Verify license number matches if provided
	if req.LicenseNumber != "" && driver.LicenseNumber != req.LicenseNumber {
		return &genproto.VerifyDriverLicenseResponse{
			IsValid:            false,
			IsExpired:          false,
			VerificationSource: "internal_check",
			VerifiedAt:         timestamppb.New(time.Now()),
			Notes:              "License number mismatch",
		}, nil
	}

	// Check if license is expired
	isExpired := driver.LicenseExpired

	// In a real implementation, this would integrate with external systems
	// like NTSA (National Transport and Safety Authority) in Kenya
	return &genproto.VerifyDriverLicenseResponse{
		IsValid:            !isExpired,
		IsExpired:          isExpired,
		VerificationSource: "internal_check",
		VerifiedAt:         timestamppb.New(time.Now()),
		Notes:              fmt.Sprintf("License status verified. Days until expiry: %d", driver.DaysUntilLicenseExpiry),
	}, nil
}

// UpdateDriver handles driver information updates
func (s *service) UpdateDriver(ctx context.Context, req *genproto.UpdateDriverRequest) (*genproto.UpdateDriverResponse, error) {
	// Validate the request
	if err := validator.ValidateUpdateDriverRequest(req); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Check if driver exists
	existingDriver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get existing driver: %v", err)
	}

	driver := req.Driver

	// Check license number uniqueness if being updated
	if driver.LicenseNumber != "" && driver.LicenseNumber != existingDriver.LicenseNumber {
		existing, err := s.store.GetDriverByLicenseNumber(ctx, driver.LicenseNumber)
		if err != nil && !errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.Internal, "failed to check license uniqueness: %v", err)
		}
		if existing != nil && existing.Id != existingDriver.Id {
			return nil, status.Errorf(codes.AlreadyExists, "driver with license number %s already exists", driver.LicenseNumber)
		}
	}

	// Check user ID uniqueness if being updated
	if driver.UserId != "" && driver.UserId != existingDriver.UserId {
		existing, err := s.store.GetDriverByUserID(ctx, driver.UserId)
		if err != nil && !errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.Internal, "failed to check user ID uniqueness: %v", err)
		}
		if existing != nil && existing.Id != existingDriver.Id {
			return nil, status.Errorf(codes.AlreadyExists, "driver profile already exists for user %s", driver.UserId)
		}
	}

	// Prepare update fields
	updates := types.DriverUpdateFields{}
	if driver.UserId != "" {
		updates.UserID = &driver.UserId
	}
	if driver.LicenseNumber != "" {
		updates.LicenseNumber = &driver.LicenseNumber
	}
	if driver.LicenseClass != genproto.LicenseClass_LICENSE_UNSPECIFIED {
		updates.LicenseClass = &driver.LicenseClass
	}
	if driver.LicenseExpiry != nil {
		expiryStr := driver.LicenseExpiry.AsTime().Format("2006-01-02")
		updates.LicenseExpiry = &expiryStr
	}
	if driver.ExperienceYears != 0 {
		updates.ExperienceYears = &driver.ExperienceYears
	}
	if driver.PhoneNumber != "" {
		updates.PhoneNumber = &driver.PhoneNumber
	}
	if driver.EmergencyContactName != "" {
		updates.EmergencyContactName = &driver.EmergencyContactName
	}
	if driver.EmergencyContactPhone != "" {
		updates.EmergencyContactPhone = &driver.EmergencyContactPhone
	}
	if driver.HireDate != nil {
		hireDateStr := driver.HireDate.AsTime().Format("2006-01-02")
		updates.HireDate = &hireDateStr
	}

	// Update driver in store
	updatedDriver, err := s.store.UpdateDriver(ctx, driverID, updates, req.UpdateMask)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		if errors.Is(err, types.ErrDuplicateEntry) {
			return nil, status.Errorf(codes.AlreadyExists, "duplicate license number or user ID")
		}
		return nil, status.Errorf(codes.Internal, "failed to update driver: %v", err)
	}

	return &genproto.UpdateDriverResponse{
		Driver: updatedDriver,
	}, nil
}

// DeleteDriver handles driver profile deletion (soft delete)
func (s *service) DeleteDriver(ctx context.Context, req *genproto.DeleteDriverRequest) error {
	if req.DriverId == "" {
		return status.Errorf(codes.InvalidArgument, "driver ID is required")
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Check if driver exists and get current status
	existingDriver, err := s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return status.Errorf(codes.NotFound, "driver not found")
		}
		return status.Errorf(codes.Internal, "failed to get driver: %v", err)
	}

	// Business rule: Cannot delete active drivers with recent activity
	// This would be expanded to check for active vehicle assignments, recent trips, etc.
	if existingDriver.Status == genproto.DriverStatus_ACTIVE {
		log.Printf("Warning: Attempting to delete active driver %s", req.DriverId)
		// In a real system, we'd check for active assignments, ongoing trips, etc.
	}

	// Soft delete by setting status to INACTIVE
	if err := s.store.DeleteDriver(ctx, driverID); err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return status.Errorf(codes.NotFound, "driver not found")
		}
		return status.Errorf(codes.Internal, "failed to delete driver: %v", err)
	}

	log.Printf("Driver %s marked as inactive (soft deleted)", req.DriverId)
	return nil
}

// ListDriverCertifications handles listing certifications for a driver
func (s *service) ListDriverCertifications(ctx context.Context, req *genproto.ListDriverCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error) {
	if req.DriverId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "driver ID is required")
	}

	// Parse driver ID
	driverID, err := uuid.FromString(req.DriverId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid driver ID format: %v", err)
	}

	// Verify driver exists
	_, err = s.store.GetDriverByID(ctx, driverID)
	if err != nil {
		if errors.Is(err, types.ErrDriverNotFound) {
			return nil, status.Errorf(codes.NotFound, "driver not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to verify driver: %v", err)
	}

	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListCertificationsParams{
		PageSize:     pageSize,
		PageToken:    req.GetPageToken(),
		StatusFilter: req.StatusFilter,
		ExpiringSoon: req.ExpiringSoon,
	}

	certifications, nextPageToken, err := s.store.GetDriverCertifications(ctx, driverID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list driver certifications: %v", err)
	}

	return &genproto.ListDriverCertificationsResponse{
		Certifications: certifications,
		NextPageToken:  nextPageToken,
	}, nil
}

// UpdateCertification handles certification updates
func (s *service) UpdateCertification(ctx context.Context, req *genproto.UpdateCertificationRequest) (*genproto.UpdateCertificationResponse, error) {
	if req.CertificationId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "certification ID is required")
	}

	if req.Certification == nil {
		return nil, status.Errorf(codes.InvalidArgument, "certification data is required")
	}

	// Parse certification ID
	certIDStr := req.CertificationId
	certID, err := strconv.ParseUint(certIDStr, 10, 64)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid certification ID format: %v", err)
	}

	cert := req.Certification

	// Validate certification data if provided
	if cert.CertificationName != "" {
		if err := validator.ValidateCertificationName("certification_name", cert.CertificationName); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}

	if cert.IssuedBy != "" {
		if err := validator.ValidateIssuingAuthority("issued_by", cert.IssuedBy); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}

	// Validate dates if provided
	if cert.IssueDate != nil && cert.ExpiryDate != nil {
		issueDate := cert.IssueDate.AsTime()
		expiryDate := cert.ExpiryDate.AsTime()

		if expiryDate.Before(issueDate) {
			return nil, status.Errorf(codes.InvalidArgument, "expiry date must be after issue date")
		}
	}

	// Prepare update fields
	updates := types.CertificationUpdateFields{}
	if cert.CertificationName != "" {
		updates.CertificationName = &cert.CertificationName
	}
	if cert.IssuedBy != "" {
		updates.IssuedBy = &cert.IssuedBy
	}
	if cert.IssueDate != nil {
		issueDateStr := cert.IssueDate.AsTime().Format("2006-01-02")
		updates.IssueDate = &issueDateStr
	}
	if cert.ExpiryDate != nil {
		expiryDateStr := cert.ExpiryDate.AsTime().Format("2006-01-02")
		updates.ExpiryDate = &expiryDateStr
	}

	// Update certification
	updatedCert, err := s.store.UpdateCertification(ctx, certID, updates, req.UpdateMask)
	if err != nil {
		if errors.Is(err, types.ErrCertificationNotFound) {
			return nil, status.Errorf(codes.NotFound, "certification not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to update certification: %v", err)
	}

	return &genproto.UpdateCertificationResponse{
		Certification: updatedCert,
	}, nil
}

// DeleteCertification handles certification deletion (soft delete)
func (s *service) DeleteCertification(ctx context.Context, req *genproto.DeleteCertificationRequest) error {
	if req.CertificationId == "" {
		return status.Errorf(codes.InvalidArgument, "certification ID is required")
	}

	// Parse certification ID
	certIDStr := req.CertificationId
	certID, err := strconv.ParseUint(certIDStr, 10, 64)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid certification ID format: %v", err)
	}

	// Soft delete certification
	if err := s.store.DeleteCertification(ctx, certID); err != nil {
		if errors.Is(err, types.ErrCertificationNotFound) {
			return status.Errorf(codes.NotFound, "certification not found")
		}
		return status.Errorf(codes.Internal, "failed to delete certification: %v", err)
	}

	log.Printf("Certification %s marked as revoked (soft deleted)", req.CertificationId)
	return nil
}

// GetExpiringLicenses handles getting drivers with expiring licenses
func (s *service) GetExpiringLicenses(ctx context.Context, req *genproto.GetExpiringLicensesRequest) (*genproto.ListDriversResponse, error) {
	daysAhead := req.GetDaysAhead()
	if daysAhead <= 0 {
		daysAhead = 30 // Default to 30 days
	}

	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListDriversParams{
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	}

	drivers, nextPageToken, err := s.store.GetExpiringLicenses(ctx, daysAhead, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get expiring licenses: %v", err)
	}

	return &genproto.ListDriversResponse{
		Drivers:       drivers,
		NextPageToken: nextPageToken,
		TotalCount:    int32(len(drivers)),
	}, nil
}

// GetExpiredCertifications handles getting expired certifications
func (s *service) GetExpiredCertifications(ctx context.Context, req *genproto.GetExpiredCertificationsRequest) (*genproto.ListDriverCertificationsResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 100 {
		pageSize = 100
	}

	params := types.ListCertificationsParams{
		PageSize:  pageSize,
		PageToken: req.GetPageToken(),
	}

	certifications, nextPageToken, err := s.store.GetExpiredCertifications(ctx, req.ExpiredSinceDays, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get expired certifications: %v", err)
	}

	return &genproto.ListDriverCertificationsResponse{
		Certifications: certifications,
		NextPageToken:  nextPageToken,
	}, nil
}
