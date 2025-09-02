//services/user/internal/validator/validate.go
package validator

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"

	"github.com/adammwaniki/bebabeba/services/user/proto/genproto"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Name normalisation for human readability
func NormalizeName(name string) string {
	name = strings.TrimSpace(name)

	// Return early if empty
	if name == "" {
		return name
	}

	// Capitalize first rune of each word-like segment
	var builder strings.Builder
	capNext := true

	for _, r := range name {
		if capNext && unicode.IsLetter(r) {
			builder.WriteRune(unicode.ToUpper(r))
			capNext = false
		} else {
			builder.WriteRune(unicode.ToLower(r))
			capNext = r == ' ' || r == '-' || r == '\''
		}
	}

	return builder.String()
}

// Name validation
func ValidateName(field, name string) error {
	name = NormalizeName(name)

	name = strings.TrimSpace(name)

	if len(name) == 0 {
		return ValidationError{
            Field: field, 
            Message: "must have at least one non-space character",
        }
	}

	if len(name) > 30 {
		return ValidationError{
            Field: field, 
            Message: "cannot be more than 30 characters",
        }
	}

	for _, char := range name {
        if unicode.IsLetter(char) || unicode.IsMark(char) ||
		char == ' ' || char == '-' || char == '\'' || char == '.' || char == ',' {
            continue
        }
        return ValidationError{
            Field: field, 
            Message: fmt.Sprintf("contains invalid character: %q", char),
        }
    }

	return nil
}

// ValidateEmails validates emails for RFC 5322 compliance
func ValidateEmails(field, email string) error {
    trimmedEmail := strings.TrimSpace(email)

    if trimmedEmail == "" {
        return ValidationError{
            Field:   "email",
            Message: "email cannot be empty",
        }
    }

    // Parse and validate the email format using net/mail package
    addr, err := mail.ParseAddress(trimmedEmail)
    if err != nil {
            return ValidationError{
                Field:   field,
                Message: fmt.Sprintf("rfc 5322 invalid email address: %s", err.Error()),
            }
        }
    
    // Check for display name
    if addr.Name != "" {
        return ValidationError{
            Field:   field,
            Message: fmt.Sprintf("rfc 5322 display name prohibited: %q", email),
        }
    }

	// Format consistency check if the parsed address is different from the input
    if addr.Address != trimmedEmail {
        return ValidationError{
            Field:   field,
            Message: "rfc 5322 non-compliant comments or formatting",
        }
    }

    return nil
}

// ValidateSSOID validates the SSO identifier.
// It ensures the ID is not empty after trimming and does not exceed a maximum length.
func ValidateSSOID(field string, ssoID string) error {
	trimmedSsoID := strings.TrimSpace(ssoID)

	if trimmedSsoID == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Example: Maximum length for SSO ID. Adjust as necessary.
	const maxSsoIDLength = 256
	if len(trimmedSsoID) > maxSsoIDLength {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("cannot be more than %d characters", maxSsoIDLength),
		}
	}

	// Other specific validations if needed, e.g., ensure it doesn't contain control characters or only URL-safe characters.
	// But for many SSO IDs (like UUIDs or opaque strings from IdPs),
	// extensive character validation might not be required beyond basic sanity checks.
	// For now, non-empty and length checks are good starting points.

	return nil
}

// ValidatePassword validates the password.
func ValidatePassword(field string, password string) error {
	// Password cannot be empty if this method is chosen.
	if password == "" {
		return ValidationError{
			Field:   field,
			Message: "cannot be empty",
		}
	}

	// Example: Basic length check for password
	const minPasswordLength = 8
	const maxPasswordLength = 128

	if len(password) < minPasswordLength {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters long", minPasswordLength),
		}
	}
	if len(password) > maxPasswordLength {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("cannot be more than %d characters", maxPasswordLength),
		}
	}

	// Room to add more complex rules (e.g., requiring uppercase, lowercase, numbers, special characters)
	// For example:
	// var (
	//  hasUpper   = regexp.MustCompile(`[A-Z]`).MatchString(password)
	//  hasLower   = regexp.MustCompile(`[a-z]`).MatchString(password)
	//  hasNumber  = regexp.MustCompile(`[0-9]`).MatchString(password)
	//  hasSpecial = regexp.MustCompile(`[\W_]`).MatchString(password) // \W is non-word chars
	// )
	// if !(hasUpper && hasLower && hasNumber && hasSpecial) {
	//  return ValidationError{
	//      Field: field,
	//      Message: "must include uppercase, lowercase, number, and special characters",
	//  }
	// }

	return nil
}

