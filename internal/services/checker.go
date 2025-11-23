package services

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"webring/internal/config"
	"webring/internal/database"
	"webring/internal/models"
)

func CheckSiteBacklinks(site models.Site, webringURL string) string {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(site.URL)
	if err != nil {
		fmt.Printf("[DEAD] %s: %v\n", site.URL, err)
		return "dead"
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[DEAD] %s (status %d)\n", site.URL, resp.StatusCode)
		return "dead"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[DEAD] %s error reading body: %v\n", site.URL, err)
		return "dead"
	}

	content := strings.ToLower(string(body))

	nextLink := strings.TrimRight(webringURL, "/") + "/next/" + site.Name
	prevLink := strings.TrimRight(webringURL, "/") + "/prev/" + site.Name

	hasNext := strings.Contains(content, nextLink)
	hasPrev := strings.Contains(content, prevLink)

	if hasNext && hasPrev {
		fmt.Printf("[UP] %s (valid backlinks)\n", site.Name)
		return "up"
	}

	fmt.Printf("[BROKEN] %s (missing backlinks: next=%v, prev=%v)\n", site.Name, hasNext, hasPrev)
	return "broken"
}

func RecheckAllSites(db *sql.DB) {
	cfg := config.Load()
	sites, err := database.GetAllSites(db, "added_at")
	if err != nil {
		fmt.Printf("Error getting sites: %v\n", err)
		return
	}

	if len(sites) == 0 {
		fmt.Println("No sites to check")
		return
	}

	fmt.Printf("Checking %d sites...\n", len(sites))

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, site := range sites {
		wg.Add(1)
		go func(s models.Site) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			status := CheckSiteBacklinks(s, cfg.WebringURL)
			if err := database.UpdateSiteStatus(db, s.Name, status); err != nil {
				fmt.Printf("Error updating %s: %v\n", s.Name, err)
			}
		}(site)
	}

	wg.Wait()
	fmt.Println("All sites checked")
}

func StartPeriodicChecker(db *sql.DB, interval time.Duration) {
	cfg := config.Load()
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("Running periodic status check...")

		sites, err := database.GetAllSites(db, "added_at")
		if err != nil {
			fmt.Printf("Error getting sites: %v\n", err)
			continue
		}

		for _, site := range sites {
			if time.Since(site.LastChecked) > interval {
				status := CheckSiteBacklinks(site, cfg.WebringURL)
				database.UpdateSiteStatus(db, site.Name, status)
				time.Sleep(2 * time.Second)
			}
		}

		fmt.Println("Periodic check completed")
	}
}

func UpdateSiteIfNeeded(db *sql.DB, siteName string, cooldown time.Duration) {
	cfg := config.Load()
	site, err := database.GetSiteByName(db, siteName)
	if err != nil || site == nil {
		return
	}

	if site.LastChecked.IsZero() || site.LastChecked.Year() <= 1970 || time.Since(site.LastChecked) > cooldown {
		fmt.Printf("Rechecking %s...\n", siteName)
		status := CheckSiteBacklinks(*site, cfg.WebringURL)
		database.UpdateSiteStatus(db, siteName, status)
	}
}
