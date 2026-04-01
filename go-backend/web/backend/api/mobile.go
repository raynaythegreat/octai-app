package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type mobileConfig struct {
	Enabled          bool   `json:"enabled"`
	ConnectionMethod string `json:"connectionMethod"`
	AccessURL        string `json:"accessUrl"`
}

type mobileStatusResponse struct {
	Enabled          bool   `json:"enabled"`
	ConnectionMethod string `json:"connectionMethod"`
	AccessURL        string `json:"accessUrl"`
}

type mobileConfigRequest struct {
	Enabled          *bool  `json:"enabled"`
	ConnectionMethod string `json:"connectionMethod"`
}

func (h *Handler) registerMobileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/mobile/status", h.handleMobileStatus)
	mux.HandleFunc("PUT /api/mobile/config", h.handleMobileConfig)
	mux.HandleFunc("GET /api/mobile/devices", h.handleMobileDevices)
}

func (h *Handler) handleMobileStatus(w http.ResponseWriter, r *http.Request) {
	mobileConfigMu.RLock()
	defer mobileConfigMu.RUnlock()

	resp := mobileStatusResponse{
		Enabled:          mobileConfigData.Enabled,
		ConnectionMethod: mobileConfigData.ConnectionMethod,
		AccessURL:        mobileConfigData.AccessURL,
	}

	if mobileConfigData.Enabled {
		accessURL := resolveMobileAccessURL(mobileConfigData.ConnectionMethod)
		resp.AccessURL = accessURL
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleMobileConfig(w http.ResponseWriter, r *http.Request) {
	var req mobileConfigRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mobileConfigMu.Lock()
	defer mobileConfigMu.Unlock()

	if req.Enabled != nil {
		mobileConfigData.Enabled = *req.Enabled
	}

	if req.ConnectionMethod != "" {
		if req.ConnectionMethod != "tailscale" && req.ConnectionMethod != "lan" {
			http.Error(w, "connectionMethod must be 'tailscale' or 'lan'", http.StatusBadRequest)
			return
		}
		mobileConfigData.ConnectionMethod = req.ConnectionMethod
	}

	if mobileConfigData.Enabled {
		mobileConfigData.AccessURL = resolveMobileAccessURL(mobileConfigData.ConnectionMethod)
	} else {
		mobileConfigData.AccessURL = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleMobileDevices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{})
}

func resolveMobileAccessURL(method string) string {
	if method == "tailscale" && isTailscaleInstalled() {
		ts, err := getTailscaleStatusJSON()
		if err == nil && ts.Self.Online && ts.Self.DNSName != "" {
			return "https://" + ts.Self.DNSName
		}
	}

	if ip := getLocalIP(); ip != "" {
		return "http://" + ip + ":" + fmt.Sprintf("%d", 18800)
	}

	return ""
}
