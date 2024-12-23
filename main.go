package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"v400_monitor/controller"
	"v400_monitor/moonraker"
	"v400_monitor/web"
)

func getTerminalInput(input chan string) {
	_, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		input <- scanner.Text()
	}
}

func getLogger(isDevMode bool) *zap.Logger {
	if isDevMode {
		logger, err := zap.NewDevelopment()
		if err != nil {
			panic(err)
		}

		return logger
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	return logger
}

func main() {
	isDevMode := len(os.Getenv("dev")) != 0

	logger := getLogger(isDevMode)
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil && !errors.Is(err, syscall.ENOTTY) {
			panic(err)
		}
	}(logger)

	sugar := logger.Sugar()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	termInput := make(chan string)
	go getTerminalInput(termInput)

	config, err := LoadConfig("./config.yaml")
	if err != nil {
		panic(err)
	}

	monitors := make(map[string]*moonraker.Monitor)

	ctx := context.Background()

	for _, p := range config.Printers {
		monConfig := moonraker.MonitorConfig{
			NoPauseDuration:      config.NoPauseDuration,
			ShouldPauseProgress:  config.ShouldPauseProgress,
			ShouldCancelProgress: config.ShouldCancelProgress,
			WillPauseMessage:     config.DisplayMessages.WillPauseMessage,
			PauseMessage:         config.DisplayMessages.PauseMessage,
		}

		m, err := moonraker.NewMonitor(p.Name, p.Url, monConfig, sugar.With("PrinterName", p.Name))
		if err != nil {
			panic(err)
		}

		m.SetAllowNoRegPrint(p.ControllerFailMode != FailModeNoPrint)

		m.Start(ctx)

		monitors[p.Key] = m
	}

	var ctrlConnector *controller.Connector
	if config.Controller.Url != nil {
		ctrlConnector = controller.NewConnector(config.Controller.Url, config.Controller.HubId,
			sugar.Named("controller"), monitors)
		ctrlConnector.Connect(ctx)
	}

	server := web.NewServer(ctx, isDevMode, sugar.Named("web"), monitors)
	go server.Run()

	for {
		select {
		case inputStr := <-termInput:
			input := strings.Split(inputStr, " ")
			if len(input) < 1 {
				fmt.Println("Usage: <printer_idx> [<job_id>]")
			} else {
				key := input[0]

				if _, ok := monitors[key]; ok {
					if len(input) == 1 {
						monitors[key].SetRegisteredJobId("")
					} else {
						monitors[key].SetRegisteredJobId(input[1])
					}
				} else {
					fmt.Println("Error: Printer not found!!")
				}
			}
		case s := <-interrupt:
			go server.Shutdown()

			if ctrlConnector != nil {
				go ctrlConnector.Close()
			}

			for _, m := range monitors {
				m.Stop()
			}

			fmt.Println("Got signal:", s)
			return
		}
	}
}
