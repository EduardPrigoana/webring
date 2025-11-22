package services

import (
	"database/sql"
	"time"
	"webring/internal/database"

	"github.com/gorilla/feeds"
)

func GenerateNewSitesFeed(db *sql.DB, webringURL string) (*feeds.Feed, error) {
	sites, err := database.GetAllSites(db, "added_at")
	if err != nil {
		return nil, err
	}

	feed := &feeds.Feed{
		Title:       "Webring - New Sites",
		Link:        &feeds.Link{Href: webringURL},
		Description: "Recently joined sites in the webring",
		Created:     time.Now(),
	}

	for i, site := range sites {
		if i >= 20 {
			break
		}

		item := &feeds.Item{
			Title:       site.Name,
			Link:        &feeds.Link{Href: site.URL},
			Description: site.Description,
			Created:     site.AddedAt,
		}
		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

func GenerateUpdatedSitesFeed(db *sql.DB, webringURL string) (*feeds.Feed, error) {
	sites, err := database.GetRecentlyUpdatedSites(db, 20)
	if err != nil {
		return nil, err
	}

	feed := &feeds.Feed{
		Title:       "Webring - Recently Updated",
		Link:        &feeds.Link{Href: webringURL},
		Description: "Recently updated sites in the webring",
		Created:     time.Now(),
	}

	for _, site := range sites {
		item := &feeds.Item{
			Title:       site.Name,
			Link:        &feeds.Link{Href: site.URL},
			Description: site.Description,
			Updated:     site.UpdatedAt,
		}
		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}

func GenerateStatusFeed(db *sql.DB, webringURL string, status string) (*feeds.Feed, error) {
	sites, err := database.GetSitesByStatus(db, status)
	if err != nil {
		return nil, err
	}

	feed := &feeds.Feed{
		Title:       "Webring - " + status + " sites",
		Link:        &feeds.Link{Href: webringURL},
		Description: "Sites with status: " + status,
		Created:     time.Now(),
	}

	for _, site := range sites {
		item := &feeds.Item{
			Title:       site.Name,
			Link:        &feeds.Link{Href: site.URL},
			Description: site.Description,
			Created:     site.AddedAt,
		}
		feed.Items = append(feed.Items, item)
	}

	return feed, nil
}
