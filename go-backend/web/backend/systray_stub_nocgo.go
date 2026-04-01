//go:build (darwin || freebsd) && !cgo

package webconsole

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
)

// runTray falls back to a headless mode on platforms where systray requires cgo.
func runTray() {
	logger.Infof("System tray is unavailable in %s builds without cgo; running without tray", runtime.GOOS)

	if !noBrowserFlag {
		go func() {
			time.Sleep(browserDelay)
			if err := openBrowser(); err != nil {
				logger.Errorf("Warning: Failed to auto-open browser: %v", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	shutdownApp()
}
