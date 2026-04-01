package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/web/backend/api"
	"github.com/raynaythegreat/octai-app/web/backend/launcherconfig"
	"github.com/raynaythegreat/octai-app/web/backend/middleware"
	"github.com/raynaythegreat/octai-app/web/backend/utils"
)

const (
	appName   = "OctAi"
	logPath   = "logs"
	panicFile = "octai_panic.log"
	logFile   = "octai.log"
)

var (
	appVersion = config.Version

	server     *http.Server
	apiHandler *api.Handler
	shutdownMu sync.Once
)

func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func detectTailscale() (string, bool) {
	ip, err := net.ResolveIPAddr("ip", "octai.tail1234.ts.net")
	if err != nil {
		if out, err := exec.Command("tailscale", "ip", "-4").Output(); err == nil && len(out) > 0 {
			return string(out[:len(out)-1]), true
		}
		return "", false
	}
	return ip.IP.String(), true
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		logger.Errorf("Failed to start PTY: %v", err)
		return
	}
	defer ptmx.Close()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if readErr != nil {
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		ptmx.Write(msg)
	}
}

func gracefulShutdown() {
	shutdownMu.Do(func() {
		logger.Info("Shutting down OctAi backend...")

		if apiHandler != nil {
			apiHandler.Shutdown()
		}

		if server != nil {
			server.SetKeepAlivesEnabled(false)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					logger.Info("Server shutdown timeout, forcing close")
				} else {
					logger.Errorf("Server shutdown error: %v", err)
				}
			}
		}

		logger.Info("Shutdown complete")
	})
}

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	public := flag.Bool("public", false, "Listen on all interfaces (0.0.0.0) instead of localhost only")
	console := flag.Bool("console", true, "Console mode (always true for Tauri backend)")
	flag.Parse()

	picoHome := utils.GetPicoclawHome()

	f := filepath.Join(picoHome, logPath, panicFile)
	panicFunc, err := logger.InitPanic(f)
	if err != nil {
		panic(fmt.Sprintf("error initializing panic log: %v", err))
	}
	defer panicFunc()

	if !*console {
		logger.SetConsoleLevel(logger.FATAL)
		f := filepath.Join(picoHome, logPath, logFile)
		if err = logger.EnableFileLogging(f); err != nil {
			panic(fmt.Sprintf("error enabling file logging: %v", err))
		}
		defer logger.DisableFileLogging()
	}

	logger.InfoC("web", fmt.Sprintf("%s Backend %s starting...", appName, appVersion))
	logger.InfoC("web", fmt.Sprintf("OctAi Home: %s", picoHome))

	if tsIP, ok := detectTailscale(); ok {
		logger.InfoC("web", fmt.Sprintf("Tailscale detected: %s", tsIP))
	}

	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.Fatalf("Failed to resolve config path: %v", err)
	}
	if err = utils.EnsureOnboarded(absPath); err != nil {
		logger.Errorf("Warning: Failed to initialize OctAi config automatically: %v", err)
	}

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: Failed to load %s: %v", launcherPath, err))
		launcherCfg = launcherconfig.Default()
	}

	effectivePort := *port
	if !isPortAvailable(18800) && *port == "18800" {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		logger.Fatalf("Invalid port %q: %v", effectivePort, err)
	}

	var addr string
	if *public {
		addr = "0.0.0.0:" + effectivePort
	} else {
		addr = "127.0.0.1:" + effectivePort
	}

	mux := http.NewServeMux()

	apiHandler = api.NewHandler(absPath)
	if _, err = apiHandler.EnsurePicoChannel(""); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to ensure pico channel on startup: %v", err))
	}
	apiHandler.SetServerOptions(portNum, *public, false, launcherCfg.AllowedCIDRs)
	apiHandler.RegisterRoutes(mux)

	mux.HandleFunc("/ws/terminal", handleTerminalWS)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		logger.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.JSONContentType(accessControlledMux),
		),
	)

	fmt.Print(utils.Banner)
	fmt.Println()
	fmt.Printf("  OctAi Backend listening on http://localhost:%s\n", effectivePort)
	if *public {
		if ip := utils.GetLocalIP(); ip != "" {
			fmt.Printf("  Public access: http://%s:%s\n", ip, effectivePort)
		}
	}
	fmt.Println()

	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	server = &http.Server{Addr: addr, Handler: handler}
	go func() {
		logger.InfoC("web", fmt.Sprintf("Server listening on %s", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	for {
		sig := <-sigChan
		if sig == syscall.SIGHUP {
			logger.Info("SIGHUP received, continuing to run")
			continue
		}
		gracefulShutdown()
		return
	}
}
