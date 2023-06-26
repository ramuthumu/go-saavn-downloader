package main

import (
	"crypto/des"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type Response struct {
	ID string `json:"id"`
}

type Song struct {
	ID                 string `json:"id"`
	Song               string `json:"song"`
	Album              string `json:"album"`
	Music              string `json:"music"`
	PrimaryArtists     string `json:"primary_artists"`
	FeaturedArtists    string `json:"featured_artists"`
	Singers            string `json:"singers"`
	Image              string `json:"image"`
	AlbumID            string `json:"albumid"`
	Language           string `json:"language"`
	Years              string `json:"year"`
	PlayCount          string `json:"play_count"`
	ExplicitContent    int    `json:"explicit_content"`
	EncryptedMediaURL  string `json:"encrypted_media_url"`
	EncryptedMediaPath string `json:"encrypted_media_path"`
	MediaPreviewURL    string `json:"media_preview_url"`
	PermaURL           string `json:"perma_url"`
	AlbumURL           string `json:"album_url"`
	Duration           string `json:"duration"`
	WebP               bool   `json:"webp"`
	ReleaseDate        string `json:"release_date"`
}

type AlbumResponse struct {
	Title          string `json:"title"`
	Name           string `json:"name"`
	Year           string `json:"year"`
	ReleaseDate    string `json:"release_date"`
	PrimaryArtists string `json:"primary_artists"`
	AlbumID        string `json:"albumid"`
	PermaURL       string `json:"perma_url"`
	Image          string `json:"image"`
	Songs          []Song `json:"songs"`
}

var httpClient = &http.Client{
	Timeout: time.Second * 10, // Timeout after 10 seconds
}

func decryptURL(encryptedMediaURL string) (string, error) {
	key := []byte("38346591")

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedMediaURL)
	if err != nil {
		return "", fmt.Errorf("base64 decoding error: %w", err)
	}

	block, err := des.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new DES cipher error: %w", err)
	}

	if len(ciphertext)%des.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of the block size")
	}

	decrypted := make([]byte, len(ciphertext))
	for bs := 0; bs < len(ciphertext); bs += des.BlockSize {
		block.Decrypt(decrypted[bs:], ciphertext[bs:])
	}

	decrypted = pkcs5UnPadding(decrypted)

	return string(decrypted), nil
}

func pkcs5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func scanURL(url string) string {
	urlParts := strings.Split(url, "/")
	if contains(urlParts, "album") {
		return "album"
	} else if contains(urlParts, "artist") {
		return "artist"
	} else if contains(urlParts, "playlist") || contains(urlParts, "featured") {
		return "playlist"
	} else if contains(urlParts, "song") {
		return "song"
	}
	return ""
}

func contains(slice []string, keyword string) bool {
	for _, value := range slice {
		if value == keyword {
			return true
		}
	}
	return false
}

func getAlbumID(inputURL string) (string, error) {
	parts := strings.Split(inputURL, "/")
	token := parts[len(parts)-1]

	inputURL = fmt.Sprintf("https://www.jiosaavn.com/api.php?__call=webapi.get&token=%s&type=album&includeMetaTags=0&ctx=web6dot0&api_version=4&_format=json&_marker=0", token)

	req, err := http.NewRequest("GET", inputURL, nil)
	if err != nil {
		return "", fmt.Errorf("new request error: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http client error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body error: %w", err)
	}

	var data Response
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", fmt.Errorf("unmarshal json error: %w", err)
	}

	return data.ID, nil
}

func getAlbum(albumID string) (AlbumResponse, error) {
	requestURL := fmt.Sprintf("https://www.jiosaavn.com/api.php?_format=json&__call=content.getAlbumDetails&albumid=%s", albumID)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return AlbumResponse{}, fmt.Errorf("new request error: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return AlbumResponse{}, fmt.Errorf("http client error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AlbumResponse{}, fmt.Errorf("read response body error: %w", err)
	}

	var data AlbumResponse
	err = json.Unmarshal(body, &data)
	if err != nil {
		return AlbumResponse{}, fmt.Errorf("unmarshal json error: %w", err)
	}

	return data, nil
}

