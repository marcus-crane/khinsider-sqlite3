package main

import (
	"fmt"
	"log"
	"strconv"
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
	FilesizeMP3Bytes  int        `json:"filesize_mp3_bytes"`
	FilesizeFlacBytes int        `json:"filesize_flac_bytes"`
	Tracks            []Track    `json:"tracks"`
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
	AlbumID           string `json:"album_id"`
	URL               string `json:"url"`
	TrackNumber       int    `json:"track_number"`
	DiscNumber        int    `json:"disc_number"`
	Title             string `json:"title"`
	Runtime           string `json:"runtime"`
	MP3Available      bool   `json:"mp3_available"`
	FlacAvailable     bool   `json:"flac_available"`
	FilesizeMP3Bytes  string `json:"filesize_mp3_bytes"`
	FilesizeFlacBytes string `json:"filesize_flac_bytes"`
}

var (
	BASE_URL   = "https://downloads.khinsider.com"
	FIRST_PAGE = "https://downloads.khinsider.com/game-soundtracks?page=1"
)

func main() {
	db, err := gorm.Open(sqlite.Open("khinsider.db"))
	if err != nil {
		panic("failed to connect to database")
	}

	db.AutoMigrate(Album{})
	db.AutoMigrate(Track{})

	// updatePlatforms(db)
	updateAlbums(db)
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
	db.Select("url").Find(&albums)

	c := colly.NewCollector()

	q, _ := queue.New(
		2,
		&queue.InMemoryQueueStorage{MaxSize: 10000},
	)

	tableHeaders := map[int]string{}

	c.OnHTML("table#songlist tbody tr", func(e *colly.HTMLElement) {
		album := Album{}
		fmt.Println(e.Request.URL.Path)
		albumID := strings.ReplaceAll(e.Request.URL.Path, "/game-soundtracks/album/", "")
		fmt.Println(albumID)
		db.First(&album, "id = ?", albumID)
		if e.Index == 0 {
			e.ForEach("th", func(index int, e *colly.HTMLElement) {
				tableHeaders[index] = strings.TrimSpace(e.Text)
			})
		}
		track := Track{}
		e.ForEach("td", func(index int, e *colly.HTMLElement) {
			if e.Index == 0 {
				return
			}
			if tableHeaders[e.Index] == "CD" {
				discNum, err := strconv.Atoi(strings.TrimSpace(e.Text))
				if err != nil {
					fmt.Println("failed to convert disc num")
				} else {
					track.DiscNumber = discNum
				}
			}
			if e.Attr("class") == "clickable-row" && e.Attr("align") == "right" {
				track.Runtime = e.Text
			}
			if tableHeaders[e.Index] == "#" {
				trackNum, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(e.Text), ".", ""))
				if err != nil {
					fmt.Println("failed to convert track num")
				} else {
					track.TrackNumber = trackNum
				}
			}
			if tableHeaders[e.Index] == "Song Name" {
				track.Title = strings.TrimSpace(e.Text)
			}
			if tableHeaders[e.Index] == "MP3" {
				link := e.ChildAttr("a", "href")
				track.URL = fmt.Sprintf("%s%s", BASE_URL, link)
				track.MP3Available = true
				track.FilesizeMP3Bytes = e.Text
			}
			if tableHeaders[e.Index] == "FLAC" {
				link := e.ChildAttr("a", "href")
				track.URL = fmt.Sprintf("%s%s", BASE_URL, link)
				track.FlacAvailable = true
				track.FilesizeFlacBytes = e.Text
			}
		})
		album.Tracks = append(album.Tracks, track)
		fmt.Println(album)
		db.Save(&album)
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
