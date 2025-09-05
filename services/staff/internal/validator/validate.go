// services/staff/internal/validator/validate.go
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Kenyan driving license patterns
var (
	// Modern format: DL followed by numbers/letters
	kenyaLicenseRegex = regexp.MustCompile(`^DL\d{7,10}[A-Z]*$`)
	// Legacy format: numbers followed by letters
	legacyLicenseRegex = regexp.MustCompile(`^\d{6,8}[A-Z]{1,3}$`)
)

// ValidateKenyanLicense validates Kenyan driving license format
func ValidateKenyanLicense(field, licenseNumber string) error {
	license := strings.ToUpper(strings.TrimSpace(licenseNumber))
	
	if license == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Check against Kenyan license patterns
	if kenyaLicenseRegex.MatchString(license) || legacyLicenseRegex.MatchString(license) {
		return nil
	}

	return ValidationError{
		Field:   field,
		Message: "invalid Kenyan driving license format (expected: DL1234567 or 123456A)",
	}
}

// NormalizeLicense standardizes license number format
func NormalizeLicense(licenseNumber string) string {
	return strings.ToUpper(strings.TrimSpace(licenseNumber))
}

// ValidatePhoneNumber validates Kenyan phone numbers
func ValidatePhoneNumber(field, phoneNumber string) error {
	phone := strings.TrimSpace(phoneNumber)
	
	if phone == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Remove common prefixes and spaces
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Kenyan phone number patterns
	kenyaPatterns := []string{
		`^254[17]\d{8}$`,     // Full international format (254 country code)
		`^\+254[17]\d{8}$`,   // International with + prefix
		`^0[17]\d{8}$`,       // Local format starting with 0
		`^[17]\d{8}$`,        // Without leading 0
	}

	for _, pattern := range kenyaPatterns {
		if matched, _ := regexp.MatchString(pattern, phone); matched {
			return nil
		}
	}

	return ValidationError{
		Field:   field,
		Message: "invalid Kenyan phone number format (expected: 0712345678, 254712345678, etc.)",
	}
}

// NormalizePhoneNumber standardizes phone number to international format
func NormalizePhoneNumber(phoneNumber string) string {
	phone := strings.TrimSpace(phoneNumber)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")

	// Convert to international format (254XXXXXXXXX)
	if strings.HasPrefix(phone, "+254") {
		return phone[1:] // Remove +
	} else if strings.HasPrefix(phone, "254") {
		return phone // Already in correct format
	} else if strings.HasPrefix(phone, "0") {
		return "254" + phone[1:] // Replace leading 0 with 254
	} else if len(phone) == 9 && (strings.HasPrefix(phone, "7") || strings.HasPrefix(phone, "1")) {
		return "254" + phone // Add country code
	}

	return phone // Return as-is if no pattern matches
}

// ValidateExperienceYears validates driver experience
func ValidateExperienceYears(field string, years int32) error {
	if years < 0 {
		return ValidationError{
			Field:   field,
			Message: "cannot be negative",
		}
	}

	if years > 60 {
		return ValidationError{
			Field:   field,
			Message: "cannot exceed 60 years",
		}
	}

	return nil
}

// ValidateEmergencyContact validates emergency contact information
func ValidateEmergencyContact(nameField, phoneField, name, phone string) error {
	if name == "" {
		return ValidationError{
			Field:   nameField,
			Message: "emergency contact name cannot be empty",
		}
	}

	if len(name) < 2 || len(name) > 100 {
		return ValidationError{
			Field:   nameField,
			Message: "emergency contact name must be between 2 and 100 characters",
		}
	}

	// Validate name characters
	for _, char := range name {
		if !unicode.IsLetter(char) && char != ' ' && char != '-' && char != '\'' && char != '.' {
			return ValidationError{
				Field:   nameField,
				Message: fmt.Sprintf("emergency contact name contains invalid character: %q", char),
			}
		}
	}

	return ValidatePhoneNumber(phoneField, phone)
}

// ValidateCertificationName validates certification names
func ValidateCertificationName(field, name string) error {
	name = strings.TrimSpace(name)
	
	if name == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(name) < 3 || len(name) > 100 {
		return ValidationError{
			Field:   field,
			Message: "must be between 3 and 100 characters",
		}
	}

	// Allow letters, numbers, spaces, and common certification characters
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && 
		   char != ' ' && char != '-' && char != '.' && char != '(' && char != ')' {
			return ValidationError{
				Field:   field,
				Message: fmt.Sprintf("contains invalid character: %q", char),
			}
		}
	}

	return nil
}

// ValidateIssuingAuthority validates certification issuing authority
func ValidateIssuingAuthority(field, authority string) error {
	authority = strings.TrimSpace(authority)
	
	if authority == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	if len(authority) < 2 || len(authority) > 100 {
		return ValidationError{
			Field:   field,
			Message: "must be between 2 and 100 characters",
		}
	}

	return nil
}

