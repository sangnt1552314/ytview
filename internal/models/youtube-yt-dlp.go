package models

type YtDlpVideoResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Duration  int `json:"duration"`
	Views     int `json:"view_count"`
	Channel   string `json:"channel"`
	Thumbnail string `json:"thumbnail"`
}
