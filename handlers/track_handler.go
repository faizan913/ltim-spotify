package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/faizan/spotify/config"
	"github.com/faizan/spotify/models"
	"github.com/gin-gonic/gin"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
	"gorm.io/gorm"
)

var (
	clientID     = "ea1a7d9fe03340d1a49abf8aac30817c"
	clientSecret = "8482469c7f5149059bba601edc00f18a"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()

	r := gin.Default()

	r.POST("/fetch-and-store", FetchAndStore)

	r.GET("/track/:isrc", GetByISRC)

	r.GET("/tracks-by-artist/:artist", GetByArtistName)

	//router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := "8080"
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}

	return router
}

// @Summary Fetch and store track metadata
// @Description Fetches track metadata from Spotify based on ISRC and stores it in the local DB.
// @Tags tracks
// @Accept json
// @Produce json
// @Param isrc query string true "ISRC code of the track"
// @Success 200 {object} TrackResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /fetch-and-store [post]
func FetchAndStore(c *gin.Context) {
	isrc := c.Query("isrc")
	if isrc == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ISRC parameter is required"})
		return
	}

	track, err := fetchTrackMetadata(isrc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Store the track metadata into the database
	if err := storeTrackMetadata(config.DB, isrc, track); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image_uri":  track.ImageURL,
		"title":      track.Name,
		"artists":    track.Artists,
		"popularity": track.Popularity,
	})
}

// GetByISRC godoc
// @Summary Get track details by ISRC
// @Description Retrieves track details from the local database based on the provided ISRC code.
// @Tags tracks
// @Accept json
// @Produce json
// @Param isrc path string true "ISRC code of the track"
// @Success 200 {object} TrackResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /track/{isrc} [get]
func GetByISRC(c *gin.Context) {
	isrc := c.Param("isrc")

	var track models.Track
	if err := config.DB.Preload("Artists").Where("isrc = ?", isrc).First(&track).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Track not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         track.ID,
		"image_uri":  track.ImageURL,
		"title":      track.Name,
		"artists":    track.Artists,
		"popularity": track.Popularity,
	})
}

// GetByArtistName godoc
// @Summary Get tracks by artist name
// @Description Retrieves tracks from the local database based on the provided artist name.
// @Tags tracks
// @Accept json
// @Produce json
// @Param artist path string true "Artist name to search for in tracks"
// @Success 200 {array} TrackResponse
// @Failure 500 {object} ErrorResponse
// @Router /tracks-by-artist/{artist} [get]
func GetByArtistName(c *gin.Context) {
	artist := c.Param("artist")

	var tracks []models.Track
	if err := config.DB.
		Preload("Artists").
		Joins("JOIN artists ON tracks.id = artists.track_id").
		Where("artists.name LIKE ?", "%"+artist+"%").
		Find(&tracks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tracks"})
		return
	}

	var response []gin.H
	for _, track := range tracks {
		response = append(response, gin.H{
			"id":         track.ID,
			"image_uri":  track.ImageURL,
			"title":      track.Name,
			"artists":    track.Artists,
			"popularity": track.Popularity,
		})
	}

	c.JSON(http.StatusOK, response)
}

func fetchTrackMetadata(isrc string) (*models.Track, error) {

	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotify.TokenURL,
	}
	token, err := config.Token(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Spotify API token: %v", err)
	}

	client := spotify.Authenticator{}.NewClient(token)

	// Search for tracks with the given ISRC
	results, err := client.Search(isrc, spotify.SearchTypeTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to search for track: %v", err)
	}

	if len(results.Tracks.Tracks) == 0 {
		return nil, fmt.Errorf("no track found for the given ISRC")
	}

	highestPopularityTrack := results.Tracks.Tracks[0]
	for _, track := range results.Tracks.Tracks {
		if track.Popularity > highestPopularityTrack.Popularity {
			highestPopularityTrack = track
		}
	}

	artists := make([]string, len(highestPopularityTrack.Artists))
	for i, artist := range highestPopularityTrack.Artists {
		artists[i] = artist.Name
	}

	track := &models.Track{
		ISRC:       isrc,
		ImageURL:   highestPopularityTrack.Album.Images[0].URL,
		Name:       highestPopularityTrack.Name,
		Popularity: highestPopularityTrack.Popularity,
		Artists:    []models.Artist{{Name: artists[0]}},
	}

	return track, nil
}

func storeTrackMetadata(db *gorm.DB, isrc string, track *models.Track) error {
	var existingTrack models.Track
	if err := db.Preload("Artists").Where("isrc = ?", isrc).First(&existingTrack).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newTrack := &models.Track{
				ISRC:       isrc,
				ImageURL:   track.ImageURL,
				Name:       track.Name,
				Popularity: track.Popularity,
			}

			for _, artistName := range track.Artists {
				newTrack.Artists = append(newTrack.Artists, models.Artist{Name: artistName.Name})
			}

			if err := db.Create(newTrack).Error; err != nil {
				return fmt.Errorf("failed to store track metadata: %v", err)
			}
		} else {
			return fmt.Errorf("failed to query track metadata: %v", err)
		}
	} else {
		existingTrack.ImageURL = track.ImageURL
		existingTrack.Name = track.Name
		existingTrack.Popularity = track.Popularity

		existingTrack.Artists = nil
		for _, artistName := range track.Artists {
			existingTrack.Artists = append(existingTrack.Artists, models.Artist{Name: artistName.Name})
		}

		if err := db.Save(&existingTrack).Error; err != nil {
			return fmt.Errorf("failed to update track metadata: %v", err)
		}
	}

	return nil
}
