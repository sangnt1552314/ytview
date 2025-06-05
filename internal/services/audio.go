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
)

func PlayMedia(url string) error {
	// Stop any currently playing media first
	stopCurrentMedia()

	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	switch runtime.GOOS {
	case "darwin":
		// macOS: Try VLC first, then fall back to QuickTime Player
		vlcPaths := []string{
			"/Applications/VLC.app/Contents/MacOS/VLC",
			filepath.Join(os.Getenv("HOME"), "Applications/VLC.app/Contents/MacOS/VLC"),
		}

		// Try VLC first
		for _, path := range vlcPaths {
			if _, err := os.Stat(path); err == nil {
				cmd := exec.Command(path, "--intf", "dummy", url)
				if err := cmd.Start(); err == nil {
					currentCmd = cmd
					return nil
				}
			}
		}

		// Fall back to QuickTime Player if VLC is not available
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

// stopCurrentMedia stops any currently playing media
func stopCurrentMedia() {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if currentCmd != nil && currentCmd.Process != nil {
		// On Unix-like systems, negative PID kills process group
		if runtime.GOOS != "windows" {
			currentCmd.Process.Kill()
		} else {
			exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(currentCmd.Process.Pid)).Run()
		}
		currentCmd = nil
	}
}

// Cleanup should be called when your application exits
func Cleanup() {
	stopCurrentMedia()
}
