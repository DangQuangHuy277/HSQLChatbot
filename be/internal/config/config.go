package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	CORS     CORSConfig     `yaml:"cors"`
	OpenAI   OpenAIConfig   `yaml:"openai"`
	GeminiAI GeminiAIConfig `yaml:"gemini_ai"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type CORSConfig struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	ExposeHeaders    []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
}

type GeminiAIConfig struct {
	APIKey string `yaml:"api_key"`
}

type JWTConfig struct {
	SecretKey   string        `yaml:"secret_key"`
	ExpiryHours time.Duration `yaml:"expiry_hours" default:"24"`
}

func LoadConfig(configPath string, envPath string) (*Config, error) {
	// Load .env file first
	if err := godotenv.Load(envPath); err != nil {
		return nil, err
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
