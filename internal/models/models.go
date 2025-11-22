package models

import "time"

type Site struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	URL                  string    `json:"url"`
	Description          string    `json:"description,omitempty"`
	Contact              string    `json:"contact,omitempty"`
	LogoURL              string    `json:"logo_url,omitempty"`
	Status               string    `json:"status"`
	ManualStatusOverride *string   `json:"manual_status_override,omitempty"`
	ClickCount           int       `json:"click_count"`
	LastChecked          time.Time `json:"last_checked"`
	AddedAt              time.Time `json:"added_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Click struct {
	ID        int       `json:"id"`
	SiteID    int       `json:"site_id"`
	SiteName  string    `json:"site_name"`
	Action    string    `json:"action"`
	Referrer  string    `json:"referrer,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	TotalSites  int       `json:"total_sites"`
	ActiveSites int       `json:"active_sites"`
	BrokenSites int       `json:"broken_sites"`
	DeadSites   int       `json:"dead_sites"`
	TotalClicks int       `json:"total_clicks"`
	LastUpdated time.Time `json:"last_updated"`
}

type TrafficSource struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
