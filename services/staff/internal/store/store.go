// services/staff/internal/store/store.go
package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/adammwaniki/bebabeba/services/staff/internal/types"
	"github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type store struct {
	db *sql.DB
}

// Returns a raw *sql.DB for use in migrations
func NewRawDB(cfg mysql.Config) (*sql.DB, error) {
	return sql.Open("mysql", cfg.FormatDSN())
}

// NewStore creates a new staff store
func NewStore(dsn string) (*store, error) {
	// Ensure conversion of DATETIME columns to Go's time.Time
	dsn += "?parseTime=true&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	return &store{db: db}, nil
}

// Driver operations

const createDriverQuery = `
INSERT INTO drivers (
	internal_id, external_id, user_id, license_number, license_class, license_expiry,
	experience_years, phone_number, emergency_contact_name, emergency_contact_phone,
	status, hire_date, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *store) CreateDriver(ctx context.Context, internalID uint64, externalID uuid.UUID, driver *types.DriverData) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			fmt.Printf("rollback failed: %v\n", rerr)
		}
	}()

	now := time.Now()

	// Parse license expiry date
	licenseExpiry, err := time.Parse("2006-01-02", driver.LicenseExpiry)
	if err != nil {
		return fmt.Errorf("invalid license expiry date: %w", err)
	}

	// Parse hire date if provided
	var hireDate sql.NullTime
	if driver.HireDate != nil {
		if parsed, err := time.Parse("2006-01-02", *driver.HireDate); err == nil {
			hireDate = sql.NullTime{Time: parsed, Valid: true}
		}
	} else {
		// Default to today if not provided
		hireDate = sql.NullTime{Time: now, Valid: true}
	}

	_, err = tx.ExecContext(ctx, createDriverQuery,
		internalID,
		externalID.Bytes(),
		driver.UserID,
		driver.LicenseNumber,
		driver.LicenseClass.String(),
		licenseExpiry,
		driver.ExperienceYears,
		driver.PhoneNumber,
		driver.EmergencyContactName,
		driver.EmergencyContactPhone,
		genproto.DriverStatus_PENDING_VERIFICATION.String(), // Default status
		hireDate,
		now,
		now,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return types.ErrDuplicateEntry
		}
		return fmt.Errorf("failed to insert driver: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

const getDriverByIDQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE external_id = ?
LIMIT 1`

func (s *store) GetDriverByID(ctx context.Context, externalID uuid.UUID) (*genproto.Driver, error) {
	driver, err := s.scanDriver(ctx, getDriverByIDQuery, externalID.Bytes())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to get driver by ID: %w", err)
	}
	return driver, nil
}

const getDriverByUserIDQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE user_id = ?
LIMIT 1`

func (s *store) GetDriverByUserID(ctx context.Context, userID string) (*genproto.Driver, error) {
	driver, err := s.scanDriver(ctx, getDriverByUserIDQuery, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to get driver by user ID: %w", err)
	}
	return driver, nil
}

const getDriverByLicenseQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE license_number = ?
LIMIT 1`

func (s *store) GetDriverByLicenseNumber(ctx context.Context, licenseNumber string) (*genproto.Driver, error) {
	driver, err := s.scanDriver(ctx, getDriverByLicenseQuery, licenseNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrDriverNotFound
		}
		return nil, fmt.Errorf("failed to get driver by license: %w", err)
	}
	return driver, nil
}

const listDriversQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE (?='' OR status = ?)
  AND (?='' OR license_class = ?)
  AND (? = 0 OR (? = 1 AND license_expiry BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 30 DAY)))
  AND (?='' OR created_at > ?)
