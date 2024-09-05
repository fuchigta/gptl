package internal

type Config struct {
	Provider  string `yaml:"provider"`
	Endpoint  string `yaml:"endpoint"`
	ApiKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens uint   `yaml:"max_tokens,omitempty"`
}
