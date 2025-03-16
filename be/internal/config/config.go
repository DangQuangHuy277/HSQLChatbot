package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	CORS     CORSConfig     `mapstructure:"cors"`
	OpenAI   OpenAIConfig   `mapstructure:"openai"`
	GeminiAI GeminiAIConfig `mapstructure:"gemini"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	SerpApi  SerpApiConfig  `mapstructure:"serpapi"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type GeminiAIConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type JWTConfig struct {
	SecretKey   string        `mapstructure:"secret_key"`
	ExpiryHours time.Duration `mapstructure:"expiry_hours"`
}

type SerpApiConfig struct {
	APIKey string `mapstructure:"api_key"`
}

func LoadConfig(configPath string, envPath string) (*Config, error) {
	// Load .env file first
	if err := godotenv.Load(envPath); err != nil {
		return nil, err
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// Process the config to expand environment variables
	for _, key := range viper.AllKeys() {
		val := viper.GetString(key)
		if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
			envVar := strings.TrimSuffix(strings.TrimPrefix(val, "${"), "}")
			envVal := os.Getenv(envVar)
			if envVal != "" {
				viper.Set(key, envVal)
			}
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
