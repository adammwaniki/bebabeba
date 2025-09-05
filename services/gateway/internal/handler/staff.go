// services/gateway/internal/handler/staff.go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/adammwaniki/bebabeba/services/common/utils"
	staffproto "github.com/adammwaniki/bebabeba/services/staff/proto/genproto"
	"github.com/gofrs/uuid/v5"
)

// StaffHandler handles HTTP requests for the staff service
type StaffHandler struct {
	staffClient staffproto.StaffServiceClient
}

// NewStaffHandler creates a new staff handler
func NewStaffHandler(staffClient staffproto.StaffServiceClient) *StaffHandler {
	return &StaffHandler{
		staffClient: staffClient,
	}
}

// HandleCreateDriver handles POST requests to create a new driver
func (h *StaffHandler) HandleCreateDriver(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	// Parse the request payload
	var driverInput staffproto.DriverInput
	if err := json.Unmarshal(body, &driverInput); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Create the gRPC request
	grpcReq := &staffproto.CreateDriverRequest{
		Driver: &driverInput,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.CreateDriver(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusCreated, resp)
}

// HandleGetDriver handles GET requests to retrieve a driver by ID
func (h *StaffHandler) HandleGetDriver(w http.ResponseWriter, r *http.Request) {
	driverIDStr := r.PathValue("id")
	if driverIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("driver ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(driverIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid driver ID format: %w", err))
		return
	}

	// Create gRPC request
	grpcReq := &staffproto.GetDriverRequest{
		DriverId: driverIDStr,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.GetDriver(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleGetDriverByUserID handles GET requests to retrieve a driver by user ID
func (h *StaffHandler) HandleGetDriverByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("user_id")
	if userIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("user ID is required"))
		return
	}

	// Create gRPC request
	grpcReq := &staffproto.GetDriverByUserIDRequest{
		UserId: userIDStr,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.GetDriverByUserID(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleListDrivers handles GET requests to list drivers
func (h *StaffHandler) HandleListDrivers(w http.ResponseWriter, r *http.Request) {
	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &staffproto.ListDriversRequest{
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Handle filters
	if status := r.URL.Query().Get("status"); status != "" {
		if statusVal, ok := staffproto.DriverStatus_value[status]; ok {
			grpcReq.StatusFilter = staffproto.DriverStatus(statusVal).Enum()
		}
	}

	if licenseClass := r.URL.Query().Get("license_class"); licenseClass != "" {
		if classVal, ok := staffproto.LicenseClass_value[licenseClass]; ok {
			grpcReq.LicenseClassFilter = staffproto.LicenseClass(classVal).Enum()
		}
	}

	if expiring := r.URL.Query().Get("license_expiring_soon"); expiring == "true" {
		grpcReq.LicenseExpiringSoon = &[]bool{true}[0]
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.ListDrivers(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleUpdateDriverStatus handles PATCH requests to update driver status
func (h *StaffHandler) HandleUpdateDriverStatus(w http.ResponseWriter, r *http.Request) {
	driverIDStr := r.PathValue("id")
	if driverIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("driver ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(driverIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid driver ID format: %w", err))
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	var statusRequest struct {
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	}

	if err := json.Unmarshal(body, &statusRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Validate status
	statusVal, ok := staffproto.DriverStatus_value[statusRequest.Status]
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid status: %s", statusRequest.Status))
		return
	}

	// Create gRPC request
	grpcReq := &staffproto.UpdateDriverStatusRequest{
		DriverId: driverIDStr,
		Status:   staffproto.DriverStatus(statusVal),
		Reason:   statusRequest.Reason,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.UpdateDriverStatus(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleGetActiveDrivers handles GET requests to get active drivers
func (h *StaffHandler) HandleGetActiveDrivers(w http.ResponseWriter, r *http.Request) {
	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &staffproto.GetActiveDriversRequest{
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Handle license class filter
	if licenseClass := r.URL.Query().Get("license_class"); licenseClass != "" {
		if classVal, ok := staffproto.LicenseClass_value[licenseClass]; ok {
			grpcReq.LicenseClassFilter = staffproto.LicenseClass(classVal).Enum()
		}
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.GetActiveDrivers(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleAddDriverCertification handles POST requests to add driver certifications
func (h *StaffHandler) HandleAddDriverCertification(w http.ResponseWriter, r *http.Request) {
	driverIDStr := r.PathValue("id")
	if driverIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("driver ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(driverIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid driver ID format: %w", err))
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	var certInput staffproto.CertificationInput
	if err := json.Unmarshal(body, &certInput); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Create gRPC request
	grpcReq := &staffproto.AddDriverCertificationRequest{
		DriverId:      driverIDStr,
		Certification: &certInput,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.AddDriverCertification(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusCreated, resp)
}

// HandleListDriverCertifications handles GET requests to list driver certifications
func (h *StaffHandler) HandleListDriverCertifications(w http.ResponseWriter, r *http.Request) {
	driverIDStr := r.PathValue("id")
	if driverIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("driver ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(driverIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid driver ID format: %w", err))
		return
	}

	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &staffproto.ListDriverCertificationsRequest{
		DriverId:  driverIDStr,
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Handle filters
	if status := r.URL.Query().Get("status"); status != "" {
		if statusVal, ok := staffproto.CertificationStatus_value[status]; ok {
			grpcReq.StatusFilter = staffproto.CertificationStatus(statusVal).Enum()
		}
	}

	if expiring := r.URL.Query().Get("expiring_soon"); expiring == "true" {
		grpcReq.ExpiringSoon = &[]bool{true}[0]
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.ListDriverCertifications(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleVerifyDriverLicense handles POST requests to verify driver licenses
func (h *StaffHandler) HandleVerifyDriverLicense(w http.ResponseWriter, r *http.Request) {
	driverIDStr := r.PathValue("id")
	if driverIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("driver ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(driverIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid driver ID format: %w", err))
		return
	}

	// Read optional request body for license number verification
	var verifyRequest struct {
		LicenseNumber string `json:"license_number,omitempty"`
	}

	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err == nil {
			json.Unmarshal(body, &verifyRequest)
		}
		defer r.Body.Close()
	}

	// Create gRPC request
	grpcReq := &staffproto.VerifyDriverLicenseRequest{
		DriverId:      driverIDStr,
		LicenseNumber: verifyRequest.LicenseNumber,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.VerifyDriverLicense(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleGetExpiringLicenses handles GET requests to get drivers with expiring licenses
func (h *StaffHandler) HandleGetExpiringLicenses(w http.ResponseWriter, r *http.Request) {
	daysAhead := int32(30) // Default 30 days
	if da := r.URL.Query().Get("days_ahead"); da != "" {
		if n, err := strconv.Atoi(da); err == nil && n > 0 {
			daysAhead = int32(n)
		}
	}

	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &staffproto.GetExpiringLicensesRequest{
		DaysAhead: daysAhead,
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.staffClient.GetExpiringLicenses(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}