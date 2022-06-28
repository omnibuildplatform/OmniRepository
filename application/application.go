package application

import (
	"context"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omnibuildplatform/omni-repository/app"
)

const MaxBodySize = 1024 * 1024 * 1024 * 20 //20GB

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

func MaxSizeLimitationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var w http.ResponseWriter = c.Writer
		c.Request.Body = http.MaxBytesReader(w, c.Request.Body, MaxBodySize)
		c.Next()
	}
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
	AddRoutes(publicEngine)

	internalEngine = gin.New()
	if app.EnvName == app.EnvDev {
		internalEngine.Use(gin.Logger(), gin.Recovery(), MaxSizeLimitationMiddleware())
	} else {
		internalEngine.Use(gin.Logger(), MaxSizeLimitationMiddleware())
	}
}

func Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if publicHttpServer != nil {
		if err := publicHttpServer.Shutdown(ctx); err != nil {
			app.Logger.Error(fmt.Sprintf("failed to close public server %v", err))
		}
	}
	if internalHttpServer != nil {
		if err := internalHttpServer.Shutdown(ctx); err != nil {
			app.Logger.Error(fmt.Sprintf("failed to close internal server %v", err))
		}
	}
}

func Run(config config.ServerConfig) {
	wg := sync.WaitGroup{}

	publicHttpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.PublicHttpPort),
		Handler:      PublicEngine(),
		ReadTimeout:  60 * 10 * time.Second,
		WriteTimeout: 60 * 10 * time.Second,
	}

	internalHttpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.InternalHttpPort),
		Handler:      InternalEngine(),
		ReadTimeout:  60 * 10 * time.Second,
		WriteTimeout: 60 * 10 * time.Second,
	}
	wg.Add(1)
	go func() {
		app.Logger.Info(fmt.Sprintf("public server starts up with port %d", config.PublicHttpPort))
		err := publicHttpServer.ListenAndServe()
		if err != nil {
			app.Logger.Error(fmt.Sprintf("unable to start public http server %s", err))
			os.Exit(1)
		}
		defer wg.Done()
	}()
	wg.Add(1)
	go func() {
		app.Logger.Info(fmt.Sprintf("internal server starts up with port %d", config.InternalHttpPort))
		err := internalHttpServer.ListenAndServe()
		if err != nil {
			app.Logger.Error(fmt.Sprintf("unable to start internal http server %s", err))
			os.Exit(1)
		}
		defer wg.Done()
	}()
	wg.Wait()
}
