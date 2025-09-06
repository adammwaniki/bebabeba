// services/gateway/internal/handler/routes.go (FIXED)
package handler

import (
	"net/http"

	"github.com/adammwaniki/bebabeba/services/auth/session"
	"github.com/adammwaniki/bebabeba/services/gateway/internal/middleware"
)

// SetupAPIRoutes configures the HTTP routes with JWT authentication and session management
func SetupAPIRoutes(
	mux *http.ServeMux, 
	userHandler *UserHandler, 
	authHandler *AuthHandler,
	vehicleHandler *VehicleHandler,
	staffHandler *StaffHandler,
	healthHandler *HealthHandler,
	authMiddleware *middleware.AuthMiddleware,
	sessionManager *session.SessionManager,
) {
	// API v1 subrouter - this handles requests AFTER /api/v1 is stripped
	apiV1Router := http.NewServeMux()

	// Wrapper for Google OAuth callback with session management
	googleCallbackWithSessions := func(w http.ResponseWriter, r *http.Request) {
		userHandler.HandleGoogleCallbackWithJWT(sessionManager, w, r)
	}

	// ================= PUBLIC ENDPOINTS =================
	// No authentication required - these paths are seen WITHOUT /api/v1
	apiV1Router.HandleFunc("POST /users/register", authHandler.HandleCreateUserWithJWT)
	apiV1Router.HandleFunc("POST /auth/login", authHandler.HandleLogin)
	apiV1Router.HandleFunc("POST /auth/refresh", authHandler.HandleRefresh)
	apiV1Router.HandleFunc("GET /auth/google/login", userHandler.HandleGoogleLogin)
	apiV1Router.HandleFunc("GET /auth/google/callback", googleCallbackWithSessions)
	
	// Health endpoints (public)
	apiV1Router.HandleFunc("GET /healthz", healthHandler.LivenessCheck)
	apiV1Router.HandleFunc("GET /readyz", healthHandler.ReadinessCheck)

	// ================= PROTECTED ENDPOINTS =================
	// Require authentication - wrapped with auth middleware individually
	
	// Auth & User Management
	apiV1Router.HandleFunc("GET /auth/profile", authMiddleware.RequireAuth(authHandler.HandleProfile))
	apiV1Router.HandleFunc("GET /auth/sessions", authMiddleware.RequireAuth(authHandler.HandleGetSessions))
	apiV1Router.HandleFunc("POST /auth/logout", authMiddleware.RequireAuth(authHandler.HandleLogout))
	apiV1Router.HandleFunc("GET /users/{id}", authMiddleware.RequireAuth(userHandler.HandleGetUserByID))
	apiV1Router.HandleFunc("GET /users", authMiddleware.RequireAuth(userHandler.HandleListUsers))
	apiV1Router.HandleFunc("PUT /users/{id}", authMiddleware.RequireAuth(userHandler.HandleUpdateUserByID))
	apiV1Router.HandleFunc("DELETE /users/{id}", authMiddleware.RequireAuth(userHandler.HandleDeleteUserByID))

	// ================= TRANSPORT ENDPOINTS =================
	
	// Vehicle Management
	apiV1Router.HandleFunc("POST /transport/vehicles", authMiddleware.RequireAuth(vehicleHandler.HandleCreateVehicle))
	apiV1Router.HandleFunc("GET /transport/vehicles/{id}", authMiddleware.RequireAuth(vehicleHandler.HandleGetVehicle))
	apiV1Router.HandleFunc("GET /transport/vehicles", authMiddleware.RequireAuth(vehicleHandler.HandleListVehicles))
	apiV1Router.HandleFunc("PUT /transport/vehicles/{id}", authMiddleware.RequireAuth(vehicleHandler.HandleUpdateVehicle))
	apiV1Router.HandleFunc("DELETE /transport/vehicles/{id}", authMiddleware.RequireAuth(vehicleHandler.HandleDeleteVehicle))
	apiV1Router.HandleFunc("PATCH /transport/vehicles/{id}/status", authMiddleware.RequireAuth(vehicleHandler.HandleUpdateVehicleStatus))
	
	// Vehicle queries
	apiV1Router.HandleFunc("GET /transport/vehicles/types/{type_id}/vehicles", authMiddleware.RequireAuth(vehicleHandler.HandleGetVehiclesByType))
	apiV1Router.HandleFunc("GET /transport/vehicles/available", authMiddleware.RequireAuth(vehicleHandler.HandleGetAvailableVehicles))
	
	// Vehicle type management
	apiV1Router.HandleFunc("POST /transport/vehicle-types", authMiddleware.RequireAuth(vehicleHandler.HandleCreateVehicleType))
	apiV1Router.HandleFunc("GET /transport/vehicle-types", authMiddleware.RequireAuth(vehicleHandler.HandleListVehicleTypes))

	// ================= STAFF MANAGEMENT =================
	// Restructured to group all literal paths together, then all parameterized paths to handle Go specificity errors
	
	// All literal/static driver endpoints first (no parameters)
	apiV1Router.HandleFunc("GET /transport/drivers/active", authMiddleware.RequireAuth(staffHandler.HandleGetActiveDrivers))
	apiV1Router.HandleFunc("GET /transport/drivers/expiring-licenses", authMiddleware.RequireAuth(staffHandler.HandleGetExpiringLicenses))
	
	// Base driver operations (collection-level)
	apiV1Router.HandleFunc("POST /transport/drivers", authMiddleware.RequireAuth(staffHandler.HandleCreateDriver))
	apiV1Router.HandleFunc("GET /transport/drivers", authMiddleware.RequireAuth(staffHandler.HandleListDrivers))
	
	// User lookup endpoint (moved to avoid conflicts with ID-based routes)
	apiV1Router.HandleFunc("GET /users/{user_id}/driver", authMiddleware.RequireAuth(staffHandler.HandleGetDriverByUserID))
	
	// Individual driver operations (all ID-based routes together)
	apiV1Router.HandleFunc("GET /transport/drivers/{id}", authMiddleware.RequireAuth(staffHandler.HandleGetDriver))
	apiV1Router.HandleFunc("PATCH /transport/drivers/{id}/status", authMiddleware.RequireAuth(staffHandler.HandleUpdateDriverStatus))
	apiV1Router.HandleFunc("POST /transport/drivers/{id}/verify-license", authMiddleware.RequireAuth(staffHandler.HandleVerifyDriverLicense))
	
	// Driver certifications (sub-resource of driver)
	apiV1Router.HandleFunc("POST /transport/drivers/{id}/certifications", authMiddleware.RequireAuth(staffHandler.HandleAddDriverCertification))
	apiV1Router.HandleFunc("GET /transport/drivers/{id}/certifications", authMiddleware.RequireAuth(staffHandler.HandleListDriverCertifications))

	// Mount the API router at /api/v1/ with prefix stripping
	// The StripPrefix happens BEFORE routes are matched, so the apiV1Router sees clean paths
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1Router))
	
	// Redirect requests at /api/v1 to /api/v1/
	mux.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/api/v1/", http.StatusPermanentRedirect)
	})

	// Gateway-level health for load balancers (public) - these see the full path
	mux.HandleFunc("/healthz", healthHandler.LivenessCheck)
	mux.HandleFunc("/readyz", healthHandler.ReadinessCheck)
}