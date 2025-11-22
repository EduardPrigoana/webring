package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"webring/internal/api"
	"webring/internal/config"
	"webring/internal/database"
	"webring/internal/models"
	"webring/internal/services"
)

var cfg *config.Config

func SetupRoutes(db *sql.DB, config *config.Config) {
	cfg = config

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	if _, err := os.Stat("./frontend/index.html"); err == nil {
		fmt.Println("Using custom frontend from ./frontend")
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, "./frontend/index.html")
			} else if !strings.HasPrefix(r.URL.Path, "/api/") &&
				!strings.HasPrefix(r.URL.Path, "/next/") &&
				!strings.HasPrefix(r.URL.Path, "/prev/") &&
				!strings.HasPrefix(r.URL.Path, "/rand/") &&
				!strings.HasPrefix(r.URL.Path, "/feed/") &&
				!strings.HasPrefix(r.URL.Path, "/signup") &&
				!strings.HasPrefix(r.URL.Path, "/data") {
				http.ServeFile(w, r, "./frontend"+r.URL.Path)
			}
		})
	} else {
		fmt.Println("Using default built-in frontend")
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				handleHome(w, r, db)
			} else {
				http.NotFound(w, r)
			}
		})
	}

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		handleData(w, r, db)
	})

	http.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		handleSignup(w, r, db)
	})
	http.HandleFunc("/next/", func(w http.ResponseWriter, r *http.Request) {
		handleNext(w, r, db)
	})
	http.HandleFunc("/prev/", func(w http.ResponseWriter, r *http.Request) {
		handlePrev(w, r, db)
	})
	http.HandleFunc("/rand/", func(w http.ResponseWriter, r *http.Request) {
		handleRand(w, r, db)
	})

	http.HandleFunc("/feed/new", func(w http.ResponseWriter, r *http.Request) {
		handleNewFeed(w, r, db)
	})
	http.HandleFunc("/feed/updated", func(w http.ResponseWriter, r *http.Request) {
		handleUpdatedFeed(w, r, db)
	})
	http.HandleFunc("/feed/status/", func(w http.ResponseWriter, r *http.Request) {
		handleStatusFeed(w, r, db)
	})

	api.SetupAPIRoutes(db, config)
}

