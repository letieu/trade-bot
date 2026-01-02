package bybit

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/types"
)

type Client struct {
	config *config.BybitConfig
	client *http.Client
}

type InstrumentsResponse struct {
	RetCode int `json:"retCode"`
	Result  struct {
		List []Instrument `json:"list"`
	} `json:"result"`
}

type Instrument struct {
	Symbol    string `json:"symbol"`
	Status    string `json:"status"`
	BaseCoin  string `json:"baseCoin"`
	QuoteCoin string `json:"quoteCoin"`
}

type KlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List [][]string `json:"list"`
	} `json:"result"`
}

type TickersResponse struct {
	RetCode int `json:"retCode"`
	Result  struct {
		List []Ticker `json:"list"`
	} `json:"result"`
}

type Ticker struct {
	Symbol       string `json:"symbol"`
	LastPrice    string `json:"lastPrice"`
	PrevPrice24h string `json:"prevPrice24h"`
	Volume24h    string `json:"volume24h"`
	Turnover24h  string `json:"turnover24h"`
}

func NewClient(cfg *config.BybitConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) GetSymbols() ([]string, error) {
	url := fmt.Sprintf("%s/v5/market/instruments-info?category=linear", c.config.BaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var instrumentsResp InstrumentsResponse
	if err := json.Unmarshal(body, &instrumentsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if instrumentsResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: retCode=%d", instrumentsResp.RetCode)
	}

	var symbols []string
	for _, instrument := range instrumentsResp.Result.List {
		if instrument.Status == "Trading" && strings.HasSuffix(instrument.Symbol, "USDT") {
			symbols = append(symbols, instrument.Symbol)
		}
	}

	log.Printf("Retrieved %d symbols from Bybit", len(symbols))
	return symbols, nil
}

func (c *Client) GetCandles(symbol, interval string, limit int) ([]types.Candle, error) {
	bybitInterval := mapIntervalToBybit(interval)
	url := fmt.Sprintf("%s/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d",
		c.config.BaseURL, symbol, bybitInterval, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var klineResp KlineResponse
	if err := json.Unmarshal(body, &klineResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if klineResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: retCode=%d, msg=%s", klineResp.RetCode, klineResp.RetMsg)
	}

	var candles []types.Candle
	for _, candleData := range klineResp.Result.List {
		if len(candleData) < 6 {
			continue
		}

		timestamp, err := strconv.ParseInt(candleData[0], 10, 64)
		if err != nil {
			log.Printf("Failed to parse timestamp for %s: %v", symbol, err)
			continue
		}

		open, err := strconv.ParseFloat(candleData[1], 64)
		if err != nil {
			log.Printf("Failed to parse open price for %s: %v", symbol, err)
			continue
		}

		high, err := strconv.ParseFloat(candleData[2], 64)
		if err != nil {
			log.Printf("Failed to parse high price for %s: %v", symbol, err)
			continue
		}

		low, err := strconv.ParseFloat(candleData[3], 64)
		if err != nil {
			log.Printf("Failed to parse low price for %s: %v", symbol, err)
			continue
		}

		close, err := strconv.ParseFloat(candleData[4], 64)
		if err != nil {
			log.Printf("Failed to parse close price for %s: %v", symbol, err)
			continue
		}

		volume, err := strconv.ParseFloat(candleData[5], 64)
		if err != nil {
			log.Printf("Failed to parse volume for %s: %v", symbol, err)
			continue
		}

		candle := types.Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Symbol:    symbol,
			Interval:  interval,
		}

		candles = append(candles, candle)
	}

	// Reverse candles to be chronological (Oldest First)
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}

	return candles, nil
}

func mapIntervalToBybit(interval string) string {
	switch interval {
	case "1m":
		return "1"
	case "3m":
		return "3"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h":
		return "60"
	case "2h":
		return "120"
	case "4h":
		return "240"
	case "6h":
		return "360"
	case "12h":
		return "720"
	case "1d":
		return "D"
	case "1w":
		return "W"
	case "1M":
		return "M"
	default:
		return interval
	}
}

func (c *Client) GetTickerInfo(symbol string) (types.TickerInfo, error) {
	url := fmt.Sprintf("%s/v5/market/tickers?category=linear&symbol=%s", c.config.BaseURL, symbol)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return types.TickerInfo{}, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return types.TickerInfo{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.TickerInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var tickersResp TickersResponse
	if err := json.Unmarshal(body, &tickersResp); err != nil {
		return types.TickerInfo{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if tickersResp.RetCode != 0 {
		return types.TickerInfo{}, fmt.Errorf("API error: retCode=%d", tickersResp.RetCode)
	}

	if len(tickersResp.Result.List) == 0 {
		return types.TickerInfo{}, fmt.Errorf("no ticker data found for symbol %s", symbol)
	}

	ticker := tickersResp.Result.List[0]

	lastPrice, err := strconv.ParseFloat(ticker.LastPrice, 64)
	if err != nil {
		return types.TickerInfo{}, fmt.Errorf("failed to parse last price: %w", err)
	}

	prevPrice24h, err := strconv.ParseFloat(ticker.PrevPrice24h, 64)
	if err != nil {
		log.Printf("Failed to parse prev price 24h for %s: %v", symbol, err)
		prevPrice24h = 0
	}

	volume24h, err := strconv.ParseFloat(ticker.Volume24h, 64)
	if err != nil {
		log.Printf("Failed to parse volume 24h for %s: %v", symbol, err)
		volume24h = 0
	}

	turnover24h, err := strconv.ParseFloat(ticker.Turnover24h, 64)
	if err != nil {
		log.Printf("Failed to parse turnover 24h for %s: %v", symbol, err)
		turnover24h = 0
	}

	return types.TickerInfo{
		Symbol:       ticker.Symbol,
		LastPrice:    lastPrice,
		PrevPrice24h: prevPrice24h,
		Volume24h:    volume24h,
		Turnover24h:  turnover24h,
	}, nil
}
