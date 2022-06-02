package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/omnibuildplatform/omni-repository/common"

	"github.com/gookit/color"
	"github.com/omnibuildplatform/omni-repository/app"
	"github.com/omnibuildplatform/omni-repository/application"
)

type CancelContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	repoManager   *application.RepositoryManager
	workManager   *application.WorkManager
	store         *common.Store
	globalContext *CancelContext
	Tag           string //Git tag name, filled when generating binary
	CommitID      string //Git commit ID, filled when generating binary
	ReleaseAt     string //Publish date, filled when generating binary
)

func init() {

	app.Bootstrap("./config", Tag, CommitID, ReleaseAt)
	application.InitServer()
}

func printVersion() {
	app.Logger.Info("============ Release Info ============")
	app.Logger.Info(fmt.Sprintf("Git Tag: %s", app.Info.Tag))
	app.Logger.Info(fmt.Sprintf("Git CommitID: %s", app.Info.CommitID))
	app.Logger.Info(fmt.Sprintf("Released At: %s", app.Info.ReleaseAt))
}

func main() {
	printVersion()
	listenSignals()
	ctx, cancel := context.WithCancel(context.TODO())
	globalContext = &CancelContext{
		ctx:    ctx,
		cancel: cancel,
	}
	var err error
	store, err = common.NewStore(&app.AppConfig.Store, app.Logger)
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to initialize database store %v", err))
		os.Exit(1)
	}
	imageStore := store.GetImageStorage(globalContext.ctx)
	repoManager, err = application.NewRepositoryManager(
		globalContext.ctx,
		app.AppConfig.RepoManager,
		application.PublicEngine().Group("/"),
		application.InternalEngine().Group("/"),
		imageStore,
		app.AppConfig.ServerConfig.DataFolder, app.Logger)
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to initialize repository manager %v", err))
		os.Exit(1)
	}
	err = repoManager.Initialize()
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to start repository manager %v", err))
		os.Exit(1)
	}
	app.Logger.Info("repo manager fully start up")
	workManager, err := application.NewWorkManager(
		globalContext.ctx,
		app.AppConfig.WorkManager,
		app.Logger,
		imageStore,
		app.AppConfig.ServerConfig.DataFolder)
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to start work manager %v", err))
		os.Exit(1)
	}
	go app.InitMQ()
	go workManager.StartLoop()
	app.Logger.Info("work manager fully start up")
	app.Logger.Info(fmt.Sprintf("============  Begin Running(PID: %d) ============", os.Getpid()))
	application.Run(app.AppConfig.ServerConfig)
}

// listenSignals Graceful start/stop server
func listenSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go handleSignals(sigChan)
}

// handleSignals handle process signal
func handleSignals(c chan os.Signal) {
	color.Info.Printf("Notice: System signal monitoring is enabled(watch: SIGINT,SIGTERM,SIGQUIT)\n")

	switch <-c {
	case syscall.SIGINT:
		color.Info.Printf("\nShutdown by Ctrl+C")
	case syscall.SIGTERM: // by kill
		color.Info.Printf("\nShutdown quickly")
	case syscall.SIGQUIT:
		color.Info.Printf("\nShutdown gracefully")
		// do graceful shutdown
	}

	// sync logs
	_ = app.Logger.Sync()

	if globalContext != nil {
		globalContext.cancel()
	}

	if repoManager != nil {
		repoManager.Close()
	}
	if workManager != nil {
		workManager.Close()
	}
	if store != nil {
		store.Close()
	}
	time.Sleep(3 * time.Second)
	application.Close()
	//sleep and exit
	color.Info.Println("\nGoodBye...")

	os.Exit(0)
}