ORDER BY created_at DESC
LIMIT ?`

func (s *store) ListDrivers(ctx context.Context, params types.ListDriversParams) ([]*genproto.Driver, string, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 50
	}

	// Parse page token
	var cursorTime time.Time
	if params.PageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(params.PageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := cursorTime.UnmarshalText(decoded); err != nil {
			return nil, "", fmt.Errorf("invalid page token format: %w", err)
		}
	}

	// Prepare filter parameters
	statusStr := ""
	if params.StatusFilter != nil {
		statusStr = params.StatusFilter.String()
	}

	licenseClassStr := ""
	if params.LicenseClassFilter != nil {
		licenseClassStr = params.LicenseClassFilter.String()
	}

	expiringSoon := 0
	if params.LicenseExpiringSoon != nil && *params.LicenseExpiringSoon {
		expiringSoon = 1
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, listDriversQuery,
		statusStr, statusStr,
		licenseClassStr, licenseClassStr,
		expiringSoon, expiringSoon,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list drivers: %w", err)
	}
	defer rows.Close()

	var drivers []*genproto.Driver
	var lastCreatedAt time.Time

	for rows.Next() {
		driver, err := s.scanDriverFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan driver: %w", err)
		}
		drivers = append(drivers, driver)
		lastCreatedAt = driver.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(drivers)) > params.PageSize {
		drivers = drivers[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return drivers, nextPageToken, nil
}

const updateDriverStatusQuery = `
UPDATE drivers 
SET status = ?, updated_at = ?
WHERE external_id = ?`

func (s *store) UpdateDriverStatus(ctx context.Context, externalID uuid.UUID, status genproto.DriverStatus, reason string) (*genproto.Driver, error) {
	result, err := s.db.ExecContext(ctx, updateDriverStatusQuery,
		status.String(),
		time.Now(),
		externalID.Bytes(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update driver status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil, types.ErrDriverNotFound
	}

	return s.GetDriverByID(ctx, externalID)
}

const getActiveDriversQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE status = 'ACTIVE'
  AND license_expiry > NOW()
  AND (?='' OR license_class = ?)
  AND (?='' OR created_at > ?)
ORDER BY created_at DESC
LIMIT ?`

