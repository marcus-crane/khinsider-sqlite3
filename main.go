package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/queue"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Album struct {
	ID                string     `gorm:"primaryKey" json:"id"`
	URL               string     `json:"url"`
	Title             string     `json:"title"`
	Year              string     `json:"year"`
	DateAdded         string     `json:"date_added"`
	Runtime           string     `json:"runtime"`
	Type              string     `json:"type"`
	FileCount         int        `json:"file_count"`
	FilesizeMP3Bytes  int        `json:filesize_mp3_bytes`
	FilesizeFlacBytes int        `json:"filesize_flac_bytes`
	Platforms         []Platform `gorm:"many2many:album_platforms;" json:"platforms"`
	Images            []Image    `gorm:"many2many:album_images;" json:"images"`
}

type Platform struct {
	Name string `gorm:"primaryKey"`
}

type Image struct {
	URL string `gorm:"primaryKey"`
}

type Track struct {
	gorm.Model
	URL               string `json:"url"`
	TrackNumber       int    `json:"track_number"`
	DiscNumber        int    `json:"disc_number"`
	Title             string `json:"title"`
	Runtime           string `json:"runtime"`
	MP3Available      bool   `json:"mp3_available"`
	FlacAvailable     bool   `json:"flac_available"`
	FilesizeMP3Bytes  int    `json:"filesize_mp3_bytes"`
	FilesizeFlacBytes int    `json:"filesize_flac_bytes`
}

var (
	BASE_URL   = "https://downloads.khinsider.com"
	FIRST_PAGE = "https://downloads.khinsider.com/game-soundtracks?page=1"
	db         *gorm.DB
)

func main() {
	db, err := gorm.Open(sqlite.Open("khinsider.db"))
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(Album{})
	db.AutoMigrate(Track{})

	// updatePlatforms(db)
	// updateAlbums(db)
	updateAlbumMetadata(db)
}

func contains(s []string, sub string) bool {
	for _, e := range s {
		if e == sub {
			return true
		}
	}
	return false
}

func updateAlbumMetadata(db *gorm.DB) {
	albums := []Album{}
	db.Find(&albums)

	c := colly.NewCollector()

	q, _ := queue.New(
		2,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		fmt.Print(e.Text)
	})

	for _, album := range albums {
		q.AddURL(album.URL)
	}

	q.Run(c)
}

func updateAlbums(db *gorm.DB) {
	c := colly.NewCollector()

	c.OnHTML("table.albumList tr", func(e *colly.HTMLElement) {
		if e.Index == 0 {
			return
		}
		album := Album{}
		e.ForEach("td", func(index int, e *colly.HTMLElement) {
			if index == 1 {
				link := e.ChildAttr("a", "href")
				album.ID = strings.TrimSpace(strings.ReplaceAll(link, "/game-soundtracks/album/", ""))
				album.URL = fmt.Sprintf("%s%s", BASE_URL, link)
				album.Title = strings.TrimSpace(e.Text)
			}
			if index == 2 {
				e.ForEach("a", func(_ int, e *colly.HTMLElement) {
					platform := strings.TrimSpace(e.Text)
					album.Platforms = append(album.Platforms, Platform{Name: platform})
				})
			}
			if index == 3 {
				album.Type = strings.TrimSpace(e.Text)
			}
			if index == 4 {
				album.Year = strings.TrimSpace(e.Text)
			}
		})
		db.Save(&album)
	})

	c.OnHTML("div.pagination > ul > li.pagination-next > a[href]", func(e *colly.HTMLElement) {
		log.Println("Next page link found:", e.Attr("href"))
		e.Request.Visit(e.Attr("href"))
	})

	c.Visit(FIRST_PAGE)
}
