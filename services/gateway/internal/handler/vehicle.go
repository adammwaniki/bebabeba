// services/gateway/internal/handler/vehicle.go
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
	vehicleproto "github.com/adammwaniki/bebabeba/services/vehicle/proto/genproto"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// VehicleHandler handles HTTP requests for the vehicle service
type VehicleHandler struct {
	vehicleClient vehicleproto.VehicleServiceClient
}

// NewVehicleHandler creates a new vehicle handler
func NewVehicleHandler(vehicleClient vehicleproto.VehicleServiceClient) *VehicleHandler {
	return &VehicleHandler{
		vehicleClient: vehicleClient,
	}
}

// HandleCreateVehicle handles POST requests to create a new vehicle
func (h *VehicleHandler) HandleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	// Parse the request payload
	var vehicleInput vehicleproto.VehicleInput
	if err := json.Unmarshal(body, &vehicleInput); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Create the gRPC request
	grpcReq := &vehicleproto.CreateVehicleRequest{
		Vehicle: &vehicleInput,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.CreateVehicle(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusCreated, resp)
}

// HandleGetVehicle handles GET requests to retrieve a vehicle by ID
func (h *VehicleHandler) HandleGetVehicle(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := r.PathValue("id")
	if vehicleIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(vehicleIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid vehicle ID format: %w", err))
		return
	}

	// Create gRPC request
	grpcReq := &vehicleproto.GetVehicleRequest{
		VehicleId: vehicleIDStr,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.GetVehicle(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleListVehicles handles GET requests to list vehicles
func (h *VehicleHandler) HandleListVehicles(w http.ResponseWriter, r *http.Request) {
	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &vehicleproto.ListVehiclesRequest{
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Handle filters
	if status := r.URL.Query().Get("status"); status != "" {
		if statusVal, ok := vehicleproto.VehicleStatus_value[status]; ok {
			grpcReq.StatusFilter = vehicleproto.VehicleStatus(statusVal).Enum()
		}
	}

	if vehicleType := r.URL.Query().Get("vehicle_type"); vehicleType != "" {
		grpcReq.VehicleTypeFilter = &vehicleType
	}

	if make := r.URL.Query().Get("make"); make != "" {
		grpcReq.MakeFilter = &make
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.ListVehicles(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleUpdateVehicle handles PUT requests to update a vehicle
func (h *VehicleHandler) HandleUpdateVehicle(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := r.PathValue("id")
	if vehicleIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(vehicleIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid vehicle ID format: %w", err))
		return
	}

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	var updateRequest struct {
		Vehicle    *vehicleproto.VehicleInput `json:"vehicle"`
		UpdateMask []string                   `json:"update_mask,omitempty"`
	}

	if err := json.Unmarshal(body, &updateRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	if updateRequest.Vehicle == nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle data is required"))
		return
	}

	// Create field mask if provided
	var fieldMask *fieldmaskpb.FieldMask
	if len(updateRequest.UpdateMask) > 0 {
		fieldMask = &fieldmaskpb.FieldMask{
			Paths: updateRequest.UpdateMask,
		}
	}

	// Create gRPC request
	grpcReq := &vehicleproto.UpdateVehicleRequest{
		VehicleId:  vehicleIDStr,
		Vehicle:    updateRequest.Vehicle,
		UpdateMask: fieldMask,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.UpdateVehicle(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleDeleteVehicle handles DELETE requests to soft-delete a vehicle
func (h *VehicleHandler) HandleDeleteVehicle(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := r.PathValue("id")
	if vehicleIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(vehicleIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid vehicle ID format: %w", err))
		return
	}

	// Create gRPC request
	grpcReq := &vehicleproto.DeleteVehicleRequest{
		VehicleId: vehicleIDStr,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	_, err = h.vehicleClient.DeleteVehicle(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleGetVehiclesByType handles GET requests to get vehicles by type
func (h *VehicleHandler) HandleGetVehiclesByType(w http.ResponseWriter, r *http.Request) {
	vehicleTypeID := r.PathValue("type_id")
	if vehicleTypeID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle type ID is required"))
		return
	}

	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &vehicleproto.GetVehiclesByTypeRequest{
		VehicleTypeId: vehicleTypeID,
		PageSize:      pageSize,
		PageToken:     r.URL.Query().Get("page_token"),
	}

	// Handle status filter
	if status := r.URL.Query().Get("status"); status != "" {
		if statusVal, ok := vehicleproto.VehicleStatus_value[status]; ok {
			grpcReq.StatusFilter = vehicleproto.VehicleStatus(statusVal).Enum()
		}
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.GetVehiclesByType(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleGetAvailableVehicles handles GET requests to get available vehicles
func (h *VehicleHandler) HandleGetAvailableVehicles(w http.ResponseWriter, r *http.Request) {
	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &vehicleproto.GetAvailableVehiclesRequest{
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Handle vehicle type filter
	if vehicleType := r.URL.Query().Get("vehicle_type"); vehicleType != "" {
		grpcReq.VehicleTypeId = &vehicleType
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.GetAvailableVehicles(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// HandleUpdateVehicleStatus handles PATCH requests to update vehicle status
func (h *VehicleHandler) HandleUpdateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	vehicleIDStr := r.PathValue("id")
	if vehicleIDStr == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("vehicle ID is required"))
		return
	}

	// Validate UUID format
	_, err := uuid.FromString(vehicleIDStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid vehicle ID format: %w", err))
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
	}

	if err := json.Unmarshal(body, &statusRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Validate status
	statusVal, ok := vehicleproto.VehicleStatus_value[statusRequest.Status]
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid status: %s", statusRequest.Status))
		return
	}

	// Create gRPC request
	grpcReq := &vehicleproto.UpdateVehicleStatusRequest{
		VehicleId: vehicleIDStr,
		Status:    vehicleproto.VehicleStatus(statusVal),
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.UpdateVehicleStatus(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}

// Vehicle type management

// HandleCreateVehicleType handles POST requests to create a vehicle type
func (h *VehicleHandler) HandleCreateVehicleType(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %w", err))
		return
	}
	defer r.Body.Close()

	var typeRequest struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &typeRequest); err != nil {
		utils.WriteError(w, http.StatusBadRequest, fmt.Errorf("invalid request format: %w", err))
		return
	}

	// Create gRPC request
	grpcReq := &vehicleproto.CreateVehicleTypeRequest{
		Name:        typeRequest.Name,
		Description: typeRequest.Description,
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.CreateVehicleType(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusCreated, resp)
}

// HandleListVehicleTypes handles GET requests to list vehicle types
func (h *VehicleHandler) HandleListVehicleTypes(w http.ResponseWriter, r *http.Request) {
	pageSize := int32(50) // Default page size
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 {
			pageSize = int32(n)
		}
	}

	// Create gRPC request
	grpcReq := &vehicleproto.ListVehicleTypesRequest{
		PageSize:  pageSize,
		PageToken: r.URL.Query().Get("page_token"),
	}

	// Set context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Call the gRPC service
	resp, err := h.vehicleClient.ListVehicleTypes(ctx, grpcReq)
	if err != nil {
		utils.HandleGRPCError(w, err)
		return
	}

	utils.WriteProtoJSON(w, http.StatusOK, resp)
}