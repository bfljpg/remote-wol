package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"remote-wol/internal/agent"
	"remote-wol/internal/db"
)

type DeviceHandler struct {
	db    *db.DB
	agent *agent.Client
}

type createDeviceRequest struct {
	Name         string `json:"name"`
	MACAddress   string `json:"mac_address"`
	IPAddress    string `json:"ip_address"`
	NetInterface string `json:"net_interface"`
	Icon         string `json:"icon"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type messageResponse struct {
	Message string `json:"message"`
}

func NewDeviceHandler(database *db.DB, agentClient *agent.Client) *DeviceHandler {
	return &DeviceHandler{
		db:    database,
		agent: agentClient,
	}
}

// ServeHTTP routes device requests based on path and method
func (h *DeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimPrefix(r.URL.Path, "/api/devices")
	path = strings.TrimPrefix(path, "/")

	// GET /api/devices
	if path == "" && r.Method == http.MethodGet {
		h.listDevices(w, r)
		return
	}

	// POST /api/devices
	if path == "" && r.Method == http.MethodPost {
		h.createDevice(w, r)
		return
	}

	// GET /api/devices/status/all
	if path == "status/all" && r.Method == http.MethodGet {
		h.allStatus(w, r)
		return
	}

	// GET /api/devices/agent/health
	if path == "agent/health" && r.Method == http.MethodGet {
		h.agentHealth(w, r)
		return
	}

	// Routes with device ID: /api/devices/{id}[/action]
	parts := strings.SplitN(path, "/", 2)
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "invalid device id"})
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.getDevice(w, r, id)
		case http.MethodPut:
			h.updateDevice(w, r, id)
		case http.MethodDelete:
			h.deleteDevice(w, r, id)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		}
		return
	}

	// /api/devices/{id}/wake or /api/devices/{id}/status
	action := parts[1]
	switch {
	case action == "wake" && r.Method == http.MethodPost:
		h.wakeDevice(w, r, id)
	case action == "status" && r.Method == http.MethodGet:
		h.deviceStatus(w, r, id)
	default:
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	}
}

func (h *DeviceHandler) listDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.db.ListDevices()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list devices"})
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *DeviceHandler) getDevice(w http.ResponseWriter, r *http.Request, id int64) {
	dev, err := h.db.GetDevice(id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "device not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database error"})
		return
	}
	writeJSON(w, http.StatusOK, dev)
}

func (h *DeviceHandler) createDevice(w http.ResponseWriter, r *http.Request) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" || req.MACAddress == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and mac_address are required"})
		return
	}

	if req.NetInterface == "" {
		req.NetInterface = "br-lan"
	}
	if req.Icon == "" {
		req.Icon = "desktop"
	}

	dev := &db.Device{
		Name:         req.Name,
		MACAddress:   req.MACAddress,
		IPAddress:    req.IPAddress,
		NetInterface: req.NetInterface,
		Icon:         req.Icon,
	}

	if err := h.db.CreateDevice(dev); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to create device"})
		return
	}

	// Re-read from DB to get auto-populated timestamps
	created, _ := h.db.GetDevice(dev.ID)
	if created != nil {
		dev = created
	}
	writeJSON(w, http.StatusCreated, dev)
}

func (h *DeviceHandler) updateDevice(w http.ResponseWriter, r *http.Request, id int64) {
	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" || req.MACAddress == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "name and mac_address are required"})
		return
	}

	if req.NetInterface == "" {
		req.NetInterface = "br-lan"
	}
	if req.Icon == "" {
		req.Icon = "desktop"
	}

	dev := &db.Device{
		ID:           id,
		Name:         req.Name,
		MACAddress:   req.MACAddress,
		IPAddress:    req.IPAddress,
		NetInterface: req.NetInterface,
		Icon:         req.Icon,
	}

	if err := h.db.UpdateDevice(dev); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to update device"})
		return
	}

	updated, _ := h.db.GetDevice(id)
	writeJSON(w, http.StatusOK, updated)
}

func (h *DeviceHandler) deleteDevice(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.db.DeleteDevice(id); err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "device not found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete device"})
		return
	}
	writeJSON(w, http.StatusOK, messageResponse{Message: "device deleted"})
}

func (h *DeviceHandler) wakeDevice(w http.ResponseWriter, r *http.Request, id int64) {
	dev, err := h.db.GetDevice(id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "device not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database error"})
		return
	}

	result, err := h.agent.Wake(dev.MACAddress, dev.NetInterface)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "agent error: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *DeviceHandler) deviceStatus(w http.ResponseWriter, r *http.Request, id int64) {
	dev, err := h.db.GetDevice(id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "device not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database error"})
		return
	}

	if dev.IPAddress == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "device has no IP address configured"})
		return
	}

	result, err := h.agent.Ping(dev.IPAddress)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: "agent error: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":   dev.ID,
		"device_name": dev.Name,
		"online":      result.Alive,
		"ip":          dev.IPAddress,
	})
}

func (h *DeviceHandler) allStatus(w http.ResponseWriter, r *http.Request) {
	devices, err := h.db.ListDevices()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to list devices"})
		return
	}

	type deviceStatus struct {
		DeviceID   int64  `json:"device_id"`
		DeviceName string `json:"device_name"`
		Online     bool   `json:"online"`
		IP         string `json:"ip"`
		Error      string `json:"error,omitempty"`
	}

	statuses := make([]deviceStatus, 0, len(devices))
	for _, dev := range devices {
		ds := deviceStatus{
			DeviceID:   dev.ID,
			DeviceName: dev.Name,
			IP:         dev.IPAddress,
		}

		if dev.IPAddress != "" {
			result, err := h.agent.Ping(dev.IPAddress)
			if err != nil {
				ds.Error = err.Error()
			} else {
				ds.Online = result.Alive
			}
		}

		statuses = append(statuses, ds)
	}

	writeJSON(w, http.StatusOK, statuses)
}

func (h *DeviceHandler) agentHealth(w http.ResponseWriter, r *http.Request) {
	result, err := h.agent.Health()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
		"agent":     result,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