// UserInput handles the Update in CRUD for users
// Combined Validation and Normalisation helper for all user input
func ValidateAndNormalizeUserInput(user *genproto.UserInput) error {
	// Validate and normalize first name
	rawFirstName := user.GetFirstName()
	if err := ValidateName("first_name", rawFirstName); err != nil {
		return err
	}
	user.FirstName = NormalizeName(rawFirstName) // Set normalized name

	// Validate and normalize last name
	rawLastName := user.GetLastName()
	if err := ValidateName("last_name", rawLastName); err != nil {
		return err
	}
	user.LastName = NormalizeName(rawLastName) // Set normalized name

	// Validate email
	rawEmail := user.GetEmail()
	if err := ValidateEmails("email", rawEmail); err != nil {
		return err
	}
	user.Email = strings.TrimSpace(rawEmail) // Set trimmed email

	// Validate authentication method (oneof password or sso_id)
	authMethod := user.GetAuthMethod()

	if authMethod == nil {
		// This means neither password nor sso_id is provided in the oneof.
		// For CreateUser, this is an error.
		// For UpdateUser, this might be acceptable if auth_method is not being updated
		// (i.e., not in the update_mask). Assuming this function is called when
		// an auth method IS expected or being explicitly set/updated.
		return ValidationError{
			Field:   "auth_method", // General field name for the oneof
			Message: "either password or sso_id must be provided",
		}
	}

	switch auth := authMethod.(type) {
	case *genproto.UserInput_Password:
		if err := ValidatePassword("password", auth.Password); err != nil {
			return err
		}
		// Our ValidatePassword checks for empty after potential implicit trim if password was " ".
	case *genproto.UserInput_SsoId:
		if err := ValidateSSOID("sso_id", auth.SsoId); err != nil {
			return err
		}
		// SSO IDs are opaque and are typically used as-is.
		// We could trim here if we wanted to:
		// auth.SsoId = strings.TrimSpace(auth.SsoId)
		// user.AuthMethod = &genproto.UserInput_SsoId{SsoId: strings.TrimSpace(auth.SsoId)} // Re-assign to update
	default:
		// This case should ideally not be reached if authMethod was not nil and
		// the oneof only contains Password and SsoId.
		// It's a safeguard for future or unexpected types.
		return ValidationError{
			Field:   "auth_method",
			Message: "an unknown authentication method was provided",
		}
	}

	return nil
}

// RegistrationInput handles the Create in CRUD for users
// Combined Validation and Normalisation helper for all registration input
func ValidateAndNormalizeRegistrationInput(req *genproto.RegistrationRequest) error {
    // Validate and normalize first name
    rawFirstName := req.GetFirstName()
    if err := ValidateName("first_name", rawFirstName); err != nil {
        return err
    }
    req.FirstName = NormalizeName(rawFirstName)

    // Validate and normalize last name
    rawLastName := req.GetLastName()
    if err := ValidateName("last_name", rawLastName); err != nil {
        return err
    }
    req.LastName = NormalizeName(rawLastName)

    // Validate email
    rawEmail := req.GetEmail()
    if err := ValidateEmails("email", rawEmail); err != nil {
        return err
    }
    req.Email = strings.TrimSpace(rawEmail)

    // Validate authentication method
    authMethod := req.GetAuthMethod()
    if authMethod == nil {
        return ValidationError{
            Field:   "auth_method",
            Message: "either password or sso_id must be provided",
        }
    }

    switch auth := authMethod.(type) {
    case *genproto.RegistrationRequest_Password:
        if err := ValidatePassword("password", auth.Password); err != nil {
            return err
        }
    case *genproto.RegistrationRequest_SsoId:
        if err := ValidateSSOID("sso_id", auth.SsoId); err != nil {
            return err
        }
        // req.AuthMethod = &genproto.RegistrationRequest_SsoId{SsoId: strings.TrimSpace(auth.SsoId)} // If trimming
    default:
        return ValidationError{
            Field:   "auth_method",
            Message: "an unknown authentication method was provided",
        }
    }

    return nil
}