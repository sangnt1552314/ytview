package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	currentCmd *exec.Cmd
	cmdMutex   sync.Mutex
	isPaused   bool
	lastUrl    string
)

// PlayMedia starts playing the media from the given URL
func PlayMedia(url string) error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	// If we're resuming the same track that was paused
	if isPaused && url == lastUrl {
		return resumeMedia()
	}

	// Otherwise, stop current playback and start new
	stopCurrentMedia()
	lastUrl = url
	isPaused = false

	switch runtime.GOOS {
	case "darwin":
		vlcPaths := []string{
			"/Applications/VLC.app/Contents/MacOS/VLC",
			filepath.Join(os.Getenv("HOME"), "Applications/VLC.app/Contents/MacOS/VLC"),
		}

		for _, path := range vlcPaths {
			if _, err := os.Stat(path); err == nil {
				cmd := exec.Command(path, "--intf", "dummy", url)
				if err := cmd.Start(); err == nil {
					currentCmd = cmd
					return nil
				}
			}
		}

		cmd := exec.Command("open", "-g", "-a", "QuickTime Player", url)
		if err := cmd.Start(); err == nil {
			currentCmd = cmd
			return nil
		}
		return cmd.Start()

	case "windows":
		// Windows: Try Windows Media Player, then VLC
		mediaPlayers := []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Windows Media Player", "wmplayer.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Windows Media Player", "wmplayer.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "VideoLAN", "VLC", "vlc.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "VideoLAN", "VLC", "vlc.exe"),
		}

		for _, player := range mediaPlayers {
			if _, err := os.Stat(player); err == nil {
				cmd := exec.Command(player, "--qt-start-minimized", url)
				if err := cmd.Start(); err == nil {
					currentCmd = cmd
					return nil
				}
			}
		}
		return fmt.Errorf("no suitable media player found")

	case "linux":
		// Linux: Try multiple players in order
		players := []string{"vlc", "mpv", "mplayer"}
		for _, player := range players {
			if path, err := exec.LookPath(player); err == nil {
				cmd := exec.Command(path, "--intf", "dummy", url)
				if err := cmd.Start(); err == nil {
					currentCmd = cmd
					return nil
				}
			}
		}
		return fmt.Errorf("no suitable media player found")

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// PauseMedia pauses the currently playing media
func PauseMedia() error {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if currentCmd == nil || currentCmd.Process == nil {
		return fmt.Errorf("no media is playing")
	}

	if runtime.GOOS == "darwin" {
		// For VLC
		pauseCmd := exec.Command("killall", "-STOP", "VLC")
		err := pauseCmd.Run()
		if err == nil {
			isPaused = true
			return nil
		}
		// For QuickTime
		pauseCmd = exec.Command("killall", "-STOP", "QuickTime Player")
		err = pauseCmd.Run()
		if err == nil {
			isPaused = true
			return nil
		}
	}
	return fmt.Errorf("pause not supported on this OS")
}

// resumeMedia resumes the paused media
func resumeMedia() error {
	if !isPaused || currentCmd == nil {
		return fmt.Errorf("no paused media to resume")
	}

	if runtime.GOOS == "darwin" {
		// For VLC
		resumeCmd := exec.Command("killall", "-CONT", "VLC")
		err := resumeCmd.Run()
		if err == nil {
			isPaused = false
			return nil
		}
		// For QuickTime
		resumeCmd = exec.Command("killall", "-CONT", "QuickTime Player")
		err = resumeCmd.Run()
		if err == nil {
			isPaused = false
			return nil
		}
	}
	return fmt.Errorf("resume not supported on this OS")
}

// stopCurrentMedia stops any currently playing media
func stopCurrentMedia() {
	if currentCmd != nil && currentCmd.Process != nil {
		if runtime.GOOS != "windows" {
			currentCmd.Process.Kill()
		} else {
			exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(currentCmd.Process.Pid)).Run()
		}
		currentCmd = nil
		isPaused = false
		lastUrl = ""
	}
}

// Cleanup stops any playing media and cleans up resources
func Cleanup() {
	stopCurrentMedia()
}

// GetPlayerState returns the current state of the player
func GetPlayerState() string {
	if currentCmd == nil {
		return "stopped"
	}
	if isPaused {
		return "paused"
	}
	return "playing"
}
