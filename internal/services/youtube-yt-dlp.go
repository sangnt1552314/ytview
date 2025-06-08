package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/sangnt1552314/ytview/internal/models"
)

func getYtDlpPath() string {
	if runtime.GOOS == "windows" {
		return "tools/yt-dlp.exe"
	}
	return "tools/yt-dlp"
}

func GetYtDlpInfo(videoURL string) ([]byte, error) {
	ytDlpPath := getYtDlpPath()
	cmd := exec.Command(ytDlpPath, "-j", videoURL)
	return cmd.Output()
}

func GetTrendingSongListYtDlp(maxResults int) ([]models.Video, error) {
	ytDlpPath := getYtDlpPath()

	args := []string{
		"--flat-playlist",
		"-J",
		"-I",
		fmt.Sprintf("1:%d", maxResults),
		"https://www.youtube.com/feed/trending?bp=4gINGgt5dG1hX2NoYXJ0cw%3D%3D",
	}

	cmd := exec.Command(ytDlpPath, args...)
	stdout, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("Command failed with stderr: %s\n", string(exitErr.Stderr))
		}
		log.Printf("Error running command: %v\n", err)
		return nil, err
	}

	var videos []models.Video
	lines := bytes.Split(stdout, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var item models.YtDlpTrendingMusicResponse
		err := json.Unmarshal(line, &item)
		if err != nil {
			return nil, err
		}

		for _, entry := range item.Entries {
			videos = append(videos, models.Video{
				ID:        entry.ID,
				Title:     entry.Title,
				Thumbnail: entry.Thumbnail,
				Duration:  strconv.Itoa(int(entry.Duration)),
				Views:     strconv.Itoa(entry.Views),
				Channel:   entry.Channel,
			})
		}
	}

	return videos, err
}

func GetSongListYtDlp(query string, maxResults int) ([]models.Video, error) {
	ytDlpPath := getYtDlpPath()

	// query = strings.TrimSpace(query)
	// query = strings.Replace(query, " ", "+", -1)

	args := []string{
		// "--match-filter", "categories~='(?i)Music'",
		"--format", "best",
		"--no-warnings",
		"--no-playlist",
		"--skip-download",
		"--quiet",
		"-j",
		"-S view_count",
		fmt.Sprintf("ytsearch%d:%s", maxResults, query),
	}

	cmd := exec.Command(ytDlpPath, args...)
	stdout, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("Command failed with stderr: %s\n", string(exitErr.Stderr))
		}
		log.Printf("Error running command: %v\n", err)
		return nil, err
	}

	var videos []models.Video
	lines := bytes.Split(stdout, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var item models.YtDlpVideoResponse
		err := json.Unmarshal(line, &item)
		if err != nil {
			return nil, err
		}
		videos = append(videos, models.Video{
			ID:        item.ID,
			Title:     item.Title,
			Thumbnail: item.Thumbnail,
			Duration:  strconv.Itoa(int(item.Duration)),
			Views:     strconv.Itoa(item.Views),
			Channel:   item.Channel,
		})
	}

	return videos, nil
}

func GetVideoAudioUrlYtDlp(videoId string) (string, error) {
	ytDlpPath := getYtDlpPath()

	args := []string{
		"--get-url",
		"--format", "bestaudio/best",
		videoId,
	}

	cmd := exec.Command(ytDlpPath, args...)
	stdout, err := cmd.Output()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("Command failed with stderr: %s\n", string(exitErr.Stderr))
		}
		log.Printf("Error running command: %v\n", err)
		return "", err
	}

	return string(stdout), nil
}
