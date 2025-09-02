//services/gateway/internal/handler/routes.go
package handler

import (
	"net/http"

)

// setupAPIRoutes configures the HTTP routes for the API gateway.
func SetupAPIRoutes(mux *http.ServeMux, userHandler *UserHandler, healthHandler *HealthHandler) {

    // API v1 subrouter
    apiV1Router := http.NewServeMux()

    // User endpoints
    apiV1Router.HandleFunc("POST /users/register", userHandler.HandleCreateUser)
    //apiV1Router.HandleFunc("POST /users/login", authHandler.HandleLogin) // Authenticated user
    apiV1Router.HandleFunc("GET /users/{id}", userHandler.HandleGetUserByID)
    apiV1Router.HandleFunc("GET /users", userHandler.HandleListUsers) 
    apiV1Router.HandleFunc("PUT /users/{id}", userHandler.HandleFullyUpdateUserByID)
    apiV1Router.HandleFunc("PATCH /users/{id}", userHandler.HandlePartiallyUpdateUserByID)
    apiV1Router.HandleFunc("DELETE /users/{id}", userHandler.HandleSoftDeleteUserByID)

    // Google OAuth2 authentication endpoints
	apiV1Router.HandleFunc("GET /auth/google/login", userHandler.HandleGoogleLogin)
	apiV1Router.HandleFunc("GET /auth/google/callback", userHandler.HandleGoogleCallback)

    // API Health endpoints
    apiV1Router.HandleFunc("GET /healthz", healthHandler.LivenessCheck)
    apiV1Router.HandleFunc("GET /readyz", healthHandler.ReadinessCheck)

	// Wrap the API router with the HTTP auth middleware
    //authedAPIRouter := middleware.HTTPAuthToContextMiddleware(apiV1Router)

    // Mount versioned and wrapped API at /api/v1/
    //mux.Handle("/api/v1/", http.StripPrefix("/api/v1", authedAPIRouter))
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", apiV1Router))
    // Redirect requests at /api/v1 to /api/v1/
    mux.HandleFunc("/api/v1", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/api/v1/", http.StatusPermanentRedirect)
    })

    // Gateway-level health for load balancers
    mux.HandleFunc("/healthz", healthHandler.LivenessCheck)
    mux.HandleFunc("/readyz", healthHandler.ReadinessCheck)

    // TODO: Add authn/authz middleware
    //mux.Use(authz.ClaimsMiddleware)
}