func (s *store) GetActiveDrivers(ctx context.Context, params types.ListDriversParams) ([]*genproto.Driver, string, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 50
	}

	// Parse page token
	var cursorTime time.Time
	if params.PageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(params.PageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := cursorTime.UnmarshalText(decoded); err != nil {
			return nil, "", fmt.Errorf("invalid page token format: %w", err)
		}
	}

	licenseClassStr := ""
	if params.LicenseClassFilter != nil {
		licenseClassStr = params.LicenseClassFilter.String()
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, getActiveDriversQuery,
		licenseClassStr, licenseClassStr,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get active drivers: %w", err)
	}
	defer rows.Close()

	var drivers []*genproto.Driver
	var lastCreatedAt time.Time

	for rows.Next() {
		driver, err := s.scanDriverFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan driver: %w", err)
		}
		drivers = append(drivers, driver)
		lastCreatedAt = driver.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(drivers)) > params.PageSize {
		drivers = drivers[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return drivers, nextPageToken, nil
}

// Certification operations

const addCertificationQuery = `
INSERT INTO driver_certifications (
	id, driver_id, certification_name, issued_by, issue_date, expiry_date, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`

func (s *store) AddDriverCertification(ctx context.Context, certID uint64, driverID uuid.UUID, cert *types.CertificationData) (*genproto.DriverCertification, error) {
	now := time.Now()

	// Parse dates
	issueDate, err := time.Parse("2006-01-02", cert.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("invalid issue date: %w", err)
	}

	expiryDate, err := time.Parse("2006-01-02", cert.ExpiryDate)
	if err != nil {
		return nil, fmt.Errorf("invalid expiry date: %w", err)
	}

	_, err = s.db.ExecContext(ctx, addCertificationQuery,
		certID,
		driverID.Bytes(),
		cert.CertificationName,
		cert.IssuedBy,
		issueDate,
		expiryDate,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add certification: %w", err)
	}

	// Calculate computed fields
	isExpired := expiryDate.Before(now)
	daysUntilExpiry := int32(time.Until(expiryDate).Hours() / 24)

	return &genproto.DriverCertification{
		Id:                fmt.Sprintf("%d", certID),
		DriverId:          driverID.String(),
		CertificationName: cert.CertificationName,
		IssuedBy:          cert.IssuedBy,
		IssueDate:         timestamppb.New(issueDate),
		ExpiryDate:        timestamppb.New(expiryDate),
		Status:            genproto.CertificationStatus_CERT_ACTIVE,
		CreatedAt:         timestamppb.New(now),
		IsExpired:         isExpired,
		DaysUntilExpiry:   daysUntilExpiry,
	}, nil
}

// Helper functions

func (s *store) scanDriver(ctx context.Context, query string, args ...interface{}) (*genproto.Driver, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	return s.scanDriverFromRow(row)
}

func (s *store) scanDriverFromRow(row *sql.Row) (*genproto.Driver, error) {
	var driver genproto.Driver
	var statusStr, licenseClassStr string
	var licenseExpiry time.Time
	var hireDate sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&driver.Id,
		&driver.UserId,
		&driver.LicenseNumber,
		&licenseClassStr,
		&licenseExpiry,
		&driver.ExperienceYears,
		&driver.PhoneNumber,
		&driver.EmergencyContactName,
		&driver.EmergencyContactPhone,
		&statusStr,
		&hireDate,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateDriver(&driver, statusStr, licenseClassStr, licenseExpiry, hireDate, createdAt, updatedAt)
}

func (s *store) scanDriverFromRows(rows *sql.Rows) (*genproto.Driver, error) {
	var driver genproto.Driver
	var statusStr, licenseClassStr string
	var licenseExpiry time.Time
	var hireDate sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&driver.Id,
		&driver.UserId,
		&driver.LicenseNumber,
		&licenseClassStr,
		&licenseExpiry,
		&driver.ExperienceYears,
		&driver.PhoneNumber,
		&driver.EmergencyContactName,
		&driver.EmergencyContactPhone,
		&statusStr,
		&hireDate,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateDriver(&driver, statusStr, licenseClassStr, licenseExpiry, hireDate, createdAt, updatedAt)
}

func (s *store) populateDriver(driver *genproto.Driver, statusStr, licenseClassStr string, licenseExpiry time.Time, hireDate sql.NullTime, createdAt, updatedAt time.Time) (*genproto.Driver, error) {
	// Convert status string to enum
	statusVal, ok := genproto.DriverStatus_value[statusStr]
	if !ok {
		return nil, fmt.Errorf("invalid status value: %s", statusStr)
	}
	driver.Status = genproto.DriverStatus(statusVal)

	// Convert license class string to enum
	licenseClassVal, ok := genproto.LicenseClass_value[licenseClassStr]
	if !ok {
		return nil, fmt.Errorf("invalid license class value: %s", licenseClassStr)
	}
	driver.LicenseClass = genproto.LicenseClass(licenseClassVal)

	// Set license expiry and computed fields
	driver.LicenseExpiry = timestamppb.New(licenseExpiry)
	now := time.Now()
	driver.LicenseExpired = licenseExpiry.Before(now)
	driver.DaysUntilLicenseExpiry = int32(time.Until(licenseExpiry).Hours() / 24)

	// Set hire date if valid
	if hireDate.Valid {
		driver.HireDate = timestamppb.New(hireDate.Time)
	}

	// Set timestamps
	driver.CreatedAt = timestamppb.New(createdAt)
	driver.UpdatedAt = timestamppb.New(updatedAt)

	return driver, nil
}

// UpdateDriver updates driver information based on the provided field mask
const updateDriverQuery = `
UPDATE drivers 
SET user_id = CASE WHEN ? THEN ? ELSE user_id END,
    license_number = CASE WHEN ? THEN ? ELSE license_number END,
    license_class = CASE WHEN ? THEN ? ELSE license_class END,
    license_expiry = CASE WHEN ? THEN ? ELSE license_expiry END,
    experience_years = CASE WHEN ? THEN ? ELSE experience_years END,
    phone_number = CASE WHEN ? THEN ? ELSE phone_number END,
    emergency_contact_name = CASE WHEN ? THEN ? ELSE emergency_contact_name END,
    emergency_contact_phone = CASE WHEN ? THEN ? ELSE emergency_contact_phone END,
    hire_date = CASE WHEN ? THEN ? ELSE hire_date END,
    updated_at = ?
WHERE external_id = ?`

func (s *store) UpdateDriver(ctx context.Context, externalID uuid.UUID, updates types.DriverUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.Driver, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			fmt.Printf("rollback failed: %v\n", rerr)
		}
	}()

	now := time.Now()

	// Determine which fields to update
	updateUserID := false
	updateLicenseNumber := false
	updateLicenseClass := false
	updateLicenseExpiry := false
	updateExperienceYears := false
	updatePhoneNumber := false
	updateEmergencyContactName := false
	updateEmergencyContactPhone := false
	updateHireDate := false

	if updateMask != nil {
		for _, path := range updateMask.Paths {
			switch path {
			case "user_id":
				updateUserID = true
			case "license_number":
				updateLicenseNumber = true
			case "license_class":
				updateLicenseClass = true
			case "license_expiry":
				updateLicenseExpiry = true
			case "experience_years":
				updateExperienceYears = true
			case "phone_number":
				updatePhoneNumber = true
			case "emergency_contact_name":
				updateEmergencyContactName = true
			case "emergency_contact_phone":
				updateEmergencyContactPhone = true
			case "hire_date":
				updateHireDate = true
			}
		}
	} else {
		// Update all provided fields
		updateUserID = updates.UserID != nil
		updateLicenseNumber = updates.LicenseNumber != nil
		updateLicenseClass = updates.LicenseClass != nil
		updateLicenseExpiry = updates.LicenseExpiry != nil
		updateExperienceYears = updates.ExperienceYears != nil
		updatePhoneNumber = updates.PhoneNumber != nil
		updateEmergencyContactName = updates.EmergencyContactName != nil
		updateEmergencyContactPhone = updates.EmergencyContactPhone != nil
		updateHireDate = updates.HireDate != nil
	}

	// Prepare update values
	var userID, licenseNumber, licenseClass, phoneNumber, emergencyContactName, emergencyContactPhone string
	var experienceYears int32
	var licenseExpiry, hireDate sql.NullTime

	if updateUserID && updates.UserID != nil {
		userID = *updates.UserID
	}
	if updateLicenseNumber && updates.LicenseNumber != nil {
		licenseNumber = *updates.LicenseNumber
	}
	if updateLicenseClass && updates.LicenseClass != nil {
		licenseClass = updates.LicenseClass.String()
	}
	if updateLicenseExpiry && updates.LicenseExpiry != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.LicenseExpiry); err == nil {
			licenseExpiry = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if updateExperienceYears && updates.ExperienceYears != nil {
		experienceYears = *updates.ExperienceYears
	}
	if updatePhoneNumber && updates.PhoneNumber != nil {
		phoneNumber = *updates.PhoneNumber
	}
	if updateEmergencyContactName && updates.EmergencyContactName != nil {
		emergencyContactName = *updates.EmergencyContactName
	}
	if updateEmergencyContactPhone && updates.EmergencyContactPhone != nil {
		emergencyContactPhone = *updates.EmergencyContactPhone
	}
	if updateHireDate && updates.HireDate != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.HireDate); err == nil {
			hireDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	// Execute update
	result, err := tx.ExecContext(ctx, updateDriverQuery,
		updateUserID, userID,
		updateLicenseNumber, licenseNumber,
		updateLicenseClass, licenseClass,
		updateLicenseExpiry, licenseExpiry,
		updateExperienceYears, experienceYears,
		updatePhoneNumber, phoneNumber,
		updateEmergencyContactName, emergencyContactName,
		updateEmergencyContactPhone, emergencyContactPhone,
		updateHireDate, hireDate,
		now,
		externalID.Bytes(),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, types.ErrDuplicateEntry
		}
		return nil, fmt.Errorf("failed to update driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil, types.ErrDriverNotFound
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated driver
	return s.GetDriverByID(ctx, externalID)
}

// DeleteDriver performs a soft delete by setting status to INACTIVE
const softDeleteDriverQuery = `
UPDATE drivers 
SET status = 'INACTIVE', updated_at = ?
WHERE external_id = ? AND status != 'INACTIVE'`

func (s *store) DeleteDriver(ctx context.Context, externalID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, softDeleteDriverQuery,
		time.Now(),
		externalID.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete driver: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return types.ErrDriverNotFound
	}

	return nil
}

