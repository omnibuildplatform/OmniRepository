package main

import (
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common"
	"os"
	"os/signal"
	"syscall"

	"github.com/gookit/color"
	"github.com/omnibuildplatform/omni-repository/app"
	"github.com/omnibuildplatform/omni-repository/application"
)

var (
	manager   *application.RepositoryManager
	Tag       string //Git tag name, filled when generating binary
	CommitID  string //Git commit ID, filled when generating binary
	ReleaseAt string //Publish date, filled when generating binary
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
	var err error
	store, err := common.NewStore(&app.AppConfig.Store, app.Logger, app.TimeZone)
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to initialize database store %v\n", err))
		os.Exit(1)
	}
	imageStore := store.GetImageStorage()
	manager, err = application.NewRepositoryManager(app.AppConfig.RepoManager, application.Server().Group("/data/"), imageStore)
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to initialize repository manager %v\n", err))
		os.Exit(1)
	}
	err = manager.Initialize()
	if err != nil {
		app.Logger.Error(fmt.Sprintf("failed to start repository manager %v\n ", err))
		os.Exit(1)
	}

	color.Info.Printf("============  Begin Running(PID: %d) ============\n", os.Getpid())
	application.Run()
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

	if manager != nil {
		manager.Close()
	}
	//sleep and exit
	color.Info.Println("\nGoodBye...")

	os.Exit(0)
}
