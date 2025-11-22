package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
	"webring/internal/models"

	_ "github.com/lib/pq"
)

func Connect(url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func InitSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sites (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		url TEXT NOT NULL UNIQUE,
		description TEXT,
		contact VARCHAR(255),
		logo_url TEXT,
		status VARCHAR(50) NOT NULL DEFAULT 'dead',
		manual_status_override VARCHAR(50),
		click_count INTEGER NOT NULL DEFAULT 0,
		last_checked TIMESTAMP,
		added_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS clicks (
		id SERIAL PRIMARY KEY,
		site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
		action VARCHAR(50) NOT NULL,
		referrer TEXT,
		user_agent TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	indexes := `
	CREATE INDEX IF NOT EXISTS idx_sites_status ON sites(status);
	CREATE INDEX IF NOT EXISTS idx_sites_name ON sites(name);
	CREATE INDEX IF NOT EXISTS idx_sites_click_count ON sites(click_count DESC);
	CREATE INDEX IF NOT EXISTS idx_sites_updated_at ON sites(updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_clicks_site_id ON clicks(site_id);
	CREATE INDEX IF NOT EXISTS idx_clicks_created_at ON clicks(created_at);
	`

	_, err = db.Exec(indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS description TEXT`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS contact VARCHAR(255)`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS logo_url TEXT`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS manual_status_override VARCHAR(50)`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS click_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE sites ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW()`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate column") {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	return nil
}

func CreateSite(db *sql.DB, site models.Site) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO sites (name, url, description, contact, logo_url, status, last_checked, added_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, site.Name, site.URL, site.Description, site.Contact, site.LogoURL,
		site.Status, site.LastChecked, site.AddedAt, site.UpdatedAt).Scan(&id)
	return id, err
}

func GetSiteByName(db *sql.DB, name string) (*models.Site, error) {
	site := &models.Site{}
	var manualOverride sql.NullString

	err := db.QueryRow(`
		SELECT id, name, url, COALESCE(description, ''), COALESCE(contact, ''),
		       COALESCE(logo_url, ''), status, manual_status_override, click_count,
		       COALESCE(last_checked, '1970-01-01'), added_at, updated_at
		FROM sites WHERE LOWER(name) = LOWER($1)
	`, name).Scan(&site.ID, &site.Name, &site.URL, &site.Description, &site.Contact,
		&site.LogoURL, &site.Status, &manualOverride, &site.ClickCount,
		&site.LastChecked, &site.AddedAt, &site.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if manualOverride.Valid {
		site.ManualStatusOverride = &manualOverride.String
	}

	return site, nil
}

func GetAllSites(db *sql.DB, orderBy string) ([]models.Site, error) {
	validOrders := map[string]bool{
		"added_at":    true,
		"updated_at":  true,
		"click_count": true,
		"name":        true,
	}

	if !validOrders[orderBy] {
		orderBy = "added_at"
	}

	query := fmt.Sprintf(`
		SELECT id, name, url, COALESCE(description, ''), COALESCE(contact, ''),
		       COALESCE(logo_url, ''), status, manual_status_override, click_count,
		       COALESCE(last_checked, '1970-01-01'), added_at, updated_at
		FROM sites ORDER BY %s DESC
	`, orderBy)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		var manualOverride sql.NullString

		err := rows.Scan(&site.ID, &site.Name, &site.URL, &site.Description, &site.Contact,
			&site.LogoURL, &site.Status, &manualOverride, &site.ClickCount,
			&site.LastChecked, &site.AddedAt, &site.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if manualOverride.Valid {
			site.ManualStatusOverride = &manualOverride.String
		}

		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func GetActiveSites(db *sql.DB) ([]models.Site, error) {
	rows, err := db.Query(`
		SELECT id, name, url, COALESCE(description, ''), COALESCE(contact, ''),
		       COALESCE(logo_url, ''), status, manual_status_override, click_count,
		       COALESCE(last_checked, '1970-01-01'), added_at, updated_at
		FROM sites WHERE status = 'up' ORDER BY added_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		var manualOverride sql.NullString

		err := rows.Scan(&site.ID, &site.Name, &site.URL, &site.Description, &site.Contact,
			&site.LogoURL, &site.Status, &manualOverride, &site.ClickCount,
			&site.LastChecked, &site.AddedAt, &site.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if manualOverride.Valid {
			site.ManualStatusOverride = &manualOverride.String
		}

		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func GetSitesByStatus(db *sql.DB, status string) ([]models.Site, error) {
	rows, err := db.Query(`
		SELECT id, name, url, COALESCE(description, ''), COALESCE(contact, ''),
		       COALESCE(logo_url, ''), status, manual_status_override, click_count,
		       COALESCE(last_checked, '1970-01-01'), added_at, updated_at
		FROM sites WHERE status = $1 ORDER BY added_at
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		var manualOverride sql.NullString

		err := rows.Scan(&site.ID, &site.Name, &site.URL, &site.Description, &site.Contact,
			&site.LogoURL, &site.Status, &manualOverride, &site.ClickCount,
			&site.LastChecked, &site.AddedAt, &site.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if manualOverride.Valid {
			site.ManualStatusOverride = &manualOverride.String
		}

		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func GetRecentlyUpdatedSites(db *sql.DB, limit int) ([]models.Site, error) {
	rows, err := db.Query(`
		SELECT id, name, url, COALESCE(description, ''), COALESCE(contact, ''),
		       COALESCE(logo_url, ''), status, manual_status_override, click_count,
		       COALESCE(last_checked, '1970-01-01'), added_at, updated_at
		FROM sites ORDER BY updated_at DESC LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		var manualOverride sql.NullString

		err := rows.Scan(&site.ID, &site.Name, &site.URL, &site.Description, &site.Contact,
			&site.LogoURL, &site.Status, &manualOverride, &site.ClickCount,
			&site.LastChecked, &site.AddedAt, &site.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if manualOverride.Valid {
			site.ManualStatusOverride = &manualOverride.String
		}

		sites = append(sites, site)
	}
	return sites, rows.Err()
}

func UpdateSiteStatus(db *sql.DB, name string, status string) error {
	_, err := db.Exec(`
		UPDATE sites
		SET status = $1, last_checked = $2, updated_at = $3
		WHERE LOWER(name) = LOWER($4) AND manual_status_override IS NULL
	`, status, time.Now(), time.Now(), name)
	return err
}

func UpdateSite(db *sql.DB, site models.Site) error {
	_, err := db.Exec(`
		UPDATE sites
		SET url = $1, description = $2, contact = $3, logo_url = $4, updated_at = $5
		WHERE id = $6
	`, site.URL, site.Description, site.Contact, site.LogoURL, time.Now(), site.ID)
	return err
}

func DeleteSite(db *sql.DB, name string) error {
	result, err := db.Exec("DELETE FROM sites WHERE LOWER(name) = LOWER($1)", name)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func IncrementClickCount(db *sql.DB, siteID int) error {
	_, err := db.Exec("UPDATE sites SET click_count = click_count + 1 WHERE id = $1", siteID)
	return err
}

func RecordClick(db *sql.DB, click models.Click) error {
	_, err := db.Exec(`
		INSERT INTO clicks (site_id, action, referrer, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, click.SiteID, click.Action, click.Referrer, click.UserAgent, click.CreatedAt)
	return err
}

func GetStats(db *sql.DB) (*models.Stats, error) {
	stats := &models.Stats{LastUpdated: time.Now()}

	err := db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'up') as active,
			COUNT(*) FILTER (WHERE status = 'broken') as broken,
			COUNT(*) FILTER (WHERE status = 'dead') as dead,
			COALESCE(SUM(click_count), 0) as total_clicks
		FROM sites
	`).Scan(&stats.TotalSites, &stats.ActiveSites, &stats.BrokenSites, &stats.DeadSites, &stats.TotalClicks)

	return stats, err
}

func GetTrafficSources(db *sql.DB, limit int) ([]models.TrafficSource, error) {
	rows, err := db.Query(`
		SELECT COALESCE(referrer, 'direct') as referrer, COUNT(*) as count
		FROM clicks
		WHERE referrer IS NOT NULL AND referrer != ''
		GROUP BY referrer
		ORDER BY count DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.TrafficSource
	for rows.Next() {
		var source models.TrafficSource
		if err := rows.Scan(&source.Referrer, &source.Count); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

func GetEffectiveStatus(site *models.Site) string {
	if site.ManualStatusOverride != nil && *site.ManualStatusOverride != "" {
		return *site.ManualStatusOverride
	}
	return site.Status
}
