package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"v400_monitor/moonraker"
	"v400_monitor/web"
)

func getTerminalInput(input chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		input <- scanner.Text()
	}
}

func main() {
	var logger *zap.Logger

	if len(os.Getenv("dev")) != 0 {
		var err error
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
	} else {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}

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
			NoPauseDuration:  config.NoPauseDuration,
			WillPauseMessage: config.DisplayMessages.WillPauseMessage,
			PauseMessage:     config.DisplayMessages.PauseMessage,
		}

		m, err := moonraker.NewMonitor(p.Name, p.Url, monConfig, sugar.With("PrinterName", p.Name))
		if err != nil {
			panic(err)
		}

		m.SetAllowNoRegPrint(p.ControllerFailMode != FailModeNoPrint)

		m.Start(ctx)

		monitors[p.Key] = m
	}

	server := web.NewServer(ctx, sugar.Named("web"), monitors)
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

			for _, m := range monitors {
				m.Stop()
			}

			fmt.Println("Got signal:", s)
			return
		}
	}
}
