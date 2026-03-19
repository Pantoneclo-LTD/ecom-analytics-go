package models

type ClickRateRecord struct {
	Date        string  `json:"date" clickhouse:"date"`
	UTMSource   string  `json:"utm_source" clickhouse:"utm_source"`
	Clicks      uint64  `json:"clicks" clickhouse:"clicks"`
	Conversions uint64  `json:"conversions" clickhouse:"conversions"`
	ClickRate   float64 `json:"click_rate" clickhouse:"click_rate"`
}

type ClickRateResponse struct {
	Data []ClickRateRecord `json:"data"`
}