func handleData(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "added_at"
	}

	sites, err := database.GetAllSites(db, sortBy)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	stats, err := database.GetStats(db)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	traffic, _ := database.GetTrafficSources(db, 5)

	type SiteDisplay struct {
		models.Site
		LastCheckedStr string
		AddedAtStr     string
		DisplayStatus  string
	}

	siteDisplays := make([]SiteDisplay, len(sites))
	for i, site := range sites {
		lastChecked := "never"
		if !site.LastChecked.IsZero() && site.LastChecked.Year() > 1970 {
			lastChecked = formatTime(site.LastChecked)
		}

		effectiveStatus := database.GetEffectiveStatus(&site)

		siteDisplays[i] = SiteDisplay{
			Site:           site,
			LastCheckedStr: lastChecked,
			AddedAtStr:     formatTime(site.AddedAt),
			DisplayStatus:  effectiveStatus,
		}
	}

	data := struct {
		Sites       []SiteDisplay          `json:"sites"`
		Stats       *models.Stats          `json:"stats"`
		Traffic     []models.TrafficSource `json:"traffic"`
		WebringURL  string                 `json:"webring_url"`
	}{
		Sites:      siteDisplays,
		Stats:      stats,
		Traffic:    traffic,
		WebringURL: cfg.WebringURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func handleHome(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	data := struct {
		WebringURL  string
		SiteTitle   string
		SiteFavicon string
	}{
		WebringURL:  cfg.WebringURL,
		SiteTitle:   cfg.SiteTitle,
		SiteFavicon: cfg.SiteFavicon,
	}

	tmpl.Execute(w, data)
}

func handleSignup(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	siteURL := strings.TrimSpace(r.FormValue("url"))
	description := strings.TrimSpace(r.FormValue("description"))
	contact := strings.TrimSpace(r.FormValue("contact"))
	logoURL := strings.TrimSpace(r.FormValue("logo_url"))

	if name == "" || siteURL == "" {
		http.Redirect(w, r, "/?msg=name+and+url+required&type=error", http.StatusSeeOther)
		return
	}

	if strings.Contains(name, " ") {
		http.Redirect(w, r, "/?msg=no+spaces+in+name&type=error", http.StatusSeeOther)
		return
	}

	parsedURL, err := url.Parse(siteURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Redirect(w, r, "/?msg=invalid+url&type=error", http.StatusSeeOther)
		return
	}

	existing, err := database.GetSiteByName(db, name)
	if err != nil {
		http.Redirect(w, r, "/?msg=database+error&type=error", http.StatusSeeOther)
		return
	}
	if existing != nil {
		http.Redirect(w, r, "/?msg=name+taken&type=error", http.StatusSeeOther)
		return
	}

	newSite := models.Site{
		Name:        name,
		URL:         siteURL,
		Description: description,
		Contact:     contact,
		LogoURL:     logoURL,
		Status:      "dead",
		LastChecked: time.Now(),
		AddedAt:     time.Now(),
		UpdatedAt:   time.Now(),
	}

	fmt.Printf("Checking %s...\n", name)
	status := services.CheckSiteBacklinks(newSite, cfg.WebringURL)
	newSite.Status = status

	_, err = database.CreateSite(db, newSite)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Redirect(w, r, "/?msg=already+registered&type=error", http.StatusSeeOther)
			return
		}
		fmt.Printf("Error creating site: %v\n", err)
		http.Redirect(w, r, "/?msg=database+error&type=error", http.StatusSeeOther)
		return
	}

	if status == "up" {
		http.Redirect(w, r, "/?msg=site+added+successfully&type=success", http.StatusSeeOther)
	} else if status == "broken" {
		http.Redirect(w, r, "/?msg=site+added+but+backlinks+missing&type=error", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/?msg=site+added+but+unreachable&type=error", http.StatusSeeOther)
	}
}

func handleNext(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	siteName := strings.TrimPrefix(r.URL.Path, "/next/")
	recordNavigation(db, siteName, "next", r)

	services.UpdateSiteIfNeeded(db, siteName, cfg.CheckCooldown)

	currentSite, err := database.GetSiteByName(db, siteName)
	if err != nil || currentSite == nil {
		http.Redirect(w, r, "/?msg=site+not+found&type=error", http.StatusSeeOther)
		return
	}

	activeSites, err := database.GetActiveSites(db)
	if err != nil || len(activeSites) == 0 {
		http.Redirect(w, r, "/?msg=no+active+sites&type=error", http.StatusSeeOther)
		return
	}

	currentIndex := -1
	for i, site := range activeSites {
		if site.ID == currentSite.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		database.IncrementClickCount(db, activeSites[0].ID)
		http.Redirect(w, r, activeSites[0].URL, http.StatusFound)
		return
	}

	nextIndex := (currentIndex + 1) % len(activeSites)
	database.IncrementClickCount(db, activeSites[nextIndex].ID)
	http.Redirect(w, r, activeSites[nextIndex].URL, http.StatusFound)
}

func handlePrev(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	siteName := strings.TrimPrefix(r.URL.Path, "/prev/")
	recordNavigation(db, siteName, "prev", r)

	services.UpdateSiteIfNeeded(db, siteName, cfg.CheckCooldown)

	currentSite, err := database.GetSiteByName(db, siteName)
	if err != nil || currentSite == nil {
		http.Redirect(w, r, "/?msg=site+not+found&type=error", http.StatusSeeOther)
		return
	}

	activeSites, err := database.GetActiveSites(db)
	if err != nil || len(activeSites) == 0 {
		http.Redirect(w, r, "/?msg=no+active+sites&type=error", http.StatusSeeOther)
		return
	}

	currentIndex := -1
	for i, site := range activeSites {
		if site.ID == currentSite.ID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		database.IncrementClickCount(db, activeSites[len(activeSites)-1].ID)
		http.Redirect(w, r, activeSites[len(activeSites)-1].URL, http.StatusFound)
		return
	}

	prevIndex := (currentIndex - 1 + len(activeSites)) % len(activeSites)
	database.IncrementClickCount(db, activeSites[prevIndex].ID)
	http.Redirect(w, r, activeSites[prevIndex].URL, http.StatusFound)
}

