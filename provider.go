package main

import (
	"io"
	"time"
)

type ChatOptions struct {
	Template string
	History  string
}

type ChatOption func(*ChatOptions)

func WithTemplate(tmpl string) ChatOption {
	return func(p *ChatOptions) {
		p.Template = tmpl
	}
}

func WithHistoryTitle(history string) ChatOption {
	return func(p *ChatOptions) {
		p.History = history
	}
}

func NewChatOptions(option ...ChatOption) ChatOptions {
	options := ChatOptions{
		Template: "",
		History:  time.Now().Format("20060102_150405"),
	}

	for _, option := range option {
		option(&options)
	}

	return options
}

type Provider interface {
	GetName() string
	GetEndpoint() string
	GetModels() []string
	SetConfig(config Config)
	Chat(input io.Reader, output io.Writer, option ...ChatOption) error
}
