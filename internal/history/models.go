package history

// MonthlyAverage represents the average flight price for a single calendar month.
type MonthlyAverage struct {
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	AvgPrice float64 `json:"avg_price"`
	Currency string  `json:"currency"`
}

// Response is the payload returned by GET /flights/history.
type Response struct {
	Origin      string           `json:"origin"`
	Destination string           `json:"destination"`
	History     []MonthlyAverage `json:"history"`
}
