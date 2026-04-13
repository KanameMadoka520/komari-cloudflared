package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/komari-monitor/komari/api"
	"github.com/komari-monitor/komari/database/accounts"
	"github.com/komari-monitor/komari/database/auditlog"
)

func GetThemePreferences(c *gin.Context) {
	rawUUID, exists := c.Get("uuid")
	if !exists {
		api.RespondError(c, http.StatusUnauthorized, "Unauthorized.")
		return
	}

	user, err := accounts.GetUserByUUID(rawUUID.(string))
	if err != nil {
		api.RespondError(c, http.StatusInternalServerError, "Failed to fetch account theme preferences: "+err.Error())
		return
	}

	api.RespondSuccess(c, gin.H{
		"theme_appearance": user.ThemeAppearance,
		"theme_color":      user.ThemeColor,
	})
}

func SaveThemePreferences(c *gin.Context) {
	rawUUID, exists := c.Get("uuid")
	if !exists {
		api.RespondError(c, http.StatusUnauthorized, "Unauthorized.")
		return
	}

	var req struct {
		Appearance string `json:"theme_appearance" binding:"required"`
		Color      string `json:"theme_color" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "Invalid or missing request body: "+err.Error())
		return
	}

	if err := accounts.UpdateThemePreferences(rawUUID.(string), req.Appearance, req.Color); err != nil {
		api.RespondError(c, http.StatusBadRequest, "Failed to save theme preferences: "+err.Error())
		return
	}

	auditlog.Log(c.ClientIP(), rawUUID.(string), "updated account theme preferences", "info")
	api.RespondSuccess(c, gin.H{
		"theme_appearance": req.Appearance,
		"theme_color":      req.Color,
	})
}