// GetDriverCertifications retrieves certifications for a specific driver
const getDriverCertificationsQuery = `
SELECT 
	id,
	LOWER(HEX(driver_id)) as driver_id,
	certification_name,
	issued_by,
	issue_date,
	expiry_date,
	status,
	created_at,
	updated_at
FROM driver_certifications
WHERE driver_id = ?
  AND (?='' OR status = ?)
  AND (? = 0 OR (? = 1 AND expiry_date BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 30 DAY)))
  AND (?='' OR created_at > ?)
ORDER BY created_at DESC
LIMIT ?`

func (s *store) GetDriverCertifications(ctx context.Context, driverID uuid.UUID, params types.ListCertificationsParams) ([]*genproto.DriverCertification, string, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 50
	}

	// Parse page token
	var cursorTime time.Time
	if params.PageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(params.PageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := cursorTime.UnmarshalText(decoded); err != nil {
			return nil, "", fmt.Errorf("invalid page token format: %w", err)
		}
	}

	// Prepare filter parameters
	statusStr := ""
	if params.StatusFilter != nil {
		statusStr = params.StatusFilter.String()
	}

	expiringSoon := 0
	if params.ExpiringSoon != nil && *params.ExpiringSoon {
		expiringSoon = 1
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, getDriverCertificationsQuery,
		driverID.Bytes(),
		statusStr, statusStr,
		expiringSoon, expiringSoon,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get driver certifications: %w", err)
	}
	defer rows.Close()

	var certifications []*genproto.DriverCertification
	var lastCreatedAt time.Time

	for rows.Next() {
		cert, err := s.scanCertificationFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan certification: %w", err)
		}
		certifications = append(certifications, cert)
		lastCreatedAt = cert.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(certifications)) > params.PageSize {
		certifications = certifications[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return certifications, nextPageToken, nil
}

// UpdateCertification updates certification information
const updateCertificationQuery = `
UPDATE driver_certifications 
SET certification_name = CASE WHEN ? THEN ? ELSE certification_name END,
    issued_by = CASE WHEN ? THEN ? ELSE issued_by END,
    issue_date = CASE WHEN ? THEN ? ELSE issue_date END,
    expiry_date = CASE WHEN ? THEN ? ELSE expiry_date END,
    updated_at = ?
WHERE id = ?`

func (s *store) UpdateCertification(ctx context.Context, certID uint64, updates types.CertificationUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.DriverCertification, error) {
	now := time.Now()

	// Determine which fields to update
	updateCertificationName := false
	updateIssuedBy := false
	updateIssueDate := false
	updateExpiryDate := false

	if updateMask != nil {
		for _, path := range updateMask.Paths {
			switch path {
			case "certification_name":
				updateCertificationName = true
			case "issued_by":
				updateIssuedBy = true
			case "issue_date":
				updateIssueDate = true
			case "expiry_date":
				updateExpiryDate = true
			}
		}
	} else {
		// Update all provided fields
		updateCertificationName = updates.CertificationName != nil
		updateIssuedBy = updates.IssuedBy != nil
		updateIssueDate = updates.IssueDate != nil
		updateExpiryDate = updates.ExpiryDate != nil
	}

	// Prepare update values
	var certificationName, issuedBy string
	var issueDate, expiryDate sql.NullTime

	if updateCertificationName && updates.CertificationName != nil {
		certificationName = *updates.CertificationName
	}
	if updateIssuedBy && updates.IssuedBy != nil {
		issuedBy = *updates.IssuedBy
	}
	if updateIssueDate && updates.IssueDate != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.IssueDate); err == nil {
			issueDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if updateExpiryDate && updates.ExpiryDate != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.ExpiryDate); err == nil {
			expiryDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	// Execute update
	result, err := s.db.ExecContext(ctx, updateCertificationQuery,
		updateCertificationName, certificationName,
		updateIssuedBy, issuedBy,
		updateIssueDate, issueDate,
		updateExpiryDate, expiryDate,
		now,
		certID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update certification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil, types.ErrCertificationNotFound
	}

	// Return updated certification
	return s.getCertificationByID(ctx, certID)
}

// DeleteCertification performs a soft delete by setting status to REVOKED
const softDeleteCertificationQuery = `
UPDATE driver_certifications 
SET status = 'CERT_REVOKED', updated_at = ?
WHERE id = ? AND status != 'CERT_REVOKED'`

func (s *store) DeleteCertification(ctx context.Context, certID uint64) error {
	result, err := s.db.ExecContext(ctx, softDeleteCertificationQuery,
		time.Now(),
		certID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete certification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return types.ErrCertificationNotFound
	}

	return nil
}

// GetExpiringLicenses retrieves drivers with licenses expiring within specified days
const getExpiringLicensesQuery = `
SELECT 
	LOWER(HEX(external_id)) as external_id,
	user_id,
	license_number,
	license_class,
	license_expiry,
	experience_years,
	phone_number,
	emergency_contact_name,
	emergency_contact_phone,
	status,
	hire_date,
	created_at,
	updated_at
FROM drivers
WHERE license_expiry BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL ? DAY)
  AND status = 'ACTIVE'
  AND (?='' OR created_at > ?)
ORDER BY license_expiry ASC, created_at DESC
LIMIT ?`

func (s *store) GetExpiringLicenses(ctx context.Context, daysAhead int32, params types.ListDriversParams) ([]*genproto.Driver, string, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 50
	}

	if daysAhead <= 0 {
		daysAhead = 30 // Default to 30 days
	}

	// Parse page token
	var cursorTime time.Time
	if params.PageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(params.PageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := cursorTime.UnmarshalText(decoded); err != nil {
			return nil, "", fmt.Errorf("invalid page token format: %w", err)
		}
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, getExpiringLicensesQuery,
		daysAhead,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get expiring licenses: %w", err)
	}
	defer rows.Close()

	var drivers []*genproto.Driver
	var lastCreatedAt time.Time

	for rows.Next() {
		driver, err := s.scanDriverFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan driver: %w", err)
		}
		drivers = append(drivers, driver)
		lastCreatedAt = driver.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(drivers)) > params.PageSize {
		drivers = drivers[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return drivers, nextPageToken, nil
}

// GetExpiredCertifications retrieves expired certifications
const getExpiredCertificationsQuery = `
SELECT 
	id,
	LOWER(HEX(driver_id)) as driver_id,
	certification_name,
	issued_by,
	issue_date,
	expiry_date,
	status,
	created_at,
	updated_at
FROM driver_certifications
WHERE expiry_date < NOW()
  AND (? = 0 OR expiry_date >= DATE_SUB(NOW(), INTERVAL ? DAY))
  AND status IN ('CERT_ACTIVE', 'CERT_EXPIRED')
  AND (?='' OR created_at > ?)
ORDER BY expiry_date DESC, created_at DESC
LIMIT ?`

func (s *store) GetExpiredCertifications(ctx context.Context, expiredSinceDays *int32, params types.ListCertificationsParams) ([]*genproto.DriverCertification, string, error) {
	if params.PageSize <= 0 || params.PageSize > 100 {
		params.PageSize = 50
	}

	expiredSince := int32(0)
	useExpiredSince := 0
	if expiredSinceDays != nil && *expiredSinceDays > 0 {
		expiredSince = *expiredSinceDays
		useExpiredSince = 1
	}

	// Parse page token
	var cursorTime time.Time
	if params.PageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(params.PageToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid page token: %w", err)
		}
		if err := cursorTime.UnmarshalText(decoded); err != nil {
			return nil, "", fmt.Errorf("invalid page token format: %w", err)
		}
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, getExpiredCertificationsQuery,
		useExpiredSince, expiredSince,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get expired certifications: %w", err)
	}
	defer rows.Close()

	var certifications []*genproto.DriverCertification
	var lastCreatedAt time.Time

	for rows.Next() {
		cert, err := s.scanCertificationFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan certification: %w", err)
		}
		certifications = append(certifications, cert)
		lastCreatedAt = cert.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(certifications)) > params.PageSize {
		certifications = certifications[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return certifications, nextPageToken, nil
}

// Helper methods for certifications

func (s *store) getCertificationByID(ctx context.Context, certID uint64) (*genproto.DriverCertification, error) {
	query := `
	SELECT 
		id,
		LOWER(HEX(driver_id)) as driver_id,
		certification_name,
		issued_by,
		issue_date,
		expiry_date,
		status,
		created_at,
		updated_at
	FROM driver_certifications
	WHERE id = ?
	LIMIT 1`

	row := s.db.QueryRowContext(ctx, query, certID)
	return s.scanCertificationFromRow(row)
}

func (s *store) scanCertificationFromRow(row *sql.Row) (*genproto.DriverCertification, error) {
	var cert genproto.DriverCertification
	var statusStr string
	var issueDate, expiryDate time.Time
	var createdAt, updatedAt sql.NullTime

	err := row.Scan(
		&cert.Id,
		&cert.DriverId,
		&cert.CertificationName,
		&cert.IssuedBy,
		&issueDate,
		&expiryDate,
		&statusStr,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateCertification(&cert, statusStr, issueDate, expiryDate, createdAt, updatedAt)
}

func (s *store) scanCertificationFromRows(rows *sql.Rows) (*genproto.DriverCertification, error) {
	var cert genproto.DriverCertification
	var statusStr string
	var issueDate, expiryDate time.Time
	var createdAt, updatedAt sql.NullTime

	err := rows.Scan(
		&cert.Id,
		&cert.DriverId,
		&cert.CertificationName,
		&cert.IssuedBy,
		&issueDate,
		&expiryDate,
		&statusStr,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateCertification(&cert, statusStr, issueDate, expiryDate, createdAt, updatedAt)
}

func (s *store) populateCertification(cert *genproto.DriverCertification, statusStr string, issueDate, expiryDate time.Time, createdAt, updatedAt sql.NullTime) (*genproto.DriverCertification, error) {
	// Convert status string to enum
	statusVal, ok := genproto.CertificationStatus_value[statusStr]
	if !ok {
		return nil, fmt.Errorf("invalid certification status value: %s", statusStr)
	}
	cert.Status = genproto.CertificationStatus(statusVal)

	// Set dates
	cert.IssueDate = timestamppb.New(issueDate)
	cert.ExpiryDate = timestamppb.New(expiryDate)

	// Calculate computed fields
	now := time.Now()
	cert.IsExpired = expiryDate.Before(now)
	cert.DaysUntilExpiry = int32(time.Until(expiryDate).Hours() / 24)

	// Set timestamps
	if createdAt.Valid {
		cert.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		cert.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	return cert, nil
}