// Function to download the entire album
func downloadAlbum(albumID string) {
	albumJSON, err := getAlbum(albumID)
	if err != nil {
		log.Fatalf("error getting album details: %s\n", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(albumJSON.Songs)) // buffer error channel

	p := mpb.New(mpb.WithWaitGroup(&wg))

	var limit = make(chan struct{}, 20) // Limit of 20 concurrent downloads

	for _, song := range albumJSON.Songs {
		limit <- struct{}{} // Block if there are already 20 downloads in progress

		if err != nil {
			log.Fatalf("error decrypting URL: %s\n", err)
		}

		wg.Add(1)
		go func(song Song) {
			decryptedURL, err := decryptURL(song.EncryptedMediaURL)
			if err != nil {
				log.Fatalf("error decrypting URL: %s\n", err)
			}

			highBitrateURL := strings.Replace(decryptedURL, "_96", "_320", 1)

			downloadSong(highBitrateURL, song.Song, albumJSON.PrimaryArtists, albumJSON.Name, p, &wg, errChan)
			<-limit // Free up a spot once this download is complete
		}(song)

	}

	wg.Wait() // wait for all downloads to finish

	close(errChan) // close error channel

	// Check for errors
	for err := range errChan {
		if err != nil {
			log.Fatalf("error downloading song: %s\n", err)
		}
	}
}

// Function to get the artist ID from the URL
func getArtistID(url string) (string, error) {
	token := url[strings.LastIndex(url, "/")+1:]
	apiURL := fmt.Sprintf("https://www.jiosaavn.com/api.php?__call=webapi.get&token=%s&type=artist&p=&n_song=10&n_album=14&sub_type=&category=&sort_order=&includeMetaTags=0&ctx=web6dot0&api_version=4&_format=json&_marker=0", token)

	// Make the API request
	response, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("error fetching artist ID: %s", err)
	}
	defer response.Body.Close()

	// Parse the JSON response
	var data map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return "", fmt.Errorf("error parsing artist ID response: %s", err)
	}

	// Extract the artist ID
	artistID := data["artistId"].(string)

	return artistID, nil
}

// Function to get the artist JSON
func getArtistJSON(artistID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://www.jiosaavn.com/api.php?_marker=0&_format=json&__call=artist.getArtistPageDetails&artistId=%s", artistID)

	// Make the API request
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching artist JSON: %s", err)
	}
	defer response.Body.Close()

	// Parse the JSON response
	var data map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("error parsing artist JSON response: %s", err)
	}

	return data, nil
}

// Function to get the artist albums IDs
func getArtistAlbumsIDs(artistJSON map[string]interface{}) ([]string, error) {
	topAlbums := artistJSON["topAlbums"].(map[string]interface{})
	totalAlbums := int(topAlbums["total"].(float64))
	fmt.Printf("Total Albums of the Artist: %d\n", totalAlbums)

	var albumIDs []string

	if totalAlbums%10 != 0 {
		totalRequests := (totalAlbums / 10) + 1
		fmt.Printf("Total requests: %d\n", totalRequests)

		for nAlbumPage := 0; nAlbumPage < totalRequests; nAlbumPage++ {
			fmt.Printf("Getting Album page: %d\n", nAlbumPage)
			url := fmt.Sprintf("https://www.saavn.com/api.php?_marker=0&_format=json&__call=artist.getArtistPageDetails&artistId=%s&n_album=10&page=%d", artistJSON["artistId"], nAlbumPage)
			response, err := http.Get(url)
			if err != nil {
				return nil, fmt.Errorf("error fetching artist albums page: %s", err)
			}
			defer response.Body.Close()

			var artistPageData map[string]interface{}
			err = json.NewDecoder(response.Body).Decode(&artistPageData)
			if err != nil {
				return nil, fmt.Errorf("error parsing artist albums page response: %s", err)
			}

			nAlbumsInPage := len(artistPageData["topAlbums"].(map[string]interface{})["albums"].([]interface{}))
			albums := artistPageData["topAlbums"].(map[string]interface{})["albums"].([]interface{})

			for i := 0; i < nAlbumsInPage; i++ {
				albumID := albums[i].(map[string]interface{})["albumid"].(string)
				albumIDs = append(albumIDs, albumID)
			}
		}
	} else {
		fmt.Println("No albums found for the artist.")
		return nil, nil
	}

	fmt.Printf("Total Number of Albums found: %d\n", len(albumIDs))
	return albumIDs, nil
}

