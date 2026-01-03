package bybit

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/letieu/trade-bot/internal/config"
	"github.com/letieu/trade-bot/internal/types"
)

type Client struct {
	config            *config.BybitConfig
	client            *http.Client
	cachedSymbols     []string
	lastSymbolsUpdate time.Time
	mu                sync.RWMutex
}

type InstrumentsResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List           []Instrument `json:"list"`
		NextPageCursor string       `json:"nextPageCursor"`
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

func NewClient(cfg *config.BybitConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) GetSymbols() ([]string, error) {
	c.mu.RLock()
	if len(c.cachedSymbols) > 0 && time.Since(c.lastSymbolsUpdate) < 24*time.Hour {
		defer c.mu.RUnlock()
		return c.cachedSymbols, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if len(c.cachedSymbols) > 0 && time.Since(c.lastSymbolsUpdate) < 24*time.Hour {
		return c.cachedSymbols, nil
	}

	var symbols []string
	cursor := ""

	for {
		url := fmt.Sprintf("%s/v5/market/instruments-info?category=linear&limit=1000", c.config.BaseURL)
		if cursor != "" {
			url = fmt.Sprintf("%s&cursor=%s", url, cursor)
		}

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

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var instrumentsResp InstrumentsResponse
		if err := json.Unmarshal(body, &instrumentsResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		if instrumentsResp.RetCode != 0 {
			return nil, fmt.Errorf("API error: retCode=%d, msg=%s", instrumentsResp.RetCode, instrumentsResp.RetMsg)
		}

		for _, instrument := range instrumentsResp.Result.List {
			if instrument.Status == "Trading" && strings.HasSuffix(instrument.Symbol, "USDT") {
				symbols = append(symbols, instrument.Symbol)
			}
		}

		cursor = instrumentsResp.Result.NextPageCursor
		if cursor == "" {
			break
		}

		// Avoid hitting rate limits during pagination
		time.Sleep(100 * time.Millisecond)
	}

	c.cachedSymbols = symbols
	c.lastSymbolsUpdate = time.Now()

	log.Printf("Retrieved %d symbols from Bybit", len(symbols))
	return symbols, nil
}

func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		resp, err = c.client.Do(req)
		if err == nil {
			return resp, nil
		}

		// Only retry on network errors or timeouts
		log.Printf("Request failed (attempt %d/%d): %v. Retrying in 2s...", i+1, maxRetries, err)
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries, err)
}

func (c *Client) GetCandles(symbol, interval string, limit int, endTime int64) ([]types.Candle, error) {
	bybitInterval := mapIntervalToBybit(interval)
	url := fmt.Sprintf("%s/v5/market/kline?category=linear&symbol=%s&interval=%s&limit=%d",
		c.config.BaseURL, symbol, bybitInterval, limit)

	if endTime > 0 {
		url = fmt.Sprintf("%s&end=%d", url, endTime)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.doRequestWithRetry(req)
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

