package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Telegram TelegramConfig `mapstructure:"telegram"`
	Bybit    BybitConfig    `mapstructure:"bybit"`
	Bot      BotConfig      `mapstructure:"bot"`
	Backtest BacktestConfig `mapstructure:"backtest"`
}

type TelegramConfig struct {
	BotToken string `mapstructure:"botToken"`
	ChatID   string `mapstructure:"chatId"`
}

type BybitConfig struct {
	BaseURL   string            `mapstructure:"baseUrl"`
	Timeout   time.Duration     `mapstructure:"timeout"`
	RateLimit int               `mapstructure:"rateLimit"`
	Headers   map[string]string `mapstructure:"headers"`
}

type BotConfig struct {
	ScanInterval     time.Duration `mapstructure:"scanInterval"`
	BatchSize        int           `mapstructure:"batchSize"`
	MaxConcurrency   int           `mapstructure:"maxConcurrency"`
	EnabledIntervals []string      `mapstructure:"enabledIntervals"`
	Frontend         string        `mapstructure:"frontend"`
	RunOnce          bool          `mapstructure:"runOnce"`
}

type BacktestConfig struct {
	StartTime   string `mapstructure:"startTime"`
	EndTime     string `mapstructure:"endTime"`
	DataPath    string `mapstructure:"dataPath"`
	SaveResults bool   `mapstructure:"saveResults"`
	ResultsPath string `mapstructure:"resultsPath"`
}

func Load(configFile string) *Config {
	v := viper.New()

	// Set defaults for telegram config
	v.SetDefault("telegram.botToken", "")
	v.SetDefault("telegram.chatId", "")

	// Set defaults for bybit config
	v.SetDefault("bybit.baseUrl", "https://api.bybit.com")
	v.SetDefault("bybit.timeout", "10s")
	v.SetDefault("bybit.rateLimit", 20)
	v.SetDefault("bybit.headers", map[string]interface{}{
		"Content-Type": "application/json",
	})

	// Set defaults for bot config
	v.SetDefault("bot.scanInterval", "1m")
	v.SetDefault("bot.batchSize", 20)
	v.SetDefault("bot.maxConcurrency", 5)
	v.SetDefault("bot.enabledIntervals", []string{"1h", "4h", "1d"})
	v.SetDefault("bot.frontend", "telegram")

	// If config file is specified, load it and prioritize it
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			fmt.Printf("Failed to read config file %s: %v\n", configFile, err)
			os.Exit(1)
		}
	} else {
		// Try to find default config file
		v.SetConfigName("trade-bot")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if err := v.ReadInConfig(); err != nil {
			// No config file found, continue with defaults and env vars
			fmt.Printf("No config file found, using defaults and environment variables\n")
			v.AutomaticEnv()
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		fmt.Printf("Failed to unmarshal config: %v\n", err)
		os.Exit(1)
	}

	return &cfg
}

func (c *Config) SaveToFile(filePath string) error {
	return viper.WriteConfigAs(filePath)
}

func (c *Config) GetBacktestTimes() (startTime, endTime time.Time, err error) {
	startTime, err = time.Parse(time.RFC3339, c.Backtest.StartTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse startTime: %w", err)
	}

	endTime, err = time.Parse(time.RFC3339, c.Backtest.EndTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to parse endTime: %w", err)
	}

	return startTime, endTime, nil
}
