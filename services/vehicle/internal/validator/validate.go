// services/vehicle/internal/validator/validate.go
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Kenyan license plate patterns
var (
	// Standard format: KAA 123A or KAA 123AB
	kenyaLicensePlateRegex = regexp.MustCompile(`^K[A-Z]{2}\s\d{3}[A-Z]{1,2}$`)
	// Government vehicles: GK 123A or GK 123AB  
	govLicensePlateRegex = regexp.MustCompile(`^GK\s\d{3}[A-Z]{1,2}$`)
	// Diplomatic vehicles: CD 123A
	dipLicensePlateRegex = regexp.MustCompile(`^CD\s\d{3}[A-Z]$`)
	// Trailer format: different pattern
	trailerLicensePlateRegex = regexp.MustCompile(`^T[A-Z]{2}\s\d{3}[A-Z]$`)
)

// ValidateLicensePlate validates Kenyan vehicle license plates
func ValidateLicensePlate(field, licensePlate string) error {
	// Normalize whitespace and convert to uppercase
	plate := strings.ToUpper(strings.TrimSpace(licensePlate))
	
	if plate == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Check against Kenyan license plate patterns
	if kenyaLicensePlateRegex.MatchString(plate) ||
		govLicensePlateRegex.MatchString(plate) ||
		dipLicensePlateRegex.MatchString(plate) ||
		trailerLicensePlateRegex.MatchString(plate) {
		return nil
	}

	return ValidationError{
		Field:   field,
		Message: "invalid Kenyan license plate format (expected formats: KAA 123A, GK 123A, CD 123A, etc.)",
	}
}

// NormalizeLicensePlate standardizes license plate format
func NormalizeLicensePlate(licensePlate string) string {
	// Convert to uppercase and normalize spacing
	plate := strings.ToUpper(strings.TrimSpace(licensePlate))
	
	// Remove extra spaces and ensure single space between parts
	parts := strings.Fields(plate)
	if len(parts) == 2 {
		return parts[0] + " " + parts[1]
	}
	
	return plate
}

