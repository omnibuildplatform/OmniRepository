package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gookit/color"
	"github.com/omnibuildplatform/OmniRepository/app"
	"github.com/omnibuildplatform/OmniRepository/application"
)

var (
	manager *application.RepositoryManager
)

func init() {

	app.Bootstrap("./config")
	application.InitServer()
}
func main() {
	listenSignals()
	var err error
	manager, err = application.NewRepositoryManager(application.Server().Group("/data/"))
	if err != nil {
		color.Error.Printf("failed to initialize repository manager %v\n", err)
		os.Exit(1)
	}
	err = manager.Initialize()
	if err != nil {
		color.Error.Printf("failed to start repository manager %v\n ", err)
		os.Exit(1)
	}
	manager.StartLoop()
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
	time.Sleep(time.Second * 3)

	os.Exit(0)
}
