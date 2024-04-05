package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type ConfigServer struct {
	Bind string `yaml:"bind"`
	Port int    `yaml:"port"`
}

type ConfigDisplayMessages struct {
	WillPauseMessage string `yaml:"will_pause_message"`
	PauseMessage     string `yaml:"pause_message"`
}

type RawConfig struct {
	Server          ConfigServer          `yaml:"server"`
	NoPauseDuration string                `yaml:"no_pause_duration"`
	DisplayMessages ConfigDisplayMessages `yaml:"display_messages"`
	Printers        []struct {
		Name string `yaml:"name"`
		Url  string `yaml:"url"`
		// Should be allow_print or no_print, default allow_print
		ControllerFailMode string `yaml:"controller_fail_mode"`
	} `yaml:"printers"`
}

type ControllerFailMode string

func (m ControllerFailMode) String() string {
	return string(m)
}

const (
	FailModeAllowPrint ControllerFailMode = "allow_print"
	FailModeNoPrint    ControllerFailMode = "no_print"
)

type ConfigPrinter struct {
	Name               string
	Url                string
	ControllerFailMode ControllerFailMode
}

type Config struct {
	Server          ConfigServer
	NoPauseDuration time.Duration
	DisplayMessages ConfigDisplayMessages
	Printers        []ConfigPrinter
}

func LoadConfig(fileName string) (*Config, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	var rawCfg RawConfig
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&rawCfg)
	if err != nil {
		panic(err)
	}

	return ParseRawConfig(rawCfg)
}

func ParseRawConfig(raw RawConfig) (*Config, error) {
	var cfg Config
	cfg.Server = raw.Server
	cfg.DisplayMessages = raw.DisplayMessages

	noPauseDuration, err := time.ParseDuration(raw.NoPauseDuration)
	if err != nil {
		return nil, err
	}
	cfg.NoPauseDuration = noPauseDuration

	for _, rp := range raw.Printers {
		p := ConfigPrinter{
			Name: rp.Name,
			Url:  rp.Url,
		}

		switch rp.ControllerFailMode {
		case FailModeNoPrint.String():
			p.ControllerFailMode = FailModeNoPrint
		case FailModeAllowPrint.String(), "":
			p.ControllerFailMode = FailModeAllowPrint
		default:
			return nil, errors.New(fmt.Sprintf("Unknown ControllerFailMode %s", rp.ControllerFailMode))
		}

		cfg.Printers = append(cfg.Printers, p)
	}

	return &cfg, nil
}
