package application

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gookit/color"
	"github.com/omnibuildplatform/omni-repository/app"
)

var server *gin.Engine

func Server() *gin.Engine {
	return server
}

func InitServer() {
	server = gin.New()
	skipPaths := []string{"/health"}
	if app.EnvName == app.EnvDev {
		server.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: skipPaths,
		}), gin.Recovery())
	} else {
		server.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: skipPaths,
		}))
	}

	AddRoutes(server)

}

func Run() {
	//NOTE: application will use loopback address 127.0.0.1 for internal usage, please don't remove 127.0.0.1 address
	err := server.Run(fmt.Sprintf("0.0.0.0:%d", app.HttpPort))
	if err != nil {
		color.Error.Println(err)
	}
}
