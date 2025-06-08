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

// Windows process creation flags
const (
	CREATE_NO_WINDOW = 0x08000000
	DETACHED_PROCESS = 0x00000008
)

// WindowsMediaPlayer handles audio playback using Windows Media Player
type WindowsMediaPlayer struct {
	wmpPath     string
	isAvailable bool
}

// newWindowsMediaPlayer creates a new Windows Media Player handler
func newWindowsMediaPlayer() *WindowsMediaPlayer {
	wmpPaths := []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Windows Media Player", "wmplayer.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Windows Media Player", "wmplayer.exe"),
	}

	var wmpPath string
	for _, path := range wmpPaths {
		if _, err := os.Stat(path); err == nil {
			wmpPath = path
			break
		}
	}

	return &WindowsMediaPlayer{
		wmpPath:     wmpPath,
		isAvailable: wmpPath != "",
	}
}

// playInBackground starts WMP in background with minimal UI
func (w *WindowsMediaPlayer) playInBackground(url string) (*exec.Cmd, error) {
	if !w.isAvailable {
		return nil, fmt.Errorf("Windows Media Player is not available")
	}

	// Create a new process with hidden window
	si := &syscall.StartupInfo{
		Flags:      syscall.STARTF_USESHOWWINDOW,
		ShowWindow: syscall.SW_HIDE,
	}
	pi := &syscall.ProcessInformation{}

	// Convert command line to UTF16 as required by Windows API
	argv := syscall.StringToUTF16Ptr(fmt.Sprintf(`"%s" /play /close "%s"`, w.wmpPath, url))

	// Create process in background
	err := syscall.CreateProcess(
		nil,
		argv,
		nil,
		nil,
		false,
		CREATE_NO_WINDOW|DETACHED_PROCESS,
		nil,
		nil,
		si,
		pi,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start Windows Media Player: %v", err)
	}

	// Close handles to avoid resource leaks
	syscall.CloseHandle(pi.Thread)
	syscall.CloseHandle(pi.Process)

	// Create exec.Cmd for process management
	cmd := exec.Command(w.wmpPath)
	cmd.Process = &os.Process{Pid: int(pi.ProcessId)}

	return cmd, nil
}

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
	case "darwin", "windows":
		// Try VLC first for both Windows and macOS
		vlcPaths := []string{}
		if runtime.GOOS == "darwin" {
			vlcPaths = []string{
				"/Applications/VLC.app/Contents/MacOS/VLC",
				filepath.Join(os.Getenv("HOME"), "Applications/VLC.app/Contents/MacOS/VLC"),
			}
		} else {
			vlcPaths = []string{
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "VideoLAN", "VLC", "vlc.exe"),
				filepath.Join(os.Getenv("ProgramFiles"), "VideoLAN", "VLC", "vlc.exe"),
			}
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
					go func() {
						cmd.Wait()
						cmdMutex.Lock()
						if currentCmd == cmd {
							currentCmd = nil
							isPaused = false
							lastUrl = ""
						}
						cmdMutex.Unlock()
					}()

					// Wait for VLC's HTTP interface to initialize with retries
					maxRetries := 10
					for i := 0; i < maxRetries; i++ {
						time.Sleep(100 * time.Millisecond)
						// Try to connect to VLC's HTTP interface
						resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml", "ytview", vlcPort))
						if err == nil {
							resp.Body.Close()
							return nil
						}
					}

					// If we get here, VLC started but HTTP interface failed to initialize
					stopCurrentMedia()
					return fmt.Errorf("VLC HTTP interface failed to initialize")
				}
			}
		}

		// OS-specific fallbacks if VLC fails
		if runtime.GOOS == "darwin" {
			cmd := exec.Command("open", "-g", "-a", "QuickTime Player", url)
			if err := cmd.Start(); err == nil {
				currentCmd = cmd
				go func() {
					cmd.Wait()
					cmdMutex.Lock()
					if currentCmd == cmd {
						currentCmd = nil
						isPaused = false
						lastUrl = ""
					}
					cmdMutex.Unlock()
				}()
				return nil
			}
		} else if runtime.GOOS == "windows" && wmp != nil && wmp.isAvailable {
			cmd, err := wmp.playInBackground(url)
			if err == nil {
				currentCmd = cmd
				wmpCmd = cmd
				go func() {
					for {
						if IsMediaFinished() {
							cmdMutex.Lock()
							if currentCmd == cmd {
								currentCmd = nil
								wmpCmd = nil
								isPaused = false
								lastUrl = ""
							}
							cmdMutex.Unlock()
							break
						}
						time.Sleep(500 * time.Millisecond)
					}
				}()
				return nil
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
					go func() {
						cmd.Wait()
						cmdMutex.Lock()
						if currentCmd == cmd {
							currentCmd = nil
							isPaused = false
							lastUrl = ""
						}
						cmdMutex.Unlock()
					}()
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

	// For both Windows and macOS, try VLC HTTP interface first
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		// Send pause command to VLC's HTTP interface with retries
		maxRetries := 3
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_pause", "ytview", vlcPort))
			if err == nil {
				resp.Body.Close()
				isPaused = true
				return nil
			}
			lastErr = err
			time.Sleep(100 * time.Millisecond)
		}

		// If VLC HTTP interface fails, try OS-specific fallbacks
		if runtime.GOOS == "darwin" {
			// Fallback to QuickTime if VLC command failed
			pauseCmd := exec.Command("killall", "-STOP", "QuickTime Player")
			err := pauseCmd.Run()
			if err == nil {
				isPaused = true
				return nil
			}
			return fmt.Errorf("failed to pause media: VLC failed with %v, QuickTime failed with %v", lastErr, err)
		} else if runtime.GOOS == "windows" {
			// For Windows, we might want to add a WMP-specific pause command here
			return fmt.Errorf("failed to pause media: VLC HTTP interface error: %v", lastErr)
		}
	}
	return fmt.Errorf("pause not supported on this OS or player")
}

