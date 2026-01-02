package types

import (
	"time"
)

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
	GetCandles(symbol, interval string, limit int) ([]Candle, error)
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

type Backtester interface {
	RunTest(symbols []string, matcher PatternMatcher, interval string, startTime, endTime time.Time) (*BacktestResult, error)
}

type BacktestResult struct {
	TotalSignals    int             `json:"totalSignals"`
	SignalsBySymbol map[string]int  `json:"signalsBySymbol"`
	SignalsByTime   []TimeSignal    `json:"signalsByTime"`
	MissingSignals  []MissingSignal `json:"missingSignals"`
	StartTime       time.Time       `json:"startTime"`
	EndTime         time.Time       `json:"endTime"`
	Duration        time.Duration   `json:"duration"`
}

type TimeSignal struct {
	Time    time.Time `json:"time"`
	Symbol  string    `json:"symbol"`
	Pattern string    `json:"pattern"`
}

type MissingSignal struct {
	Time   time.Time `json:"time"`
	Symbol string    `json:"symbol"`
	Reason string    `json:"reason"`
}
