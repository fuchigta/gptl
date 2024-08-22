package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	exitOk = iota
	exitErr
)

type Config struct {
	Provider  string `yaml:"provider"`
	Endpoint  string `yaml:"endpoint"`
	ApiKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens uint   `yaml:"max_tokens"`
}

func loadProvider(configPath string, historyDirPath string) (Provider, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}
	defer f.Close()

	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}

	historyRepository, _ := NewHistoryRepository(historyDirPath)

	return NewProvider(config, historyRepository)
}

func exitErrBy(f string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "[ERROR] "+f+"\n", args...)
	return exitErr
}

func run() int {
	var (
		historyDirPath = filepath.Join(os.Getenv("HOME"), ".gptl", "history")
		configPath     = filepath.Join(os.Getenv("HOME"), ".gptl", "config.yaml")
		inputPath      = ""
		outputPath     = ""
		history        = ""
	)
	flag.StringVar(&configPath, "C", configPath, "config file path")
	flag.StringVar(&historyDirPath, "H", historyDirPath, "history directory path")
	flag.StringVar(&inputPath, "i", inputPath, "input file path")
	flag.StringVar(&outputPath, "o", outputPath, "output file path")
	flag.StringVar(&history, "h", history, "history name")
	flag.Parse()

	provider, err := loadProvider(configPath, historyDirPath)
	if err != nil {
		return exitErrBy(err.Error())
	}

	var input io.Reader
	if inputPath == "" {
		input = os.Stdin
	} else {
		f, err := os.Open(inputPath)
		if err != nil {
			return exitErrBy(err.Error())
		}
		defer f.Close()

		input = f
	}

	var output io.Writer
	if outputPath == "" {
		output = os.Stdout
	} else {
		f, err := os.Create(outputPath)
		if err != nil {
			return exitErrBy(err.Error())
		}
		defer f.Close()

		output = f
	}

	option := []ChatOption{}

	if history != "" {
		option = append(option, WithHistory(history))
	}

	if err := provider.Chat(input, output, option...); err != nil {
		return exitErrBy(err.Error())
	}

	return exitOk
}

func main() {
	os.Exit(run())
}