// resumeMedia resumes the paused media
func resumeMedia() error {
	if !isPaused || currentCmd == nil {
		return fmt.Errorf("no paused media to resume")
	}

	// For both Windows and macOS, try VLC HTTP interface first
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		// Send play command to VLC's HTTP interface with retries
		maxRetries := 3
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_play", "ytview", vlcPort))
			if err == nil {
				resp.Body.Close()
				isPaused = false
				return nil
			}
			lastErr = err
			time.Sleep(100 * time.Millisecond)
		}

		// If VLC HTTP interface fails, try OS-specific fallbacks
		if runtime.GOOS == "darwin" {
			// Fallback to QuickTime if VLC command failed
			resumeCmd := exec.Command("killall", "-CONT", "QuickTime Player")
			err := resumeCmd.Run()
			if err == nil {
				isPaused = false
				return nil
			}
			return fmt.Errorf("failed to resume media: VLC failed with %v, QuickTime failed with %v", lastErr, err)
		} else if runtime.GOOS == "windows" {
			// For Windows, we might want to add a WMP-specific resume command here
			return fmt.Errorf("failed to resume media: VLC HTTP interface error: %v", lastErr)
		}
	}
	return fmt.Errorf("resume not supported on this OS or player")
}

// stopCurrentMedia stops any currently playing media
func stopCurrentMedia() {
	if currentCmd != nil && currentCmd.Process != nil {
		// Try to stop via HTTP interface first for both Windows and macOS
		if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
			// Try multiple times to stop via HTTP interface
			maxRetries := 3
			for i := 0; i < maxRetries; i++ {
				resp, err := http.Get(fmt.Sprintf("http://:%s@localhost:%s/requests/status.xml?command=pl_stop", "ytview", vlcPort))
				if err == nil {
					resp.Body.Close()
					time.Sleep(100 * time.Millisecond)
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Force kill the process if needed
		if runtime.GOOS == "windows" {
			// Try to cleanly terminate VLC first
			exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(currentCmd.Process.Pid)).Run()
		} else {
			currentCmd.Process.Kill()
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

// IsMediaFinished checks if the media playback has finished
func IsMediaFinished() bool {
	cmdMutex.Lock()
	defer cmdMutex.Unlock()

	if currentCmd == nil || currentCmd.Process == nil {
		return true
	}

	if runtime.GOOS == "windows" {
		// Special handling for Windows Media Player
		if wmpCmd != nil && wmpCmd == currentCmd {
			// Check if process still exists
			process, err := os.FindProcess(currentCmd.Process.Pid)
			if err != nil || process == nil {
				currentCmd = nil
				wmpCmd = nil
				isPaused = false
				lastUrl = ""
				return true
			}

			// Try to get process info - will fail if process is gone
			h, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(process.Pid))
			if err != nil {
				currentCmd = nil
				wmpCmd = nil
				isPaused = false
				lastUrl = ""
				return true
			}
			syscall.CloseHandle(h)
		} else {
			// For other players like VLC
			if currentCmd.ProcessState != nil && currentCmd.ProcessState.Exited() {
				currentCmd = nil
				isPaused = false
				lastUrl = ""
				return true
			}
		}
		return false
	}

	// Unix-like systems
	if process, err := os.FindProcess(currentCmd.Process.Pid); err == nil {
		// On Unix systems, FindProcess always succeeds, so we need to send signal 0 to test if process exists
		err := process.Signal(syscall.Signal(0))
		if err != nil {
			// Process not found or finished
			currentCmd = nil
			isPaused = false
			lastUrl = ""
			return true
		}

		// Additional check: see if process has exited
		if currentCmd.ProcessState != nil && currentCmd.ProcessState.Exited() {
			currentCmd = nil
			isPaused = false
			lastUrl = ""
			return true
		}
	}

	return false
}

var (
	wmp    *WindowsMediaPlayer
	wmpCmd *exec.Cmd
)

func init() {
	if runtime.GOOS == "windows" {
		wmp = newWindowsMediaPlayer()
	}
}
