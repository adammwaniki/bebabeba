//services/gateway/cmd/main.go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adammwaniki/bebabeba/services/gateway/internal/handler"
	userproto "github.com/adammwaniki/bebabeba/services/user/proto/genproto"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	userGRPCAddr = os.Getenv("USER_GRPC_ADDR")
	gatewayAddr  = os.Getenv("GATEWAY_HTTP_ADDR")

    // Google OAuth2 credentials and redirect URL
	googleClientID     = os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")
	// The registered URL in Google Cloud Console for the project's OAuth2 client.
	// It should point to the endpoint where Google redirects after authorization.
	googleRedirectURL  = os.Getenv("GOOGLE_REDIRECT_URL")
)


func main() {
	// Create gRPC connection to the User Service WITH the interceptor
    userConn, err := grpc.NewClient(
        userGRPCAddr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal("Failed to dial coreuser service: ", err)
    }
    defer userConn.Close()

    // Create user service gRPC client and health client.
	userClient := userproto.NewUserServiceClient(userConn)
	userHealth := grpc_health_v1.NewHealthClient(userConn)

    // Configure Google OAuth2.
	googleOAuthConfig := &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  googleRedirectURL,
		Scopes:       []string{"openid", "email", "profile"}, // Standard OIDC scopes
		Endpoint:     google.Endpoint,
	}

	// Setup handlers
    healthHandler := handler.NewHealthHandler(userHealth)
	userHandler := handler.NewUserHandler(userClient, googleOAuthConfig)

    // Configure server
    mux := http.NewServeMux()
    handler.SetupAPIRoutes(mux, userHandler, healthHandler)
    
    server := &http.Server{
        Addr:    gatewayAddr,
        Handler: mux,
    }

    // Graceful shutdown setup: listen for termination signals
    done := make(chan os.Signal, 1)
    signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

    // Start the HTTP server in a goroutine
    go func() {
        log.Printf("Gateway server starting on %s", gatewayAddr)
        if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
    }()

    // Wait for shutdown signal
    <-done
    log.Println("Server shutting down...")

    // Mark service as not ready to gracefully stop new requests
    healthHandler.MarkNotReady()

    // Allow existing requests to finish within a timeout
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Printf("Server shutdown error: %v", err)
    }
    log.Println("Server stopped")
}

