package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

type tailscaleStatusResponse struct {
	Installed  bool   `json:"installed"`
	Connected  bool   `json:"connected"`
	IP         string `json:"ip"`
	Hostname   string `json:"hostname"`
	TailnetURL string `json:"tailnetUrl"`
	MagicDNS   bool   `json:"magicDns"`
	AutoStart  bool   `json:"autoStart"`
}

type tailscaleInstallResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type tailscaleAuthResponse struct {
	URL string `json:"url"`
}

type tailscaleDevice struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	OS       string `json:"os"`
	Online   bool   `json:"online"`
	LastSeen string `json:"lastSeen"`
}

type tailscaleConfigRequest struct {
	MagicDNS  *bool `json:"magicDns"`
	AutoStart *bool `json:"autoStart"`
}

type tailscaleStatusJSON struct {
	BackendState string `json:"BackendState"`
	Self         struct {
		ID           string   `json:"ID"`
		HostName     string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
		Online       bool     `json:"Online"`
		DNSName      string   `json:"DNSName"`
	} `json:"Self"`
	Peer map[string]struct {
		ID           string   `json:"ID"`
		HostName     string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
		Online       bool     `json:"Online"`
		OS           string   `json:"OS"`
		LastSeen     string   `json:"LastSeen"`
		DNSName      string   `json:"DNSName"`
	} `json:"Peer"`
}

var (
	mobileConfigMu   sync.RWMutex
	mobileConfigData = mobileConfig{
		Enabled:          false,
		ConnectionMethod: "tailscale",
		AccessURL:        "",
	}
)

func (h *Handler) registerTailscaleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tailscale/status", h.handleTailscaleStatus)
	mux.HandleFunc("POST /api/tailscale/install", h.handleTailscaleInstall)
	mux.HandleFunc("POST /api/tailscale/authenticate", h.handleTailscaleAuthenticate)
	mux.HandleFunc("GET /api/tailscale/devices", h.handleTailscaleDevices)
	mux.HandleFunc("PUT /api/tailscale/config", h.handleTailscaleConfig)
}

func isTailscaleInstalled() bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

func getTailscaleStatusJSON() (*tailscaleStatusJSON, error) {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("tailscale status failed: %w", err)
	}
	var ts tailscaleStatusJSON
	if err := json.Unmarshal(out, &ts); err != nil {
		return nil, fmt.Errorf("failed to parse tailscale status: %w", err)
	}
	return &ts, nil
}

func (h *Handler) handleTailscaleStatus(w http.ResponseWriter, r *http.Request) {
	resp := tailscaleStatusResponse{}

	if !isTailscaleInstalled() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.Installed = true

	ts, err := getTailscaleStatusJSON()
	if err != nil {
		resp.Installed = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.Connected = ts.Self.Online
	resp.Hostname = ts.Self.HostName

	if len(ts.Self.TailscaleIPs) > 0 {
		resp.IP = ts.Self.TailscaleIPs[0]
	}

	if ts.Self.DNSName != "" {
		resp.TailnetURL = "https://" + ts.Self.DNSName
	}

	out, err := exec.Command("tailscale", "get", "--json").Output()
	if err == nil {
		var cfg map[string]interface{}
		if json.Unmarshal(out, &cfg) == nil {
			if v, ok := cfg["AcceptDNS"]; ok {
				if b, ok := v.(bool); ok {
					resp.MagicDNS = b
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) handleTailscaleInstall(w http.ResponseWriter, r *http.Request) {
	if isTailscaleInstalled() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tailscaleInstallResponse{
			Success: true,
			Message: "Tailscale is already installed",
		})
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh")
	case "darwin":
		cmd = exec.Command("brew", "install", "tailscale")
	default:
		http.Error(w, fmt.Sprintf("Unsupported platform: %s", runtime.GOOS), http.StatusBadRequest)
		return
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(tailscaleInstallResponse{
			Success: false,
			Message: fmt.Sprintf("Install failed: %s", string(output)),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tailscaleInstallResponse{
		Success: true,
		Message: "Tailscale installed successfully",
	})
}

func (h *Handler) handleTailscaleAuthenticate(w http.ResponseWriter, r *http.Request) {
	if !isTailscaleInstalled() {
		http.Error(w, "Tailscale is not installed", http.StatusBadRequest)
		return
	}

	out, err := exec.Command("tailscale", "login", "--qr=false").CombinedOutput()
	if err != nil {
		http.Error(w, fmt.Sprintf("Authentication failed: %s", string(out)), http.StatusInternalServerError)
		return
	}

	text := string(out)
	url := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "https://login.tailscale.com/") {
			url = line
			break
		}
	}

	if url == "" {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "http") {
				url = line
				break
			}
		}
	}

	if url == "" {
		http.Error(w, "Failed to parse login URL from tailscale output", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tailscaleAuthResponse{URL: url})
}

func (h *Handler) handleTailscaleDevices(w http.ResponseWriter, r *http.Request) {
	if !isTailscaleInstalled() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]tailscaleDevice{})
		return
	}

	ts, err := getTailscaleStatusJSON()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]tailscaleDevice{})
		return
	}

	devices := make([]tailscaleDevice, 0, len(ts.Peer)+1)

	selfIP := ""
	if len(ts.Self.TailscaleIPs) > 0 {
		selfIP = ts.Self.TailscaleIPs[0]
	}
	devices = append(devices, tailscaleDevice{
		ID:       ts.Self.ID,
		Hostname: ts.Self.HostName,
		IP:       selfIP,
		OS:       runtime.GOOS,
		Online:   ts.Self.Online,
		LastSeen: "",
	})

	for _, peer := range ts.Peer {
		peerIP := ""
		if len(peer.TailscaleIPs) > 0 {
			peerIP = peer.TailscaleIPs[0]
		}
		devices = append(devices, tailscaleDevice{
			ID:       peer.ID,
			Hostname: peer.HostName,
			IP:       peerIP,
			OS:       peer.OS,
			Online:   peer.Online,
			LastSeen: peer.LastSeen,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (h *Handler) handleTailscaleConfig(w http.ResponseWriter, r *http.Request) {
	var req tailscaleConfigRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if !isTailscaleInstalled() {
		http.Error(w, "Tailscale is not installed", http.StatusBadRequest)
		return
	}

	if req.MagicDNS != nil {
		val := "false"
		if *req.MagicDNS {
			val = "true"
		}
		if out, err := exec.Command("tailscale", "set", "--accept-dns="+val).CombinedOutput(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to set DNS config: %s", string(out)), http.StatusInternalServerError)
			return
		}
	}

	if req.AutoStart != nil {
		switch runtime.GOOS {
		case "linux":
			cmd := "disable"
			if *req.AutoStart {
				cmd = "enable"
			}
			if out, err := exec.Command("sudo", "systemctl", cmd, "tailscaled").CombinedOutput(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to set autostart: %s", string(out)), http.StatusInternalServerError)
				return
			}
		case "darwin":
			cmd := "unload"
			plistPath := "/Library/LaunchDaemons/com.tailscale.tailscaled.plist"
			if *req.AutoStart {
				cmd = "load"
			}
			if out, err := exec.Command("sudo", "launchctl", cmd, plistPath).CombinedOutput(); err != nil {
				http.Error(w, fmt.Sprintf("Failed to set autostart: %s", string(out)), http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return ""
}
