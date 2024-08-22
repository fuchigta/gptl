package main

import "net/url"

type AzureOpenAIProvider struct {
	*OpenAIProvider
}

func (p AzureOpenAIProvider) GetEndpoint() string {
	endpoint, _ := url.JoinPath(p.config.Endpoint, p.config.Model)
	return endpoint
}
