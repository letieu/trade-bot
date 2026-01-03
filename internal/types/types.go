package types

import (
	"fmt"
	"time"
)

func ParseInterval(interval string) (time.Duration, error) {
	switch interval {
	case "1m":
		return time.Minute, nil
	case "3m":
		return 3 * time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "30m":
		return 30 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "2h":
		return 2 * time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "6h":
		return 6 * time.Hour, nil
	case "12h":
		return 12 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported interval: %s", interval)
	}
}

type Candle struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	Symbol    string  `json:"symbol"`
	Interval  string  `json:"interval"`
}

type CandleColor string

const (
	ColorGreen CandleColor = "green"
	ColorRed   CandleColor = "red"
)

func (c *Candle) Color() CandleColor {
	if c.Close >= c.Open {
		return ColorGreen
	}
	return ColorRed
}

type Signal struct {
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	Pattern   string    `json:"pattern"`
	Trend     string    `json:"trend"`
	Price     float64   `json:"price"`
	RSI       float64   `json:"rsi"`
	EMA       float64   `json:"ema"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Candles   []Candle  `json:"candles"`
}

type MarketDataProvider interface {
	GetSymbols() ([]string, error)
	GetCandles(symbol, interval string, limit int, endTime int64) ([]Candle, error)
	GetTickerInfo(symbol string) (TickerInfo, error)
}

type TickerInfo struct {
	Symbol       string  `json:"symbol"`
	LastPrice    float64 `json:"lastPrice"`
	PrevPrice24h float64 `json:"prevPrice24h"`
	Volume24h    float64 `json:"volume24h"`
	Turnover24h  float64 `json:"turnover24h"`
}

type PatternMatcher interface {
	Match(candles []Candle) (bool, error)
	GetName() string
	GetDescription() string
	GetRequiredCandles() int
}

type NotificationSender interface {
	SendSignals(signals []Signal) error
	SendMessage(message string) error
}
