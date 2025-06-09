package models

type YtDlpVideoResponse struct {
	ID        string  	`json:"id"`
	Title     string  	`json:"title"`
	Duration  float64 	`json:"duration"`
	Views     int     	`json:"view_count"`
	Channel   string  	`json:"channel"`
	Thumbnail string  	`json:"thumbnail"`
}

type YtDlpTrendingMusicResponse struct {
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Entries []YtDlpVideoResponse `json:"entries"`
}