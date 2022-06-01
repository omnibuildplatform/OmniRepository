package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/omnibuildplatform/omni-repository/app"
)

func AppHealth(c *gin.Context) {
	data := map[string]interface{}{
		"status": "UP",
		"info":   app.Info,
	}
	c.JSON(200, data)
}