// ValidateVehicleMake validates vehicle manufacturer
func ValidateVehicleMake(field, make string) error {
	make = strings.TrimSpace(make)
	
	if make == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(make) < 2 || len(make) > 50 {
		return ValidationError{
			Field:   field,
			Message: "must be between 2 and 50 characters",
		}
	}

	// Check for valid characters (letters, numbers, spaces, hyphens)
	for _, char := range make {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && 
		   char != ' ' && char != '-' && char != '&' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateVehicleModel validates vehicle model
func ValidateVehicleModel(field, model string) error {
	model = strings.TrimSpace(model)
	
	if model == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(model) < 1 || len(model) > 50 {
		return ValidationError{
			Field:   field,
			Message: "must be between 1 and 50 characters",
		}
	}

	// Allow alphanumeric, spaces, hyphens, and common model characters
	for _, char := range model {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && 
		   char != ' ' && char != '-' && char != '.' && char != '/' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateVehicleYear validates manufacturing year
func ValidateVehicleYear(field string, year int32) error {
	currentYear := int32(time.Now().Year())
	
	// Reasonable range: 1900 to next year
	if year < 1900 || year > currentYear+1 {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be between 1900 and %d", currentYear+1),
		}
	}

	return nil
}

// ValidateSeatingCapacity validates passenger capacity
func ValidateSeatingCapacity(field string, capacity int32) error {
	if capacity < 1 || capacity > 100 {
		return ValidationError{
			Field:   field,
			Message: "must be between 1 and 100",
		}
	}

	return nil
}

// ValidateColor validates vehicle color
func ValidateColor(field, color string) error {
	color = strings.TrimSpace(color)
	
	if color == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(color) < 3 || len(color) > 30 {
		return ValidationError{
			Field:   field,
			Message: "must be between 3 and 30 characters",
		}
	}

	// Check for valid characters (letters, spaces, hyphens for multi-word colors)
	for _, char := range color {
		if !unicode.IsLetter(char) && char != ' ' && char != '-' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateEngineNumber validates engine number format
func ValidateEngineNumber(field, engineNumber string) error {
	engineNumber = strings.TrimSpace(engineNumber)
	
	if engineNumber == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(engineNumber) < 5 || len(engineNumber) > 100 {
		return ValidationError{
			Field:   field,
			Message: "must be between 5 and 100 characters",
		}
	}

	// Engine numbers are typically alphanumeric with limited special characters
	for _, char := range engineNumber {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && 
		   char != '-' && char != '/' && char != '*' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateChassisNumber validates chassis number format
func ValidateChassisNumber(field, chassisNumber string) error {
	chassisNumber = strings.TrimSpace(chassisNumber)
	
	if chassisNumber == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(chassisNumber) < 5 || len(chassisNumber) > 100 {
		return ValidationError{
			Field:   field,
			Message: "must be between 5 and 100 characters",
		}
	}

	// Chassis numbers are typically alphanumeric
	for _, char := range chassisNumber {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && 
		   char != '-' && char != '/' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateVehicleTypeID validates vehicle type reference
func ValidateVehicleTypeID(field, typeID string) error {
	typeID = strings.TrimSpace(typeID)
	
	if typeID == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(typeID) < 1 || len(typeID) > 10 {
		return ValidationError{
			Field:   field,
			Message: "must be between 1 and 10 characters",
		}
	}

	return nil
}

// ValidateVehicleTypeName validates vehicle type name
func ValidateVehicleTypeName(field, name string) error {
	name = strings.TrimSpace(name)
	
	if name == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(name) < 2 || len(name) > 50 {
		return ValidationError{
			Field:   field,
			Message: "must be between 2 and 50 characters",
		}
	}

	// Valid vehicle type names
	validTypes := map[string]bool{
		"cab":       true,
		"bus":       true,
		"matatu":    true,
		"bodaboda":  true,
		"truck":     true,
		"van":       true,
		"pickup":    true,
	}

	if !validTypes[strings.ToLower(name)] {
		return ValidationError{
			Field:   field,
			Message: "must be one of: cab, bus, matatu, bodaboda, truck, van, pickup",
		}
	}

	return nil
}

// NormalizeVehicleFields normalizes vehicle input fields
func NormalizeVehicleFields(input *genproto.VehicleInput) {
	if input == nil {
		return
	}

	input.LicensePlate = NormalizeLicensePlate(input.LicensePlate)
	input.Make = strings.TrimSpace(input.Make)
	input.Model = strings.TrimSpace(input.Model)
	input.Color = strings.TrimSpace(input.Color)
	input.EngineNumber = strings.ToUpper(strings.TrimSpace(input.EngineNumber))
	input.ChassisNumber = strings.ToUpper(strings.TrimSpace(input.ChassisNumber))
	input.VehicleTypeId = strings.TrimSpace(input.VehicleTypeId)
}

// ValidateCreateVehicleRequest validates vehicle creation request
func ValidateCreateVehicleRequest(req *genproto.CreateVehicleRequest) error {
	if req == nil {
		return ValidationError{Field: "request", Message: "cannot be nil"}
	}

	if req.Vehicle == nil {
		return ValidationError{Field: "vehicle", Message: "cannot be nil"}
	}

	// Normalize fields first
	NormalizeVehicleFields(req.Vehicle)

	vehicle := req.Vehicle

	// Validate required fields
	if err := ValidateVehicleTypeID("vehicle_type_id", vehicle.VehicleTypeId); err != nil {
		return err
	}

	if err := ValidateLicensePlate("license_plate", vehicle.LicensePlate); err != nil {
		return err
	}

	if err := ValidateVehicleMake("make", vehicle.Make); err != nil {
		return err
	}

	if err := ValidateVehicleModel("model", vehicle.Model); err != nil {
		return err
	}

	if err := ValidateVehicleYear("year", vehicle.Year); err != nil {
		return err
	}

	if err := ValidateColor("color", vehicle.Color); err != nil {
		return err
	}

	if err := ValidateSeatingCapacity("seating_capacity", vehicle.SeatingCapacity); err != nil {
		return err
	}

	// Validate fuel type
	if vehicle.FuelType == genproto.FuelType_FUEL_UNSPECIFIED {
		return ValidationError{
			Field:   "fuel_type",
			Message: "must be specified",
		}
	}

	// Validate optional fields if provided
	if vehicle.EngineNumber != "" {
		if err := ValidateEngineNumber("engine_number", vehicle.EngineNumber); err != nil {
			return err
		}
	}

	if vehicle.ChassisNumber != "" {
		if err := ValidateChassisNumber("chassis_number", vehicle.ChassisNumber); err != nil {
			return err
		}
	}

	// Validate dates if provided
	if vehicle.RegistrationDate != nil {
		regDate := vehicle.RegistrationDate.AsTime()
		now := time.Now()
		
		// Registration date should not be in the future
		if regDate.After(now) {
			return ValidationError{
				Field:   "registration_date",
				Message: "cannot be in the future",
			}
		}

		// Registration date should not be too old (reasonable limit)
		if regDate.Before(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)) {
			return ValidationError{
				Field:   "registration_date",
				Message: "cannot be before 1900",
			}
		}
	}

	if vehicle.InsuranceExpiry != nil {
		expiry := vehicle.InsuranceExpiry.AsTime()
		now := time.Now()
		
		// Insurance should not be expired by more than a reasonable period
		if expiry.Before(now.AddDate(-1, 0, 0)) {
			return ValidationError{
				Field:   "insurance_expiry",
				Message: "cannot be expired by more than 1 year",
			}
		}
	}

	return nil
}

// ValidateUpdateVehicleRequest validates vehicle update request
func ValidateUpdateVehicleRequest(req *genproto.UpdateVehicleRequest) error {
	if req == nil {
		return ValidationError{Field: "request", Message: "cannot be nil"}
	}

	if req.VehicleId == "" {
		return ValidationError{Field: "vehicle_id", Message: "cannot be empty"}
	}

	if req.Vehicle == nil {
		return ValidationError{Field: "vehicle", Message: "cannot be nil"}
	}

	// Normalize fields first
	NormalizeVehicleFields(req.Vehicle)

	vehicle := req.Vehicle

	// If update mask is provided, only validate specified fields
	if req.UpdateMask != nil {
		return validateMaskedFields(vehicle, req.UpdateMask)
	}

	// If no mask, validate all non-empty fields
	return validateAllProvidedFields(vehicle)
}

// validateMaskedFields validates only fields specified in the update mask
func validateMaskedFields(vehicle *genproto.VehicleInput, mask *fieldmaskpb.FieldMask) error {
	for _, path := range mask.Paths {
		switch path {
		case "vehicle_type_id":
			if vehicle.VehicleTypeId != "" {
				if err := ValidateVehicleTypeID("vehicle_type_id", vehicle.VehicleTypeId); err != nil {
					return err
				}
			}
		case "license_plate":
			if vehicle.LicensePlate != "" {
				if err := ValidateLicensePlate("license_plate", vehicle.LicensePlate); err != nil {
					return err
				}
			}
		case "make":
			if vehicle.Make != "" {
				if err := ValidateVehicleMake("make", vehicle.Make); err != nil {
					return err
				}
			}
		case "model":
			if vehicle.Model != "" {
				if err := ValidateVehicleModel("model", vehicle.Model); err != nil {
					return err
				}
			}
		case "year":
			if vehicle.Year != 0 {
				if err := ValidateVehicleYear("year", vehicle.Year); err != nil {
					return err
				}
			}
		case "color":
			if vehicle.Color != "" {
				if err := ValidateColor("color", vehicle.Color); err != nil {
					return err
				}
			}
		case "seating_capacity":
			if vehicle.SeatingCapacity != 0 {
				if err := ValidateSeatingCapacity("seating_capacity", vehicle.SeatingCapacity); err != nil {
					return err
				}
			}
		case "fuel_type":
			if vehicle.FuelType != genproto.FuelType_FUEL_UNSPECIFIED {
				// Fuel type validation is implicit - enum constraint handles it
			}
		case "engine_number":
			if vehicle.EngineNumber != "" {
				if err := ValidateEngineNumber("engine_number", vehicle.EngineNumber); err != nil {
					return err
				}
			}
		case "chassis_number":
			if vehicle.ChassisNumber != "" {
				if err := ValidateChassisNumber("chassis_number", vehicle.ChassisNumber); err != nil {
					return err
				}
			}
		default:
			return ValidationError{
				Field:   "update_mask",
				Message: fmt.Sprintf("unsupported field path: %s", path),
			}
		}
	}

	return nil
}

// validateAllProvidedFields validates all non-empty fields
func validateAllProvidedFields(vehicle *genproto.VehicleInput) error {
	if vehicle.VehicleTypeId != "" {
		if err := ValidateVehicleTypeID("vehicle_type_id", vehicle.VehicleTypeId); err != nil {
			return err
		}
	}

	if vehicle.LicensePlate != "" {
		if err := ValidateLicensePlate("license_plate", vehicle.LicensePlate); err != nil {
			return err
		}
	}

	if vehicle.Make != "" {
		if err := ValidateVehicleMake("make", vehicle.Make); err != nil {
			return err
		}
	}

	if vehicle.Model != "" {
		if err := ValidateVehicleModel("model", vehicle.Model); err != nil {
			return err
		}
	}

	if vehicle.Year != 0 {
		if err := ValidateVehicleYear("year", vehicle.Year); err != nil {
			return err
		}
	}

	if vehicle.Color != "" {
		if err := ValidateColor("color", vehicle.Color); err != nil {
			return err
		}
	}

	if vehicle.SeatingCapacity != 0 {
		if err := ValidateSeatingCapacity("seating_capacity", vehicle.SeatingCapacity); err != nil {
			return err
		}
	}

	if vehicle.EngineNumber != "" {
		if err := ValidateEngineNumber("engine_number", vehicle.EngineNumber); err != nil {
			return err
		}
	}

	if vehicle.ChassisNumber != "" {
		if err := ValidateChassisNumber("chassis_number", vehicle.ChassisNumber); err != nil {
			return err
		}
	}

	return nil
}

// ValidateVehicleStatus validates vehicle status values
func ValidateVehicleStatus(field string, status genproto.VehicleStatus) error {
	if status == genproto.VehicleStatus_STATUS_UNSPECIFIED {
		return ValidationError{
			Field:   field,
			Message: "must be specified",
		}
	}

	return nil
}