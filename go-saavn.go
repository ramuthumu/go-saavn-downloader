package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Response struct {
	Id string `json:"id"`
}

type AlbumResponse struct {
	Title            string `json:"title"`
	Name             string `json:"name"`
	Year             string `json:"year"`
	ReleaseDate      string `json:"release_date"`
	PrimaryArtists   string `json:"primary_artists"`
	PrimaryArtistsID string `json:"primary_artists_id"`
	AlbumID          string `json:"albumid"`
	PermaURL         string `json:"perma_url"`
	Image            string `json:"image"`
	Songs            []struct {
		ID                 string `json:"id"`
		Type               string `json:"type"`
		Song               string `json:"song"`
		Album              string `json:"album"`
		Year               string `json:"year"`
		Music              string `json:"music"`
		MusicID            string `json:"music_id"`
		PrimaryArtists     string `json:"primary_artists"`
		PrimaryArtistsID   string `json:"primary_artists_id"`
		FeaturedArtists    string `json:"featured_artists"`
		FeaturedArtistsID  string `json:"featured_artists_id"`
		Singers            string `json:"singers"`
		Starring           string `json:"starring"`
		Image              string `json:"image"`
		Label              string `json:"label"`
		AlbumID            string `json:"albumid"`
		Language           string `json:"language"`
		Origin             string `json:"origin"`
		PlayCount          string `json:"play_count"`
		CopyrightText      string `json:"copyright_text"`
		Kbps               string `json:"320kbps"`
		IsDolbyContent     bool   `json:"is_dolby_content"`
		ExplicitContent    int    `json:"explicit_content"`
		HasLyrics          string `json:"has_lyrics"`
		LyricsSnippet      string `json:"lyrics_snippet"`
		EncryptedMediaURL  string `json:"encrypted_media_url"`
		EncryptedMediaPath string `json:"encrypted_media_path"`
		MediaPreviewURL    string `json:"media_preview_url"`
		PermaURL           string `json:"perma_url"`
		AlbumURL           string `json:"album_url"`
		Duration           string `json:"duration"`
		Rights             struct {
			Code               int    `json:"code"`
			Reason             string `json:"reason"`
			Cacheable          bool   `json:"cacheable"`
			DeleteCachedObject bool   `json:"delete_cached_object"`
		} `json:"rights"`
		WebP             bool              `json:"webp"`
		Disabled         string            `json:"disabled"`
		DisabledText     string            `json:"disabled_text"`
		Starred          string            `json:"starred"`
		ArtistMap        map[string]string `json:"artistMap"`
		ReleaseDate      string            `json:"release_date"`
		VCode            string            `json:"vcode"`
		VLink            string            `json:"vlink"`
		TrillerAvailable bool              `json:"triller_available"`
		LabelURL         string            `json:"label_url"`
	} `json:"songs"`
}

var httpClient = &http.Client{} // Create a new HTTP client

func getAlbumID(inputUrl string) (string, error) {
	token := strings.Split(inputUrl, "/")[len(strings.Split(inputUrl, "/"))-1]
	inputUrl = fmt.Sprintf("https://www.jiosaavn.com/api.php?__call=webapi.get&token=%s&type=album&includeMetaTags=0&ctx=web6dot0&api_version=4&_format=json&_marker=0", token)

	req, err := http.NewRequest("GET", inputUrl, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data Response
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	return data.Id, nil
}

func getAlbum(albumID string) (AlbumResponse, error) {
	requestUrl := fmt.Sprintf("https://www.jiosaavn.com/api.php?_format=json&__call=content.getAlbumDetails&albumid=%s", albumID)

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return AlbumResponse{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return AlbumResponse{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return AlbumResponse{}, err
	}

	var data AlbumResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return AlbumResponse{}, err
	}

	return data, nil
}

func main() {
	fmt.Println("Please enter the album URL:")

	var url string
	_, err := fmt.Scanf("%s", &url)
	if err != nil {
		log.Fatalf("Error reading URL: %s\n", err)
	}

	albumID, err := getAlbumID(url)
	if err != nil {
		log.Fatalf("Error getting album ID: %s\n", err)
	}

	albumJSON, err := getAlbum(albumID)
	if err != nil {
		log.Fatalf("Error getting album details: %s\n", err)
	}

	fmt.Printf("Album details : %s\n", albumJSON)
}
