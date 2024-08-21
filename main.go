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

var providers = []Provider{
	&OpenAIProvider{
		Name:     "openai",
		Models:   []string{"gpt-3.5-turbo"},
		Endpoint: "https://api.openai.com/v1",
	},
	&AzureOpenAIProvider{
		OpenAIProvider: &OpenAIProvider{
			Name:   "azure-openai",
			Models: []string{"gpt-35-turbo"},
		},
	},
	&ClaudeProvider{},
}

var configPath = filepath.Join(os.Getenv("HOME"), ".gptl", "config.yml")
var templateDirPath = filepath.Join(os.Getenv("HOME"), ".gptl", "template")
var historyDirPath = filepath.Join(os.Getenv("HOME"), ".gptl", "history")

func loadProvider() (Provider, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}
	defer f.Close()

	config := Config{
		Provider: providers[0].GetName(),
		Endpoint: providers[0].GetEndpoint(),
		ApiKey:   "",
		Model:    providers[0].GetModels()[0],
	}

	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}

	if config.Provider == "" || config.Endpoint == "" || config.ApiKey == "" || config.Model == "" {
		return nil, fmt.Errorf("config fileds required")
	}

	for _, provider := range providers {
		if provider.GetName() == config.Provider {
			provider.SetConfig(config)
			return provider, nil
		}
	}

	return nil, fmt.Errorf("provider(%s) not exists", config.Provider)
}

func exitErrBy(f string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "[ERROR] "+f+"\n", args...)
	return exitErr
}

func run() int {
	var (
		inputPath  = ""
		outputPath = ""
		tmpl       = ""
		history    = ""
	)
	flag.StringVar(&configPath, "C", configPath, "config file path")
	flag.StringVar(&templateDirPath, "T", templateDirPath, "template directory path")
	flag.StringVar(&historyDirPath, "H", historyDirPath, "history directory path")
	flag.StringVar(&inputPath, "i", inputPath, "input file path")
	flag.StringVar(&outputPath, "o", outputPath, "output file path")
	flag.StringVar(&tmpl, "t", tmpl, "request template name")
	flag.StringVar(&history, "h", history, "history name")
	flag.Parse()

	provider, err := loadProvider()
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

	if tmpl != "" {
		option = append(option, WithTemplate(tmpl))
	}

	if history != "" {
		option = append(option, WithHistoryTitle(history))
	}

	if err := provider.Chat(input, output, option...); err != nil {
		return exitErrBy(err.Error())
	}

	return exitOk
}

func main() {
	os.Exit(run())
}
