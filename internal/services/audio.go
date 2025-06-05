package services

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	currentCmd *exec.Cmd
	cmdMutex   sync.Mutex
	isPaused   bool
	lastUrl    string
	vlcPort    = "8080" // VLC HTTP interface port
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
				// Start VLC with HTTP interface
				cmd := exec.Command(path,
					"--intf", "http", // Enable HTTP interface
					"--http-port", vlcPort, // Set HTTP port
					"--http-password", "ytview", // Set password for HTTP interface
					"--extraintf", "http", // Add HTTP as extra interface
					"--no-video",      // Disable video output
					"--play-and-exit", // Exit when playback ends
					url)

				if err := cmd.Start(); err == nil {
					currentCmd = cmd
					// Give VLC a moment to start up its HTTP interface
					time.Sleep(100 * time.Millisecond)
					return nil
				}
			}
		}

		// Fallback to QuickTime if VLC is not available
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
		// Send pause command to VLC's HTTP interface
		resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_pause", "ytview", vlcPort))
		if err == nil {
			resp.Body.Close()
			isPaused = true
			return nil
		}

		// Fallback to QuickTime if VLC command failed
		pauseCmd := exec.Command("killall", "-STOP", "QuickTime Player")
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
		// Send play command to VLC's HTTP interface
		resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_play", "ytview", vlcPort))
		if err == nil {
			resp.Body.Close()
			isPaused = false
			return nil
		}

		// Fallback to QuickTime if VLC command failed
		resumeCmd := exec.Command("killall", "-CONT", "QuickTime Player")
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
			// Try to stop via HTTP interface first
			if runtime.GOOS == "darwin" {
				resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_stop", "ytview", vlcPort))
				if err == nil {
					resp.Body.Close()
					time.Sleep(100 * time.Millisecond) // Give VLC time to stop
				}
			}
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
	if IsMediaFinished() {
		return "stopped"
	}
	if currentCmd == nil {
		return "stopped"
	}
	if isPaused {
		return "paused"
	}
	return "playing"
}

// Add a function to check if the media is still playing
func IsMediaFinished() bool {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if currentCmd == nil || currentCmd.Process == nil {
		return true
	}

	// For VLC, try to get process state
	if runtime.GOOS != "windows" {
		if process, err := os.FindProcess(currentCmd.Process.Pid); err == nil {
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// Process not found or finished
				currentCmd = nil
				isPaused = false
				return true
			}
		}
	}

	return false
}
