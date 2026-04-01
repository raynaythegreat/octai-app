// OctAi Dashboard Server - Web-based chat and management interface
//
// Provides a web UI for chatting with OctAi via the Pico Channel WebSocket,
// with configuration management and gateway process control.

package webconsole

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/web/backend/api"
	"github.com/raynaythegreat/octai-app/web/backend/launcherconfig"
	"github.com/raynaythegreat/octai-app/web/backend/middleware"
	"github.com/raynaythegreat/octai-app/web/backend/utils"
)

const (
	appName = "OctAi"

	logPath   = "logs"
	panicFile = "launcher_panic.log"
	logFile   = "launcher.log"
)

var (
	appVersion = config.Version

	server     *http.Server
	serverAddr string
	apiHandler *api.Handler

	// noBrowserFlag is set by Run() and read by systray files.
	noBrowserFlag bool
)

// Options controls how the web console starts.
type Options struct {
	Port           string
	Public         bool
	NoBrowser      bool
	Lang           string
	Console        bool
	ConfigPath     string
	ExplicitPort   bool // true when the caller explicitly set Port
	ExplicitPublic bool // true when the caller explicitly set Public
}

// Run starts the OctAi web dashboard with the given options.
func Run(opts Options) error {
	noBrowserFlag = opts.NoBrowser

	// Initialize logger
	picoHome := utils.GetAIBHQHome()

	f := filepath.Join(picoHome, logPath, panicFile)
	panicFunc, err := logger.InitPanic(f)
	if err != nil {
		return fmt.Errorf("error initializing panic log: %w", err)
	}
	defer panicFunc()

	enableConsole := opts.Console
	if !enableConsole {
		logger.SetConsoleLevel(logger.FATAL)

		f := filepath.Join(picoHome, logPath, logFile)
		if err = logger.EnableFileLogging(f); err != nil {
			return fmt.Errorf("error enabling file logging: %w", err)
		}
		defer logger.DisableFileLogging()
	}

	logger.InfoC("web", fmt.Sprintf("%s Launcher %s starting...", appName, appVersion))
	logger.InfoC("web", fmt.Sprintf("OctAi Home: %s", picoHome))

	if opts.Lang != "" {
		SetLanguage(opts.Lang)
	}

	// Resolve config path
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = utils.GetDefaultConfigPath()
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("failed to resolve config path: %w", err)
	}
	if err = utils.EnsureOnboarded(absPath); err != nil {
		logger.Errorf("Warning: Failed to initialize OctAi config automatically: %v", err)
	}
	if err = utils.ImportLegacyConfigIfNeeded(absPath); err != nil {
		logger.Errorf("Warning: Failed to import legacy model config automatically: %v", err)
	}

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: Failed to load %s: %v", launcherPath, err))
		launcherCfg = launcherconfig.Default()
	}

	effectivePort := opts.Port
	if effectivePort == "" {
		effectivePort = "18800"
	}
	effectivePublic := opts.Public
	if !opts.ExplicitPort {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}
	if !opts.ExplicitPublic {
		effectivePublic = launcherCfg.Public
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		return fmt.Errorf("invalid port %q: %w", effectivePort, err)
	}

	var addr string
	if effectivePublic {
		addr = "0.0.0.0:" + effectivePort
	} else {
		addr = "127.0.0.1:" + effectivePort
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %s is already in use — is another OctAi instance running?\n  Tip: use a different port with --port <port>, or kill the existing process:\n  lsof -i :%s", effectivePort, effectivePort)
	}
	ln.Close()

	// Initialize server components
	mux := http.NewServeMux()

	apiHandler = api.NewHandler(absPath)
	if _, err = apiHandler.EnsurePicoChannel(""); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to ensure pico channel on startup: %v", err))
	}
	apiHandler.SetServerOptions(portNum, effectivePublic, opts.ExplicitPublic, launcherCfg.AllowedCIDRs)
	apiHandler.RegisterRoutes(mux)

	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		return fmt.Errorf("invalid allowed CIDR configuration: %w", err)
	}

	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.CSRF(
				middleware.SecurityHeaders(middleware.SecurityHeadersConfig{})(
					middleware.RateLimit(60, 120)(
						middleware.JSONContentType(accessControlledMux),
					),
				),
			),
		),
	)

	if enableConsole {
		fmt.Print(utils.Banner)
		fmt.Println()
		fmt.Println("  Open the following URL in your browser:")
		fmt.Println()
		fmt.Printf("    >> http://localhost:%s <<\n", effectivePort)
		if effectivePublic {
			if ip := utils.GetLocalIP(); ip != "" {
				fmt.Printf("    >> http://%s:%s <<\n", ip, effectivePort)
			}
		}
		fmt.Println()
	}

	logger.InfoC("web", fmt.Sprintf("Server will listen on http://localhost:%s", effectivePort))
	if effectivePublic {
		if ip := utils.GetLocalIP(); ip != "" {
			logger.InfoC("web", fmt.Sprintf("Public access enabled at http://%s:%s", ip, effectivePort))
		}
	}

	serverAddr = fmt.Sprintf("http://localhost:%s", effectivePort)

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

	defer shutdownApp()

	if enableConsole {
		if !noBrowserFlag {
			if err := openBrowser(); err != nil {
				logger.Errorf("Warning: Failed to auto-open browser: %v", err)
			}
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

		for {
			sig := <-sigChan
			if sig == syscall.SIGHUP {
				logger.Info("SIGHUP received, continuing to run")
				continue
			}
			logger.Info("Shutting down...")
			return nil
		}
	}

	// GUI mode: start system tray
	runTray()
	return nil
}