func handleRand(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	siteName := strings.TrimPrefix(r.URL.Path, "/rand/")
	recordNavigation(db, siteName, "random", r)

	services.UpdateSiteIfNeeded(db, siteName, cfg.CheckCooldown)

	currentSite, err := database.GetSiteByName(db, siteName)
	if err != nil || currentSite == nil {
		http.Redirect(w, r, "/?msg=site+not+found&type=error", http.StatusSeeOther)
		return
	}

	activeSites, err := database.GetActiveSites(db)
	if err != nil || len(activeSites) == 0 {
		http.Redirect(w, r, "/?msg=no+active+sites&type=error", http.StatusSeeOther)
		return
	}

	var otherSites []models.Site
	for _, site := range activeSites {
		if site.ID != currentSite.ID {
			otherSites = append(otherSites, site)
		}
	}

	if len(otherSites) == 0 {
		http.Redirect(w, r, "/?msg=no+other+sites&type=error", http.StatusSeeOther)
		return
	}

	randomSite := otherSites[rand.Intn(len(otherSites))]
	database.IncrementClickCount(db, randomSite.ID)
	http.Redirect(w, r, randomSite.URL, http.StatusFound)
}

func recordNavigation(db *sql.DB, siteName, action string, r *http.Request) {
	site, err := database.GetSiteByName(db, siteName)
	if err != nil || site == nil {
		return
	}

	click := models.Click{
		SiteID:    site.ID,
		Action:    action,
		Referrer:  r.Referer(),
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
	}

	database.RecordClick(db, click)
}

func handleNewFeed(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	feed, err := services.GenerateNewSitesFeed(db, cfg.WebringURL)
	if err != nil {
		http.Error(w, "Error generating feed", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	w.Header().Set("Content-Type", "application/xml")

	if format == "atom" {
		atom, _ := feed.ToAtom()
		w.Write([]byte(atom))
	} else {
		rss, _ := feed.ToRss()
		w.Write([]byte(rss))
	}
}

func handleUpdatedFeed(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	feed, err := services.GenerateUpdatedSitesFeed(db, cfg.WebringURL)
	if err != nil {
		http.Error(w, "Error generating feed", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	w.Header().Set("Content-Type", "application/xml")

	if format == "atom" {
		atom, _ := feed.ToAtom()
		w.Write([]byte(atom))
	} else {
		rss, _ := feed.ToRss()
		w.Write([]byte(rss))
	}
}

func handleStatusFeed(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	status := strings.TrimPrefix(r.URL.Path, "/feed/status/")
	if status != "up" && status != "broken" && status != "dead" {
		http.NotFound(w, r)
		return
	}

	feed, err := services.GenerateStatusFeed(db, cfg.WebringURL, status)
	if err != nil {
		http.Error(w, "Error generating feed", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	w.Header().Set("Content-Type", "application/xml")

	if format == "atom" {
		atom, _ := feed.ToAtom()
		w.Write([]byte(atom))
	} else {
		rss, _ := feed.ToRss()
		w.Write([]byte(rss))
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() || t.Year() <= 1970 {
		return "never"
	}

	diff := time.Since(t)

	if diff < 0 {
		diff = -diff
	}

	if diff < time.Minute {
		return "< 1m ago"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else if diff < 30*24*time.Hour {
		weeks := int(diff.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	} else if diff < 365*24*time.Hour {
		months := int(diff.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo ago", months)
	}

	years := int(diff.Hours() / 24 / 365)
	return fmt.Sprintf("%dy ago", years)
}
