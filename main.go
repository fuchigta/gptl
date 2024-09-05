package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	gptl "github.com/fuchigta/gptl/internal"
	_ "github.com/fuchigta/gptl/internal/provider"
	"golang.org/x/term"

	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v2"
)

const (
	exitOk = iota
	exitErr
)

func loadProvider(configPath string, historyDirPath string) (gptl.Provider, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}
	defer f.Close()

	var config gptl.Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("can't load config[%s]: %s", configPath, err)
	}

	historyRepository, _ := gptl.NewHistoryRepository(historyDirPath)

	return gptl.NewProvider(config, historyRepository)
}

func exitErrBy(f string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "[ERROR] "+f+"\n", args...)
	return exitErr
}

func doInit(configPath string) int {
	dir := filepath.Dir(configPath)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return exitErrBy("%s: error: %s", dir, err.Error())
		}

		if !dirInfo.IsDir() {
			return exitErrBy("%s is not directory", dir)
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return exitErrBy("%s: mkdir error: %s", dir, err.Error())
		}
	}

	info, err := os.Stat(configPath)
	if err == nil {
		if info.IsDir() {
			return exitErrBy("%s is directory", configPath)
		}

		return exitOk
	}

	config := gptl.Config{}

	providerSelect := promptui.Select{
		Label:        "provider",
		Items:        gptl.Providers(),
		HideSelected: true,
	}

	_, provider, err := providerSelect.Run()
	if err != nil {
		return exitErrBy(err.Error())
	}

	config.Provider = provider

	endpointPrompt := promptui.Prompt{
		Label:       "endpoint",
		HideEntered: true,
	}

	endpoint, err := endpointPrompt.Run()
	if err != nil {
		return exitErrBy(err.Error())
	}

	config.Endpoint = endpoint

	apiKeyPrompt := promptui.Prompt{
		Label:       "api_key",
		HideEntered: true,
	}

	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return exitErrBy(err.Error())
	}

	config.ApiKey = apiKey

	modelPrompt := promptui.Prompt{
		Label:       "model",
		HideEntered: true,
	}

	model, err := modelPrompt.Run()
	if err != nil {
		return exitErrBy(err.Error())
	}

	config.Model = model

	content, _ := yaml.Marshal(config)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		return exitErrBy(err.Error())
	}

	fmt.Printf("%s created\n", configPath)

	return exitOk
}

func run() int {
	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}

	var (
		historyDirPath = filepath.Join(home, ".gptl", "history")
		configPath     = filepath.Join(home, ".gptl", "config.yaml")
		inputPath      = ""
		outputPath     = ""
		history        = ""
		init           = false
	)
	flag.StringVar(&configPath, "C", configPath, "config file path")
	flag.StringVar(&historyDirPath, "H", historyDirPath, "history directory path")
	flag.StringVar(&inputPath, "i", inputPath, "input file path")
	flag.StringVar(&outputPath, "o", outputPath, "output file path")
	flag.StringVar(&history, "h", history, "history name")
	flag.BoolVar(&init, "init", init, "init config file")
	flag.Parse()

	if init {
		return doInit(configPath)
	}

	provider, err := loadProvider(configPath, historyDirPath)
	if err != nil {
		return exitErrBy(err.Error())
	}

	var input io.Reader
	if inputPath == "" {
		if flag.NArg() != 0 {
			buffer := bytes.Buffer{}
			for _, arg := range flag.Args() {
				buffer.WriteString(arg + "\n")
			}
			input = &buffer
		} else {
			if term.IsTerminal(int(os.Stdin.Fd())) {
				messagePrompt := promptui.Prompt{
					Label: "message",
				}

				message, err := messagePrompt.Run()
				if err != nil {
					return exitErrBy(err.Error())
				}

				input = bytes.NewBufferString(message)
			} else {
				input = os.Stdin
			}
		}
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

	option := []gptl.ChatOption{}

	if history != "" {
		option = append(option, gptl.WithHistory(history))
	}

	if err := provider.Chat(input, output, option...); err != nil {
		return exitErrBy(err.Error())
	}

	return exitOk
}

func main() {
	os.Exit(run())
}
