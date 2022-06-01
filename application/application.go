package application

import (
	"context"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omnibuildplatform/omni-repository/app"
)

var publicEngine *gin.Engine
var internalEngine *gin.Engine
var publicHttpServer *http.Server
var internalHttpServer *http.Server

func PublicEngine() *gin.Engine {
	return publicEngine
}

func InternalEngine() *gin.Engine {
	return internalEngine
}

func InitServer() {
	publicEngine = gin.New()
	skipPaths := []string{"/health"}
	if app.EnvName == app.EnvDev {
		publicEngine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: skipPaths,
		}), gin.Recovery())
	} else {
		publicEngine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
			SkipPaths: skipPaths,
		}))
	}

	internalEngine = gin.New()
	if app.EnvName == app.EnvDev {
		internalEngine.Use(gin.Logger(), gin.Recovery())
	} else {
		publicEngine.Use(gin.Logger())
	}
	AddRoutes(publicEngine)
}

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := publicHttpServer.Shutdown(ctx); err != nil {
		app.Logger.Error(fmt.Sprintf("failed to close public server %v", err))
	}
	if err := internalHttpServer.Shutdown(ctx); err != nil {
		app.Logger.Error(fmt.Sprintf("failed to close internal server %v", err))
	}
}

func Run(config config.ServerConfig) {
	wg := sync.WaitGroup{}

	publicHttpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.PublicHttpPort),
		Handler:      PublicEngine(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	internalHttpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.InternalHttpPort),
		Handler:      InternalEngine(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	wg.Add(1)
	go func() {
		app.Logger.Info(fmt.Sprintf("public server starts up with port %d", config.PublicHttpPort))
		publicHttpServer.ListenAndServe()
		defer wg.Done()
	}()
	wg.Add(1)
	go func() {
		app.Logger.Info(fmt.Sprintf("internal server starts up with port %d", config.InternalHttpPort))
		internalHttpServer.ListenAndServe()
		defer wg.Done()
	}()
	wg.Wait()
}
