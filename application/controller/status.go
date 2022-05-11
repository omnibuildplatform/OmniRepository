package controller

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/omnibuildplatform/OmniRepository/app"
)

func AppHealth(c *gin.Context) {
	data := map[string]interface{}{
		"status": "UP",
		"info":   app.GitInfo,
	}
	fmt.Println("----------AppHealth-----------")
	c.JSON(200, data)
}
