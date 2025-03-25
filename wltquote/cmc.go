package wltquote

type CMCQuoteData struct {
	Id                int                       `json:"id"`
	Name              string                    `json:"name"`
	Symbol            string                    `json:"symbol"`
	Slug              string                    `json:"slug"`
	Added             string                    `json:"date_added"` // 2020-08-01T00:00:00.000Z
	Tags              []string                  `json:"tags"`
	CirculatingSupply float64                   `json:"circulating_supply"` // big.Float
	TotalSupply       float64                   `json:"total_supply"`       // big.Float?
	LastUpdated       string                    `json:"last_updated"`       // 2024-07-30T05:43:00.000Z
	Quote             map[string]*CMCQuoteEntry `json:"quote"`
}

type CMCQuoteEntry struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	VolumeChange24h  float64 `json:"volume_change_24h"`
	PercentChange1h  float64 `json:"percent_change_1h"`
	PercentChange24h float64 `json:"percent_change_24h"`
	MarketCap        float64 `json:"market_cap"`
	LastUpdated      string  `json:"last_updated"` // 2024-07-30T05:43:00.000Z
}
