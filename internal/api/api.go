package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"webring/internal/config"
	"webring/internal/database"
	"webring/internal/models"
	"webring/internal/services"
)

var cfg *config.Config

func SetupAPIRoutes(db *sql.DB, config *config.Config) {
	cfg = config

	http.HandleFunc("/api/sites", func(w http.ResponseWriter, r *http.Request) {
		handleAPISites(w, r, db)
	})
	http.HandleFunc("/api/sites/", func(w http.ResponseWriter, r *http.Request) {
		handleAPISiteDetail(w, r, db)
	})
	http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		handleAPIStats(w, r, db)
	})
	http.HandleFunc("/api/traffic", func(w http.ResponseWriter, r *http.Request) {
		handleAPITraffic(w, r, db)
	})
	http.HandleFunc("/api/check/", func(w http.ResponseWriter, r *http.Request) {
		handleAPICheck(w, r, db)
	})
	http.HandleFunc("/api/docs", handleAPIDocs)
}

func handleAPISites(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case http.MethodGet:
		status := r.URL.Query().Get("status")
		sort := r.URL.Query().Get("sort")
		if sort == "" {
			sort = "added_at"
		}

		var sites []models.Site
		var err error

		if status == "up" || status == "broken" || status == "dead" {
			sites, err = database.GetSitesByStatus(db, status)
		} else {
			sites, err = database.GetAllSites(db, sort)
		}

		if err != nil {
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Database error",
			})
			return
		}

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data:    sites,
		})

	case http.MethodPost:
		var site models.Site
		if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Invalid JSON",
			})
			return
		}

		if site.Name == "" || site.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Name and URL required",
			})
			return
		}

		site.AddedAt = time.Now()
		site.UpdatedAt = time.Now()
		site.LastChecked = time.Now()
		site.Status = services.CheckSiteBacklinks(site, cfg.WebringURL)

		id, err := database.CreateSite(db, site)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Failed to create site",
			})
			return
		}

		site.ID = id
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Message: "Site created",
			Data:    site,
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func handleAPISiteDetail(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	path := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	siteName := strings.Split(path, "/")[0]

	if siteName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Site name required",
		})
		return
	}

	switch r.Method {
	case http.MethodGet:
		site, err := database.GetSiteByName(db, siteName)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Database error",
			})
			return
		}
		if site == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Site not found",
			})
			return
		}

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Data:    site,
		})

	case "PUT":
		var updates models.Site
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Invalid JSON",
			})
			return
		}

		site, err := database.GetSiteByName(db, siteName)
		if err != nil || site == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Site not found",
			})
			return
		}

		if updates.URL != "" {
			site.URL = updates.URL
		}
		if updates.Description != "" {
			site.Description = updates.Description
		}
		if updates.Contact != "" {
			site.Contact = updates.Contact
		}
		if updates.LogoURL != "" {
			site.LogoURL = updates.LogoURL
		}

		if err := database.UpdateSite(db, *site); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Failed to update site",
			})
			return
		}

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Message: "Site updated",
			Data:    site,
		})

	case http.MethodDelete:
		err := database.DeleteSite(db, siteName)
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Site not found",
			})
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.APIResponse{
				Success: false,
				Error:   "Database error",
			})
			return
		}

		json.NewEncoder(w).Encode(models.APIResponse{
			Success: true,
			Message: "Site deleted",
		})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
	}
}

func handleAPIStats(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats, err := database.GetStats(db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}

	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

func handleAPITraffic(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	traffic, err := database.GetTrafficSources(db, 20)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Database error",
		})
		return
	}

	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Data:    traffic,
	})
}

func handleAPICheck(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	siteName := strings.TrimPrefix(r.URL.Path, "/api/check/")
	if siteName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Site name required",
		})
		return
	}

	site, err := database.GetSiteByName(db, siteName)
	if err != nil || site == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Site not found",
		})
		return
	}

	if !site.LastChecked.IsZero() && time.Since(site.LastChecked) < cfg.CheckCooldown {
		remainingTime := cfg.CheckCooldown - time.Since(site.LastChecked)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Cooldown active. Try again in %v", remainingTime.Round(time.Minute)),
		})
		return
	}

	status := services.CheckSiteBacklinks(*site, cfg.WebringURL)
	err = database.UpdateSiteStatus(db, siteName, status)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Failed to update",
		})
		return
	}

	updatedSite, _ := database.GetSiteByName(db, siteName)
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Site checked",
		Data:    updatedSite,
	})
}

func handleAPIDocs(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/docs.html")
}
