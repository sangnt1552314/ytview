package models

type Video struct {
	Title     string `json:"title"`
	Channel   string `json:"author"`
	Views     string `json:"views"`
	Duration  string `json:"duration"`
	ID        string `json:"id"`
	Thumbnail string `json:"thumb"`
}

type YoutubeVideoDetailResponse struct {
	Items []struct {
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
		Statistics struct {
			ViewCount string `json:"viewCount"`
		} `json:"statistics"`
	} `json:"items"`
}

type YouTubeResponse struct {
	Items []struct {
		ID struct {
			VideoId string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelTitle string `json:"channelTitle"`
			ChannelId    string `json:"channelId"`
			Thumbnails   struct {
				Default struct {
					Url string `json:"url"`
				} `json:"default"`
				Medium struct {
					Url string `json:"url"`
				} `json:"medium"`
				High struct {
					Url string `json:"url"`
				} `json:"high"`
			} `json:"thumbnails"`
		} `json:"snippet"`
	} `json:"items"`
}
