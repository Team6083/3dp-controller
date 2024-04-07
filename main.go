package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"v400_monitor/moonraker"
)

func getTerminalInput(input chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		input <- scanner.Text()
	}
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
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

	var monitors []*moonraker.Monitor

	ctx := context.Background()

	for _, p := range config.Printers {
		monConfig := moonraker.MonitorConfig{
			NoPauseDuration: config.NoPauseDuration,
		}

		m, err := moonraker.NewMonitor(p.Name, p.Url, monConfig, sugar.With("PrinterName", p.Name))
		if err != nil {
			panic(err)
		}

		m.Start(ctx)

		monitors = append(monitors, m)
	}

	for {
		select {
		case inputStr := <-termInput:
			fmt.Println(inputStr)
		case s := <-interrupt:
			for _, m := range monitors {
				m.Stop()
			}

			fmt.Println("Got signal:", s)
			return
		}
	}
}
