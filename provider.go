package main

import (
	"fmt"
	"io"
	"time"
)

type ChatOptions struct {
	History string
}

type ChatOption func(*ChatOptions)

func WithHistory(history string) ChatOption {
	return func(p *ChatOptions) {
		p.History = history
	}
}

func NewChatOptions(option ...ChatOption) ChatOptions {
	options := ChatOptions{
		History: time.Now().Format("20060102"),
	}

	for _, option := range option {
		option(&options)
	}

	return options
}

type Provider interface {
	Chat(input io.Reader, output io.Writer, option ...ChatOption) error
}

type ProviderFactory func(Config, HisotryRepository) (Provider, error)

var providerFactories map[string]ProviderFactory

func RegisterProviderFactory(name string, factory ProviderFactory) {
	if providerFactories == nil {
		providerFactories = map[string]ProviderFactory{}
	}
	providerFactories[name] = factory
}

func NewProvider(config Config, historyRepository HisotryRepository) (Provider, error) {
	factory, ok := providerFactories[config.Provider]
	if !ok {
		return nil, fmt.Errorf("provider(%s) not exists", config.Provider)
	}

	return factory(config, historyRepository)
}