// ValidateUserID validates user ID reference
func ValidateUserID(field, userID string) error {
	userID = strings.TrimSpace(userID)
	
	if userID == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Basic UUID format validation (36 characters with hyphens)
	if len(userID) != 36 {
		return ValidationError{
			Field:   field,
			Message: "invalid user ID format",
		}
	}

	return nil
}

// ValidateLicenseClass validates driving license class
func ValidateLicenseClass(field string, licenseClass genproto.LicenseClass) error {
	if licenseClass == genproto.LicenseClass_LICENSE_UNSPECIFIED {
		return ValidationError{
			Field:   field,
			Message: "license class must be specified",
		}
	}

	return nil
}

// ValidateDriverStatus validates driver status
func ValidateDriverStatus(field string, status genproto.DriverStatus) error {
	if status == genproto.DriverStatus_STATUS_UNSPECIFIED {
		return ValidationError{
			Field:   field,
			Message: "driver status must be specified",
		}
	}

	return nil
}

// ValidateLicenseExpiry validates license expiry date
func ValidateLicenseExpiry(field string, expiry time.Time) error {
	now := time.Now()

	// License shouldn't be expired by more than 1 year (grace period)
	if expiry.Before(now.AddDate(-1, 0, 0)) {
		return ValidationError{
			Field:   field,
			Message: "license expired more than 1 year ago",
		}
	}

	// License expiry shouldn't be more than 10 years in the future
	if expiry.After(now.AddDate(10, 0, 0)) {
		return ValidationError{
			Field:   field,
			Message: "license expiry cannot be more than 10 years in the future",
		}
	}

	return nil
}

// ValidateHireDate validates driver hire date
func ValidateHireDate(field string, hireDate time.Time) error {
	now := time.Now()

	// Hire date cannot be in the future
	if hireDate.After(now) {
		return ValidationError{
			Field:   field,
			Message: "hire date cannot be in the future",
		}
	}

	// Hire date cannot be more than 50 years ago (reasonable limit)
	if hireDate.Before(now.AddDate(-50, 0, 0)) {
		return ValidationError{
			Field:   field,
			Message: "hire date cannot be more than 50 years ago",
		}
	}

	return nil
}

// NormalizeDriverFields normalizes driver input fields
func NormalizeDriverFields(input *genproto.DriverInput) {
	if input == nil {
		return
	}

	input.LicenseNumber = NormalizeLicense(input.LicenseNumber)
	input.PhoneNumber = NormalizePhoneNumber(input.PhoneNumber)
	input.EmergencyContactName = strings.TrimSpace(input.EmergencyContactName)
	input.EmergencyContactPhone = NormalizePhoneNumber(input.EmergencyContactPhone)
	input.UserId = strings.TrimSpace(input.UserId)
}

