// services/vehicle/internal/store/store.go
package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/adammwaniki/bebabeba/services/vehicle/internal/types"
	"github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
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

// NewStore creates a new vehicle store
func NewStore(dsn string) (*store, error) {
	// Ensure conversion of DATETIME columns to Go's time.Time and local time zone
	dsn += "?parseTime=true&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	return &store{db: db}, nil
}

// Vehicle Type operations

const createVehicleTypeQuery = `
INSERT INTO vehicle_types (name, description, created_at) 
VALUES (?, ?, ?)`

func (s *store) CreateVehicleType(ctx context.Context, name, description string) (*genproto.VehicleType, error) {
	now := time.Now()
	
	result, err := s.db.ExecContext(ctx, createVehicleTypeQuery, name, description, now)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, types.ErrDuplicateEntry
		}
		return nil, fmt.Errorf("failed to create vehicle type: %w", err)
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get inserted ID: %w", err)
	}
	
	return &genproto.VehicleType{
		Id:          fmt.Sprintf("%d", id),
		Name:        name,
		Description: description,
		CreatedAt:   timestamppb.New(now),
	}, nil
}

const getVehicleTypeByIDQuery = `
SELECT id, name, description, created_at 
FROM vehicle_types 
WHERE id = ?`

func (s *store) GetVehicleTypeByID(ctx context.Context, typeID string) (*genproto.VehicleType, error) {
	var vehicleType genproto.VehicleType
	var createdAt time.Time
	
	err := s.db.QueryRowContext(ctx, getVehicleTypeByIDQuery, typeID).Scan(
		&vehicleType.Id,
		&vehicleType.Name,
		&vehicleType.Description,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrVehicleTypeNotFound
		}
		return nil, fmt.Errorf("failed to get vehicle type: %w", err)
	}
	
	vehicleType.CreatedAt = timestamppb.New(createdAt)
	return &vehicleType, nil
}

const getVehicleTypeByNameQuery = `
SELECT id, name, description, created_at 
FROM vehicle_types 
WHERE name = ?`

func (s *store) GetVehicleTypeByName(ctx context.Context, name string) (*genproto.VehicleType, error) {
	var vehicleType genproto.VehicleType
	var createdAt time.Time
	
	err := s.db.QueryRowContext(ctx, getVehicleTypeByNameQuery, name).Scan(
		&vehicleType.Id,
		&vehicleType.Name,
		&vehicleType.Description,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrVehicleTypeNotFound
		}
		return nil, fmt.Errorf("failed to get vehicle type by name: %w", err)
	}
	
	vehicleType.CreatedAt = timestamppb.New(createdAt)
	return &vehicleType, nil
}

const listVehicleTypesQuery = `
SELECT id, name, description, created_at 
FROM vehicle_types 
WHERE (?='' OR created_at > ?)
ORDER BY created_at DESC 
LIMIT ?`

