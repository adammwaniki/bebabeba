//services/user/internal/service/service.go
package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/adammwaniki/bebabeba/services/auth/authn/passwords"
	"github.com/adammwaniki/bebabeba/services/common/utils"
	"github.com/adammwaniki/bebabeba/services/user/internal/types"
	"github.com/adammwaniki/bebabeba/services/user/internal/validator"
	"github.com/adammwaniki/bebabeba/services/user/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"github.com/influxdata/influxdb/v2/pkg/snowflake"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service contains business logic pertaining to the user
type service struct {
	store types.UserStore
}

// NewService creates a new instance of the user service
func NewService(store types.UserStore) *service {
	return &service{store: store}
}

// CreateUser handles the creation of a new user, supporting both password and SSO authentication
func (s *service) CreateUser(ctx context.Context, user *genproto.RegistrationRequest) (*genproto.CreateUserResponse, error) {
	// Validate incoming registration request based on business rules
    if err := validator.ValidateAndNormalizeRegistrationInput(user); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
    }

	// Prepare variables for the hashed password and SSO ID.
	// These will be pointers to strings, allowing them to be nil if not used.
	var hashedPassword *string
	var ssoID *string

	// Determine the authentication method provided in the request (password or SSO ID).
	switch authMethod := user.AuthMethod.(type) {
	case *genproto.RegistrationRequest_Password:
		// If a password is provided, hash it using the authn package
		hash, err := passwords.HashPassword(authMethod.Password)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash password: %v", err)
		}
		hashedPassword = &hash // Assign the address of the hashed string to hashedPassword
	case *genproto.RegistrationRequest_SsoId:
		// If an SSO ID is provided, assign its address to ssoID.
		ssoID = &authMethod.SsoId
	default:
		// This case should ideally be caught by ValidateAndNormalizeRegistrationInput,
		// but it serves as a fallback for unexpected scenarios.
		return nil, status.Errorf(codes.InvalidArgument, "neither password nor SSO ID provided")
	}

    // Generate a unique internal_id using Snowflake
	nodeID, err := utils.GetSnowflakeNodeID()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get snowflake node ID: %v", err)
	}

	// Generate new internal snowflakeID
	generator := snowflake.New(int(nodeID))
	inID := generator.Next() 

	// Generate new external UUIDV4
	exID, err := uuid.NewV4()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate UUID: %v", err)
	}
	log.Printf("Generated UUID: %s, Bytes: %x", exID.String(), exID.Bytes())

    // Call the store layer to persist the new user.
	// If the error from the store is already a gRPC status error (e.g., AlreadyExists),
	// return it directly. Otherwise, wrap it as an Internal error.
	if err := s.store.Create(
		ctx,
		inID,
		exID,
		user.FirstName,
		user.LastName,
		user.Email,
		hashedPassword, // Pass the hashed password (or nil if SSO)
		ssoID,          // Pass the SSO ID (or nil if password)
	); err != nil {
		// Check for specific domain errors and map them to gRPC codes
		if errors.Is(err, types.ErrDuplicateEntry) {
			// The specific details (email, ID) will be logged by the store,
			// but the gRPC status message should be concise.
			return nil, status.Errorf(codes.AlreadyExists, "email is already in use")
		}
		// For any other unexpected store error, return an Internal gRPC error
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

    // Prepare and return the CreateUserResponse
	now := timestamppb.New(time.Now())

    return &genproto.CreateUserResponse{
        Id:        exID.String(), // UUIDV4 string instead of snowflake as surrogate so as not to expose ID logic 
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Status:    genproto.UserStatusEnum_ACTIVE, // Default status is active
		TermsAcceptedAt: now,
		CreatedAt: now,
    }, nil
}

func (s *service) GetUserByID(ctx context.Context, req *genproto.GetUserRequest) (*genproto.GetUserResponse, error) {
	// Parse string UUID
    idStr := req.GetUserId()
    id, err := uuid.FromString(idStr)
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid user ID format: %v", err)
    }

    // Call the store layer with the proper uuid.UUID type
    user, err := s.store.GetByID(ctx, id)
    if err != nil {
        // Wrapping specific store errors with service-level errors for better error handling
        if errors.Is(err, sql.ErrNoRows) { // Check if the error is due to no rows found in the database
            return nil, status.Errorf(codes.NotFound, "user not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get user from store: %v", err)
    }
    return user, nil
}

// GetUserBySSOID retrieves a user by their SSO ID.
func (s *service) GetUserBySSOID(ctx context.Context, req *genproto.GetUserBySSOIDRequest) (*genproto.GetUserResponse, error) {
	if req.GetSsoId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "SSO ID cannot be empty")
	}

	user, err := s.store.GetUserBySSOID(ctx, req.GetSsoId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user by SSO ID from store: %v", err)
	}
	return user, nil
}

// Authentication service method
func (s *service) GetUserForAuth(ctx context.Context, req *genproto.GetUserForAuthRequest) (*genproto.AuthUserResponse, error) {
    user, err := s.store.GetUserForAuth(ctx, req.Email)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
    }
    return user, nil
}

// ListUsers returns a list of users from the db
func (s *service) ListUsers(ctx context.Context, req *genproto.ListUsersRequest) (*genproto.ListUsersResponse, error) {
	// Validate page size
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 50 // Default
	}
	if pageSize > 100 {
		pageSize = 100 // Maximum limit
	}

	// Call store layer
	users, nextPageToken, err := s.store.ListUsers(
		ctx,
		pageSize,
		req.GetPageToken(),
		req.StatusFilter,
		req.GetNameFilter(),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	return &genproto.ListUsersResponse{
		Users:         users,
		NextPageToken: nextPageToken,
	}, nil
}