// ValidateCreateDriverRequest validates driver creation request
func ValidateCreateDriverRequest(req *genproto.CreateDriverRequest) error {
	if req == nil {
		return ValidationError{Field: "request", Message: "cannot be nil"}
	}

	if req.Driver == nil {
		return ValidationError{Field: "driver", Message: "cannot be nil"}
	}

	// Normalize fields first
	NormalizeDriverFields(req.Driver)

	driver := req.Driver

	// Validate required fields
	if err := ValidateUserID("user_id", driver.UserId); err != nil {
		return err
	}

	if err := ValidateKenyanLicense("license_number", driver.LicenseNumber); err != nil {
		return err
	}

	if err := ValidateLicenseClass("license_class", driver.LicenseClass); err != nil {
		return err
	}

	// Validate license expiry
	if driver.LicenseExpiry != nil {
		if err := ValidateLicenseExpiry("license_expiry", driver.LicenseExpiry.AsTime()); err != nil {
			return err
		}
	} else {
		return ValidationError{
			Field:   "license_expiry",
			Message: "is required",
		}
	}

	if err := ValidateExperienceYears("experience_years", driver.ExperienceYears); err != nil {
		return err
	}

	if err := ValidatePhoneNumber("phone_number", driver.PhoneNumber); err != nil {
		return err
	}

	if err := ValidateEmergencyContact("emergency_contact_name", "emergency_contact_phone", 
		driver.EmergencyContactName, driver.EmergencyContactPhone); err != nil {
		return err
	}

	// Validate hire date if provided
	if driver.HireDate != nil {
		if err := ValidateHireDate("hire_date", driver.HireDate.AsTime()); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdateDriverRequest validates driver update request
func ValidateUpdateDriverRequest(req *genproto.UpdateDriverRequest) error {
	if req == nil {
		return ValidationError{Field: "request", Message: "cannot be nil"}
	}

	if req.DriverId == "" {
		return ValidationError{Field: "driver_id", Message: "cannot be empty"}
	}

	if req.Driver == nil {
		return ValidationError{Field: "driver", Message: "cannot be nil"}
	}

	// Normalize fields first
	NormalizeDriverFields(req.Driver)

	driver := req.Driver

	// If update mask is provided, only validate specified fields
	if req.UpdateMask != nil {
		return validateMaskedDriverFields(driver, req.UpdateMask)
	}

	// If no mask, validate all non-empty fields
	return validateAllProvidedDriverFields(driver)
}

// validateMaskedDriverFields validates only fields specified in the update mask
func validateMaskedDriverFields(driver *genproto.DriverInput, mask *fieldmaskpb.FieldMask) error {
	for _, path := range mask.Paths {
		switch path {
		case "user_id":
			if driver.UserId != "" {
				if err := ValidateUserID("user_id", driver.UserId); err != nil {
					return err
				}
			}
		case "license_number":
			if driver.LicenseNumber != "" {
				if err := ValidateKenyanLicense("license_number", driver.LicenseNumber); err != nil {
					return err
				}
			}
		case "license_class":
			if driver.LicenseClass != genproto.LicenseClass_LICENSE_UNSPECIFIED {
				if err := ValidateLicenseClass("license_class", driver.LicenseClass); err != nil {
					return err
				}
			}
		case "license_expiry":
			if driver.LicenseExpiry != nil {
				if err := ValidateLicenseExpiry("license_expiry", driver.LicenseExpiry.AsTime()); err != nil {
					return err
				}
			}
		case "experience_years":
			if err := ValidateExperienceYears("experience_years", driver.ExperienceYears); err != nil {
				return err
			}
		case "phone_number":
			if driver.PhoneNumber != "" {
				if err := ValidatePhoneNumber("phone_number", driver.PhoneNumber); err != nil {
					return err
				}
			}
		case "emergency_contact_name":
			if driver.EmergencyContactName != "" {
				if err := ValidateEmergencyContact("emergency_contact_name", "emergency_contact_phone", 
					driver.EmergencyContactName, driver.EmergencyContactPhone); err != nil {
					return err
				}
			}
		case "emergency_contact_phone":
			if driver.EmergencyContactPhone != "" {
				if err := ValidatePhoneNumber("emergency_contact_phone", driver.EmergencyContactPhone); err != nil {
					return err
				}
			}
		case "hire_date":
			if driver.HireDate != nil {
				if err := ValidateHireDate("hire_date", driver.HireDate.AsTime()); err != nil {
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

// validateAllProvidedDriverFields validates all non-empty fields
func validateAllProvidedDriverFields(driver *genproto.DriverInput) error {
	if driver.UserId != "" {
		if err := ValidateUserID("user_id", driver.UserId); err != nil {
			return err
		}
	}

	if driver.LicenseNumber != "" {
		if err := ValidateKenyanLicense("license_number", driver.LicenseNumber); err != nil {
			return err
		}
	}

	if driver.LicenseClass != genproto.LicenseClass_LICENSE_UNSPECIFIED {
		if err := ValidateLicenseClass("license_class", driver.LicenseClass); err != nil {
			return err
		}
	}

	if driver.LicenseExpiry != nil {
		if err := ValidateLicenseExpiry("license_expiry", driver.LicenseExpiry.AsTime()); err != nil {
			return err
		}
	}

	if err := ValidateExperienceYears("experience_years", driver.ExperienceYears); err != nil {
		return err
	}

	if driver.PhoneNumber != "" {
		if err := ValidatePhoneNumber("phone_number", driver.PhoneNumber); err != nil {
			return err
		}
	}

	if driver.EmergencyContactName != "" || driver.EmergencyContactPhone != "" {
		if err := ValidateEmergencyContact("emergency_contact_name", "emergency_contact_phone", 
			driver.EmergencyContactName, driver.EmergencyContactPhone); err != nil {
			return err
		}
	}

	if driver.HireDate != nil {
		if err := ValidateHireDate("hire_date", driver.HireDate.AsTime()); err != nil {
			return err
		}
	}

	return nil
}

// ValidateAddCertificationRequest validates certification addition request
func ValidateAddCertificationRequest(req *genproto.AddDriverCertificationRequest) error {
	if req == nil {
		return ValidationError{Field: "request", Message: "cannot be nil"}
	}

	if req.DriverId == "" {
		return ValidationError{Field: "driver_id", Message: "cannot be empty"}
	}

	if req.Certification == nil {
		return ValidationError{Field: "certification", Message: "cannot be nil"}
	}

	cert := req.Certification

	if err := ValidateCertificationName("certification_name", cert.CertificationName); err != nil {
		return err
	}

	if err := ValidateIssuingAuthority("issued_by", cert.IssuedBy); err != nil {
		return err
	}

	// Validate dates
	if cert.IssueDate == nil {
		return ValidationError{Field: "issue_date", Message: "is required"}
	}

	if cert.ExpiryDate == nil {
		return ValidationError{Field: "expiry_date", Message: "is required"}
	}

	issueDate := cert.IssueDate.AsTime()
	expiryDate := cert.ExpiryDate.AsTime()

	// Issue date cannot be in the future
	if issueDate.After(time.Now()) {
		return ValidationError{
			Field:   "issue_date",
			Message: "cannot be in the future",
		}
	}

	// Expiry date must be after issue date
	if expiryDate.Before(issueDate) {
		return ValidationError{
			Field:   "expiry_date",
			Message: "must be after issue date",
		}
	}

	return nil
}