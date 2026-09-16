package server

import (
	"context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/internal/config"
	"github.com/komari-monitor/komari/utils"
	"github.com/komari-monitor/komari/utils/cloudflared"
	logger "github.com/komari-monitor/komari/utils/log"
)

// Bootstrap initializes the data directory, primary database, and settings.
func (a *App) Bootstrap() error {
	if err := os.MkdirAll("./data/theme", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create theme directory: %w", err)
	}
	if err := os.MkdirAll("./data/plugin", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}
	if err := os.MkdirAll("./data/plugin-data", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create plugin storage directory: %w", err)
	}

	dbcore.SetVersionID(utils.CurrentVersion + "-" + utils.VersionHash)
	if err := dbcore.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	a.dbReady = true
	a.addCleanup("database", func(context.Context) error { return dbcore.Close() })

	gin.SetMode(gin.ReleaseMode)
	settings, err := config.GetManyAs[config.Settings]()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	a.settings = settings
	a.addCleanup("cloudflared", func(context.Context) error { cloudflared.Shutdown(); return nil })
	if err := cloudflared.AutoStart(os.Getenv("KOMARI_CLOUDFLARED_TOKEN")); err != nil {
		logger.Errorf("cloudflared", "Auto-start failed: %v", err)
	}
	return nil
}