// Function to download all albums using album IDs
func downloadAllAlbums(albumIDs []string) {
	var wg sync.WaitGroup
	errChan := make(chan error, len(albumIDs)) // buffer error channel

	for _, albumID := range albumIDs {
		wg.Add(1)
		go func(albumID string) {
			defer wg.Done()
			downloadAlbum(albumID)
		}(albumID)
	}

	wg.Wait() // wait for all downloads to finish

	close(errChan) // close error channel

	// Check for errors
	for err := range errChan {
		if err != nil {
			log.Fatalf("error downloading album: %s\n", err)
		}
	}
}

func downloadSong(songURL, songName, artistName, albumName string, p *mpb.Progress, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	// Create the artist and album folders if they don't exist
	albumPath := filepath.Join(artistName, albumName)
	err := os.MkdirAll(albumPath, os.ModePerm)
	if err != nil {
		errChan <- err
		return
	}

	// Construct the file path for the downloaded song
	filePath := filepath.Join(albumPath, songName+".m4a")

	// Send an HTTP GET request
	resp, err := http.Get(songURL)
	if err != nil {
		errChan <- err
		return
	}
	defer resp.Body.Close()

	// Create a file to write to
	outFile, err := os.Create(filePath)
	if err != nil {
		errChan <- err
		return
	}
	defer outFile.Close()

	// Create a progress bar
	bar := p.AddBar(resp.ContentLength, mpb.PrependDecorators(
		decor.Name(songName),
		decor.CountersKibiByte(" % .2f / % .2f"),
	),
		mpb.AppendDecorators(
			decor.Elapsed(decor.ET_STYLE_GO, decor.WC{}),
		))

	// Create a proxy reader
	barReader := bar.ProxyReader(resp.Body)

	// Write the response body to file
	_, err = io.Copy(outFile, barReader)
	if err != nil {
		errChan <- err
		return
	}

	bar.Abort(false)
}

func main() {
	fmt.Println("Please enter the album URL:")

	var url string
	_, err := fmt.Scanf("%s", &url)
	if err != nil {
		log.Fatalf("error reading URL: %s\n", err)
	}

	result := scanURL(url)

	switch result {
	case "album":
		fmt.Println("Album URL entered.")
		albumID, err := getAlbumID(url)
		if err != nil {
			log.Fatalf("error getting album ID: %s\n", err)
		}

		downloadAlbum(albumID)

	case "artist":
		fmt.Println("Artist URL entered.")
		artistID, err := getArtistID(url)
		if err != nil {
			log.Fatalf("error getting artist ID: %s\n", err)
		}

		// Get the artist JSON
		artistJSON, err := getArtistJSON(artistID)
		if err != nil {
			log.Fatalf("error getting artist JSON: %s\n", err)
		}

		// Get the artist albums IDs
		albumIDs, err := getArtistAlbumsIDs(artistJSON)
		if err != nil {
			log.Fatalf("error getting artist albums IDs: %s\n", err)
		}

		downloadAllAlbums(albumIDs)

	case "playlist":
		fmt.Println("Playlist URL entered.")
		// Implement your logic to download the album from the playlist URL here
		// You can use the `url` variable to extract the playlist ID or other necessary information
		// Then download the album using the obtained playlist information

	case "song":
		fmt.Println("Song URL entered.")
		// Implement your logic to handle the song URL here
		// You can use the `url` variable to download the specific song or perform any other action

	default:
		fmt.Println("Invalid URL entered.")
		// Handle the case when an invalid URL is entered
	}
}
