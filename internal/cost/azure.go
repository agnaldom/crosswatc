package cost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrNoPrice means the meter was not found.
var ErrNoPrice = errors.New("cost: price not found")

// Azure reads the public Azure Retail Prices API (no credentials). Prices are cached.
type Azure struct {
	BaseURL  string // default https://prices.azure.com/api/retail/prices
	Currency string // default USD
	TTL      time.Duration
	HTTP     *http.Client

	mu    sync.Mutex
	cache map[PriceQuery]cached
}

type cached struct {
	p   Price
	err error
	at  time.Time
}

type retailItem struct {
	RetailPrice   float64 `json:"retailPrice"`
	CurrencyCode  string  `json:"currencyCode"`
	UnitOfMeasure string  `json:"unitOfMeasure"`
	MeterName     string  `json:"meterName"`
	ProductName   string  `json:"productName"`
	SkuName       string  `json:"skuName"`
	Type          string  `json:"type"`
}

// Price returns the Linux, on-demand (Consumption) price; Spot, Low Priority and Windows meters are ignored.
func (a *Azure) Price(ctx context.Context, q PriceQuery) (Price, error) {
	ttl := a.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	a.mu.Lock()
	if a.cache == nil {
		a.cache = map[PriceQuery]cached{}
	}
	if c, ok := a.cache[q]; ok && time.Since(c.at) < ttl {
		a.mu.Unlock()
		return c.p, c.err
	}
	a.mu.Unlock()

	p, err := a.fetch(ctx, q)
	if err == nil || errors.Is(err, ErrNoPrice) { // do not cache transport errors
		a.mu.Lock()
		a.cache[q] = cached{p: p, err: err, at: time.Now()}
		a.mu.Unlock()
	}
	return p, err
}

func esc(s string) string { return strings.ReplaceAll(s, "'", "''") }

func (a *Azure) fetch(ctx context.Context, q PriceQuery) (Price, error) {
	base := a.BaseURL
	if base == "" {
		base = "https://prices.azure.com/api/retail/prices"
	}
	currency := a.Currency
	if currency == "" {
		currency = "USD"
	}
	filter := fmt.Sprintf("serviceName eq '%s' and armSkuName eq '%s' and armRegionName eq '%s' and priceType eq 'Consumption'",
		esc(q.Service), esc(q.SKU), esc(q.Region))
	u := base + "?" + url.Values{"currencyCode": {"'" + currency + "'"}, "$filter": {filter}}.Encode()
	hc := a.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	best := Price{Unit: -1}
	for page := 0; u != "" && page < 5; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return Price{}, err
		}
		resp, err := hc.Do(req)
		if err != nil {
			return Price{}, fmt.Errorf("cost: azure prices: %w", err)
		}
		var body struct {
			Items        []retailItem `json:"Items"`
			NextPageLink string       `json:"NextPageLink"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return Price{}, fmt.Errorf("cost: azure prices: HTTP %d", resp.StatusCode)
		}
		if err != nil {
			return Price{}, fmt.Errorf("cost: azure prices: %w", err)
		}
		for _, it := range body.Items {
			name := strings.ToLower(it.MeterName + " " + it.SkuName + " " + it.ProductName)
			if strings.Contains(name, "spot") || strings.Contains(name, "low priority") || strings.Contains(name, "windows") {
				continue
			}
			if best.Unit < 0 || it.RetailPrice < best.Unit {
				best = Price{Unit: it.RetailPrice, Currency: it.CurrencyCode, UnitName: it.UnitOfMeasure}
			}
		}
		u = body.NextPageLink
	}
	if best.Unit < 0 {
		return Price{}, fmt.Errorf("%w: %s %s %s", ErrNoPrice, q.Service, q.SKU, q.Region)
	}
	return best, nil
}
