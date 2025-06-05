package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
	"github.com/sangnt1552314/ytview/internal/models"
)

func convertDuration(duration string) string {
	//PT4M13S -> 4:13
	duration = strings.Replace(duration, "PT", "", 1)
	duration = strings.Replace(duration, "H", ":", 1)
	duration = strings.Replace(duration, "M", ":", 1)
	duration = strings.Replace(duration, "S", "", 1)
	return duration
}

func GetSongList(query string, maxResults int) ([]models.Video, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	if maxResults == 0 {
		maxResults = 5
	}

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("YOUTUBE_API_KEY not found in environment")
	}

	query = strings.TrimSpace(query)
	query = strings.Replace(query, " ", "+", -1)
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&key=%s&type=video&maxResults=%d", query, apiKey, maxResults)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var ytResp models.YouTubeResponse
	if err := json.NewDecoder(resp.Body).Decode(&ytResp); err != nil {
		log.Printf("Error at ytResp: %v", err)
		return nil, err
	}

	var videos []models.Video
	for _, item := range ytResp.Items {
		detailUrl := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?part=contentDetails,statistics&id=%s&key=%s", item.ID.VideoId, apiKey)
		detailResp, err := http.Get(detailUrl)
		if err != nil {
			return nil, err
		}
		defer detailResp.Body.Close()

		var ytDetailResp models.YoutubeVideoDetailResponse
		if err := json.NewDecoder(detailResp.Body).Decode(&ytDetailResp); err != nil {
			log.Printf("Error at detailResp: %v", err)
			return nil, err
		}

		video := models.Video{
			ID:       item.ID.VideoId,
			Title:    item.Snippet.Title,
			Channel:  item.Snippet.ChannelTitle,
			Views:    ytDetailResp.Items[0].Statistics.ViewCount,
			Duration: convertDuration(ytDetailResp.Items[0].ContentDetails.Duration),
		}
		videos = append(videos, video)
	}

	return videos, nil
}

func GetVideoAudioUrl(videoId string) (string, error) {
	log.Println("Getting video audio url for videoId: ", videoId)
	client := youtube.Client{}

	video, err := client.GetVideo(videoId)
	if err != nil {
		return "", fmt.Errorf("failed to get video: %w", err)
	}

	for {
		format := video.Formats.WithAudioChannels()
		audio, err := client.GetStreamURL(video, &format[0])
		if err != nil {
			continue
		}

		data, _ := http.Get(audio)
		if data.StatusCode == 200 {
			return audio, nil
		}
	}
}