func (s *store) ListVehicleTypes(ctx context.Context, pageSize int32, pageToken string) ([]*genproto.VehicleType, string, error) {
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 50
	}

	// Parse page token to get cursor timestamp
	var cursorTime time.Time
	if pageToken != "" {
		decoded, err := base64.URLEncoding.DecodeString(pageToken)
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

	rows, err := s.db.QueryContext(ctx, listVehicleTypesQuery,
		cursorStr, cursorStr,
		pageSize+1, // Fetch one extra to determine if there are more pages
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list vehicle types: %w", err)
	}
	defer rows.Close()

	var types []*genproto.VehicleType
	var lastCreatedAt time.Time

	for rows.Next() {
		var vehicleType genproto.VehicleType
		var createdAt time.Time

		err := rows.Scan(
			&vehicleType.Id,
			&vehicleType.Name,
			&vehicleType.Description,
			&createdAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan vehicle type: %w", err)
		}

		vehicleType.CreatedAt = timestamppb.New(createdAt)
		types = append(types, &vehicleType)
		lastCreatedAt = createdAt
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(types)) > pageSize {
		types = types[:pageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return types, nextPageToken, nil
}

// Vehicle operations

const createVehicleQuery = `
INSERT INTO vehicles (
	internal_id, external_id, vehicle_type_id, license_plate, make, model, year,
	color, seating_capacity, fuel_type, engine_number, chassis_number,
	registration_date, insurance_expiry, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *store) CreateVehicle(ctx context.Context, internalID uint64, externalID uuid.UUID, vehicle *types.VehicleData) error {
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

	// Convert optional string pointers to sql.NullString
	var engineNumber, chassisNumber sql.NullString
	var registrationDate, insuranceExpiry sql.NullTime

	if vehicle.EngineNumber != nil {
		engineNumber = sql.NullString{String: *vehicle.EngineNumber, Valid: true}
	}
	if vehicle.ChassisNumber != nil {
		chassisNumber = sql.NullString{String: *vehicle.ChassisNumber, Valid: true}
	}
	if vehicle.RegistrationDate != nil {
		if parsed, err := time.Parse("2006-01-02", *vehicle.RegistrationDate); err == nil {
			registrationDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if vehicle.InsuranceExpiry != nil {
		if parsed, err := time.Parse("2006-01-02", *vehicle.InsuranceExpiry); err == nil {
			insuranceExpiry = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	_, err = tx.ExecContext(ctx, createVehicleQuery,
		internalID,
		externalID.Bytes(),
		vehicle.VehicleTypeID,
		vehicle.LicensePlate,
		vehicle.Make,
		vehicle.Model,
		vehicle.Year,
		vehicle.Color,
		vehicle.SeatingCapacity,
		vehicle.FuelType.String(),
		engineNumber,
		chassisNumber,
		registrationDate,
		insuranceExpiry,
		genproto.VehicleStatus_ACTIVE.String(), // Default status
		now,
		now,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return types.ErrDuplicateEntry
		}
		return fmt.Errorf("failed to insert vehicle: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

const getVehicleByIDQuery = `
SELECT 
	LOWER(HEX(v.external_id)) as external_id,
	v.vehicle_type_id,
	vt.name as vehicle_type_name,
	v.license_plate,
	v.make,
	v.model,
	v.year,
	v.color,
	v.seating_capacity,
	v.fuel_type,
	v.engine_number,
	v.chassis_number,
	v.registration_date,
	v.insurance_expiry,
	v.status,
	v.created_at,
	v.updated_at
FROM vehicles v
INNER JOIN vehicle_types vt ON v.vehicle_type_id = vt.id
WHERE v.external_id = ?
LIMIT 1`

func (s *store) GetVehicleByID(ctx context.Context, externalID uuid.UUID) (*genproto.Vehicle, error) {
	vehicle, err := s.scanVehicle(ctx, getVehicleByIDQuery, externalID.Bytes())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("failed to get vehicle by ID: %w", err)
	}
	return vehicle, nil
}

const getVehicleByLicensePlateQuery = `
SELECT 
	LOWER(HEX(v.external_id)) as external_id,
	v.vehicle_type_id,
	vt.name as vehicle_type_name,
	v.license_plate,
	v.make,
	v.model,
	v.year,
	v.color,
	v.seating_capacity,
	v.fuel_type,
	v.engine_number,
	v.chassis_number,
	v.registration_date,
	v.insurance_expiry,
	v.status,
	v.created_at,
	v.updated_at
FROM vehicles v
INNER JOIN vehicle_types vt ON v.vehicle_type_id = vt.id
WHERE v.license_plate = ?
LIMIT 1`

func (s *store) GetVehicleByLicensePlate(ctx context.Context, licensePlate string) (*genproto.Vehicle, error) {
	vehicle, err := s.scanVehicle(ctx, getVehicleByLicensePlateQuery, licensePlate)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrVehicleNotFound
		}
		return nil, fmt.Errorf("failed to get vehicle by license plate: %w", err)
	}
	return vehicle, nil
}

const listVehiclesQuery = `
SELECT 
	LOWER(HEX(v.external_id)) as external_id,
	v.vehicle_type_id,
	vt.name as vehicle_type_name,
	v.license_plate,
	v.make,
	v.model,
	v.year,
	v.color,
	v.seating_capacity,
	v.fuel_type,
	v.engine_number,
	v.chassis_number,
	v.registration_date,
	v.insurance_expiry,
	v.status,
	v.created_at,
	v.updated_at
FROM vehicles v
INNER JOIN vehicle_types vt ON v.vehicle_type_id = vt.id
WHERE (?='' OR v.status = ?)
  AND (?='' OR v.vehicle_type_id = ?)
  AND (?='' OR v.make LIKE ?)
  AND (?='' OR v.created_at > ?)
ORDER BY v.created_at DESC
LIMIT ?`

func (s *store) ListVehicles(ctx context.Context, params types.ListVehiclesParams) ([]*genproto.Vehicle, string, error) {
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

	vehicleTypeStr := ""
	if params.VehicleTypeFilter != nil {
		vehicleTypeStr = *params.VehicleTypeFilter
	}

	makePattern := ""
	if params.MakeFilter != nil {
		makePattern = "%" + *params.MakeFilter + "%"
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, listVehiclesQuery,
		statusStr, statusStr,
		vehicleTypeStr, vehicleTypeStr,
		makePattern, makePattern,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*genproto.Vehicle
	var lastCreatedAt time.Time

	for rows.Next() {
		vehicle, err := s.scanVehicleFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan vehicle: %w", err)
		}
		vehicles = append(vehicles, vehicle)
		lastCreatedAt = vehicle.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(vehicles)) > params.PageSize {
		vehicles = vehicles[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return vehicles, nextPageToken, nil
}

const updateVehicleQuery = `
UPDATE vehicles 
SET vehicle_type_id = CASE WHEN ? THEN ? ELSE vehicle_type_id END,
    license_plate = CASE WHEN ? THEN ? ELSE license_plate END,
    make = CASE WHEN ? THEN ? ELSE make END,
    model = CASE WHEN ? THEN ? ELSE model END,
    year = CASE WHEN ? THEN ? ELSE year END,
    color = CASE WHEN ? THEN ? ELSE color END,
    seating_capacity = CASE WHEN ? THEN ? ELSE seating_capacity END,
    fuel_type = CASE WHEN ? THEN ? ELSE fuel_type END,
    engine_number = CASE WHEN ? THEN ? ELSE engine_number END,
    chassis_number = CASE WHEN ? THEN ? ELSE chassis_number END,
    registration_date = CASE WHEN ? THEN ? ELSE registration_date END,
    insurance_expiry = CASE WHEN ? THEN ? ELSE insurance_expiry END,
    updated_at = ?
WHERE external_id = ?`

func (s *store) UpdateVehicle(ctx context.Context, externalID uuid.UUID, updates types.VehicleUpdateFields, updateMask *fieldmaskpb.FieldMask) (*genproto.Vehicle, error) {
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
	updateVehicleTypeID := false
	updateLicensePlate := false
	updateMake := false
	updateModel := false
	updateYear := false
	updateColor := false
	updateSeatingCapacity := false
	updateFuelType := false
	updateEngineNumber := false
	updateChassisNumber := false
	updateRegistrationDate := false
	updateInsuranceExpiry := false

	if updateMask != nil {
		for _, path := range updateMask.Paths {
			switch path {
			case "vehicle_type_id":
				updateVehicleTypeID = true
			case "license_plate":
				updateLicensePlate = true
			case "make":
				updateMake = true
			case "model":
				updateModel = true
			case "year":
				updateYear = true
			case "color":
				updateColor = true
			case "seating_capacity":
				updateSeatingCapacity = true
			case "fuel_type":
				updateFuelType = true
			case "engine_number":
				updateEngineNumber = true
			case "chassis_number":
				updateChassisNumber = true
			case "registration_date":
				updateRegistrationDate = true
			case "insurance_expiry":
				updateInsuranceExpiry = true
			}
		}
	} else {
		// Update all provided fields
		updateVehicleTypeID = updates.VehicleTypeID != nil
		updateLicensePlate = updates.LicensePlate != nil
		updateMake = updates.Make != nil
		updateModel = updates.Model != nil
		updateYear = updates.Year != nil
		updateColor = updates.Color != nil
		updateSeatingCapacity = updates.SeatingCapacity != nil
		updateFuelType = updates.FuelType != nil
		updateEngineNumber = updates.EngineNumber != nil
		updateChassisNumber = updates.ChassisNumber != nil
		updateRegistrationDate = updates.RegistrationDate != nil
		updateInsuranceExpiry = updates.InsuranceExpiry != nil
	}

	// Prepare update values
	var vehicleTypeID, licensePlate, make, model, color, fuelType string
	var year, seatingCapacity int32
	var engineNumber, chassisNumber sql.NullString
	var registrationDate, insuranceExpiry sql.NullTime

	if updateVehicleTypeID && updates.VehicleTypeID != nil {
		vehicleTypeID = *updates.VehicleTypeID
	}
	if updateLicensePlate && updates.LicensePlate != nil {
		licensePlate = *updates.LicensePlate
	}
	if updateMake && updates.Make != nil {
		make = *updates.Make
	}
	if updateModel && updates.Model != nil {
		model = *updates.Model
	}
	if updateYear && updates.Year != nil {
		year = *updates.Year
	}
	if updateColor && updates.Color != nil {
		color = *updates.Color
	}
	if updateSeatingCapacity && updates.SeatingCapacity != nil {
		seatingCapacity = *updates.SeatingCapacity
	}
	if updateFuelType && updates.FuelType != nil {
		fuelType = updates.FuelType.String()
	}
	if updateEngineNumber && updates.EngineNumber != nil {
		engineNumber = sql.NullString{String: *updates.EngineNumber, Valid: true}
	}
	if updateChassisNumber && updates.ChassisNumber != nil {
		chassisNumber = sql.NullString{String: *updates.ChassisNumber, Valid: true}
	}
	if updateRegistrationDate && updates.RegistrationDate != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.RegistrationDate); err == nil {
			registrationDate = sql.NullTime{Time: parsed, Valid: true}
		}
	}
	if updateInsuranceExpiry && updates.InsuranceExpiry != nil {
		if parsed, err := time.Parse("2006-01-02", *updates.InsuranceExpiry); err == nil {
			insuranceExpiry = sql.NullTime{Time: parsed, Valid: true}
		}
	}

	// Execute update
	result, err := tx.ExecContext(ctx, updateVehicleQuery,
		updateVehicleTypeID, vehicleTypeID,
		updateLicensePlate, licensePlate,
		updateMake, make,
		updateModel, model,
		updateYear, year,
		updateColor, color,
		updateSeatingCapacity, seatingCapacity,
		updateFuelType, fuelType,
		updateEngineNumber, engineNumber,
		updateChassisNumber, chassisNumber,
		updateRegistrationDate, registrationDate,
		updateInsuranceExpiry, insuranceExpiry,
		now,
		externalID.Bytes(),
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, types.ErrDuplicateEntry
		}
		return nil, fmt.Errorf("failed to update vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil, types.ErrVehicleNotFound
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated vehicle
	return s.GetVehicleByID(ctx, externalID)
}

const updateVehicleStatusQuery = `
UPDATE vehicles 
SET status = ?, updated_at = ?
WHERE external_id = ?`

func (s *store) UpdateVehicleStatus(ctx context.Context, externalID uuid.UUID, status genproto.VehicleStatus) (*genproto.Vehicle, error) {
	result, err := s.db.ExecContext(ctx, updateVehicleStatusQuery,
		status.String(),
		time.Now(),
		externalID.Bytes(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update vehicle status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil, types.ErrVehicleNotFound
	}

	return s.GetVehicleByID(ctx, externalID)
}

const deleteVehicleQuery = `
UPDATE vehicles 
SET status = 'RETIRED', updated_at = ?
WHERE external_id = ? AND status != 'RETIRED'`

func (s *store) DeleteVehicle(ctx context.Context, externalID uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, deleteVehicleQuery,
		time.Now(),
		externalID.Bytes(),
	)
	if err != nil {
		return fmt.Errorf("failed to delete vehicle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return types.ErrVehicleNotFound
	}

	return nil
}

// Specialized queries

func (s *store) GetVehiclesByType(ctx context.Context, vehicleTypeID string, params types.ListVehiclesParams) ([]*genproto.Vehicle, string, error) {
	params.VehicleTypeFilter = &vehicleTypeID
	return s.ListVehicles(ctx, params)
}

const getAvailableVehiclesQuery = `
SELECT 
	LOWER(HEX(v.external_id)) as external_id,
	v.vehicle_type_id,
	vt.name as vehicle_type_name,
	v.license_plate,
	v.make,
	v.model,
	v.year,
	v.color,
	v.seating_capacity,
	v.fuel_type,
	v.engine_number,
	v.chassis_number,
	v.registration_date,
	v.insurance_expiry,
	v.status,
	v.created_at,
	v.updated_at
FROM vehicles v
INNER JOIN vehicle_types vt ON v.vehicle_type_id = vt.id
WHERE v.status = 'ACTIVE'
  AND (?='' OR v.vehicle_type_id = ?)
  AND (?='' OR v.created_at > ?)
ORDER BY v.created_at DESC
LIMIT ?`

func (s *store) GetAvailableVehicles(ctx context.Context, vehicleTypeID *string, params types.ListVehiclesParams) ([]*genproto.Vehicle, string, error) {
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

	vehicleTypeStr := ""
	if vehicleTypeID != nil {
		vehicleTypeStr = *vehicleTypeID
	}

	cursorStr := ""
	if !cursorTime.IsZero() {
		cursorStr = cursorTime.Format(time.RFC3339Nano)
	}

	rows, err := s.db.QueryContext(ctx, getAvailableVehiclesQuery,
		vehicleTypeStr, vehicleTypeStr,
		cursorStr, cursorStr,
		params.PageSize+1,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get available vehicles: %w", err)
	}
	defer rows.Close()

	var vehicles []*genproto.Vehicle
	var lastCreatedAt time.Time

	for rows.Next() {
		vehicle, err := s.scanVehicleFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan vehicle: %w", err)
		}
		vehicles = append(vehicles, vehicle)
		lastCreatedAt = vehicle.CreatedAt.AsTime()
	}

	// Determine next page token
	var nextPageToken string
	if int32(len(vehicles)) > params.PageSize {
		vehicles = vehicles[:params.PageSize]
		tokenBytes, err := lastCreatedAt.MarshalText()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create next page token: %w", err)
		}
		nextPageToken = base64.URLEncoding.EncodeToString(tokenBytes)
	}

	return vehicles, nextPageToken, nil
}

// Helper functions

func (s *store) scanVehicle(ctx context.Context, query string, args ...interface{}) (*genproto.Vehicle, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	return s.scanVehicleFromRow(row)
}

func (s *store) scanVehicleFromRow(row *sql.Row) (*genproto.Vehicle, error) {
	var vehicle genproto.Vehicle
	var statusStr, fuelTypeStr string
	var engineNumber, chassisNumber sql.NullString
	var registrationDate, insuranceExpiry sql.NullTime
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&vehicle.Id,
		&vehicle.VehicleTypeId,
		&vehicle.VehicleTypeName,
		&vehicle.LicensePlate,
		&vehicle.Make,
		&vehicle.Model,
		&vehicle.Year,
		&vehicle.Color,
		&vehicle.SeatingCapacity,
		&fuelTypeStr,
		&engineNumber,
		&chassisNumber,
		&registrationDate,
		&insuranceExpiry,
		&statusStr,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateVehicle(&vehicle, statusStr, fuelTypeStr, engineNumber, chassisNumber, registrationDate, insuranceExpiry, createdAt, updatedAt)
}

func (s *store) scanVehicleFromRows(rows *sql.Rows) (*genproto.Vehicle, error) {
	var vehicle genproto.Vehicle
	var statusStr, fuelTypeStr string
	var engineNumber, chassisNumber sql.NullString
	var registrationDate, insuranceExpiry sql.NullTime
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&vehicle.Id,
		&vehicle.VehicleTypeId,
		&vehicle.VehicleTypeName,
		&vehicle.LicensePlate,
		&vehicle.Make,
		&vehicle.Model,
		&vehicle.Year,
		&vehicle.Color,
		&vehicle.SeatingCapacity,
		&fuelTypeStr,
		&engineNumber,
		&chassisNumber,
		&registrationDate,
		&insuranceExpiry,
		&statusStr,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return s.populateVehicle(&vehicle, statusStr, fuelTypeStr, engineNumber, chassisNumber, registrationDate, insuranceExpiry, createdAt, updatedAt)
}

func (s *store) populateVehicle(vehicle *genproto.Vehicle, statusStr, fuelTypeStr string, engineNumber, chassisNumber sql.NullString, registrationDate, insuranceExpiry sql.NullTime, createdAt, updatedAt time.Time) (*genproto.Vehicle, error) {
	// Convert status string to enum
	statusVal, ok := genproto.VehicleStatus_value[statusStr]
	if !ok {
		return nil, fmt.Errorf("invalid status value: %s", statusStr)
	}
	vehicle.Status = genproto.VehicleStatus(statusVal)

	// Convert fuel type string to enum
	fuelTypeVal, ok := genproto.FuelType_value[fuelTypeStr]
	if !ok {
		return nil, fmt.Errorf("invalid fuel type value: %s", fuelTypeStr)
	}
	vehicle.FuelType = genproto.FuelType(fuelTypeVal)

	// Handle optional fields
	if engineNumber.Valid {
		vehicle.EngineNumber = engineNumber.String
	}
	if chassisNumber.Valid {
		vehicle.ChassisNumber = chassisNumber.String
	}
	if registrationDate.Valid {
		vehicle.RegistrationDate = timestamppb.New(registrationDate.Time)
	}
	if insuranceExpiry.Valid {
		vehicle.InsuranceExpiry = timestamppb.New(insuranceExpiry.Time)
	}

	// Set timestamps
	vehicle.CreatedAt = timestamppb.New(createdAt)
	vehicle.UpdatedAt = timestamppb.New(updatedAt)

	return vehicle, nil
}