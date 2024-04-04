package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"time"
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
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	termInput := make(chan string)
	go getTerminalInput(termInput)

	m, err := moonraker.NewMonitor("v400-4", "http://10.0.26.2:7125/")
	if err != nil {
		panic(err)
	}

	m.NoPauseDuration = time.Second * 30
	m.Start()

	for {
		select {
		case inputStr := <-termInput:
			if inputStr == "1" {
				fmt.Println("Setting AllowPrint to true")
				m.AllowPrint = true
			} else if inputStr == "0" {
				fmt.Println("Setting AllowPrint to false")
				m.AllowPrint = false
			}
		case s := <-interrupt:
			m.Stop()
			fmt.Println("Got signal:", s)
			return
		}
	}
}
