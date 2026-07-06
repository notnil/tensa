package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/notnil/tensa/pkg/hware/controller"
	"github.com/notnil/tensa/pkg/hware/tensax"
)

var (
	flagConfigPath string
	flagEnableWeb  bool
	flagWebAddr    string
)

func init() {
	flag.StringVar(&flagConfigPath, "c", "cmd/tensactl/config.example.json", "path to config file")
	flag.BoolVar(&flagEnableWeb, "web", false, "enable web server for camera views")
	flag.StringVar(&flagWebAddr, "web-addr", ":8080", "address to listen on for web server")
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Println(err)
	}
}

func run() error {
	f, err := os.Open(flagConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	cfg := tensax.Config{}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	// Create a context that automatically cancels on SIGINT or SIGTERM.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	tensa, err := tensax.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create tensa: %w", err)
	}
	defer tensa.Close()
	defer tensa.Stop()

	if array := tensa.ZedArray(); array != nil {
		if err := array.Start(ctx); err != nil {
			return fmt.Errorf("failed to start camera array: %w", err)
		}
	}

	if flagEnableWeb {
		go startWebServer(ctx, tensa, flagWebAddr)
	}

	ctrl := controller.BLEController{
		DeviceName:      cfg.BLEDeviceName,
		DeviceMAC:       cfg.BLEDeviceMAC,
		Logger:          tensa.Logger(),
		DrillsDirectory: cfg.DrillsDirectory,
		Navigator:       tensa, // inject navigator implementation
		Mover:           tensa, // inject mover implementation
	}

	return ctrl.Control(ctx, tensa)
}
