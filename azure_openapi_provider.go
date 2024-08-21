package main

import "net/url"

type AzureOpenAIProvider struct {
	*OpenAIProvider
}

func (p AzureOpenAIProvider) GetEndpoint() string {
	endpoint, _ := url.JoinPath(p.Endpoint, p.Config.Model)
	return endpoint
}
