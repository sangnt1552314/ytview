package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sangnt1552314/ytview/internal/models"
	"github.com/sangnt1552314/ytview/internal/services"
)

type App struct {
	app          *tview.Application
	music_list   *tview.Table
	playing_song *models.Video
	playing_box  *tview.TextView
}

func NewApp() *App {
	return &App{
		app:         tview.NewApplication(),
		music_list:  tview.NewTable(),
		playing_box: tview.NewTextView().SetTextAlign(tview.AlignCenter),
	}
}

func (app *App) setTableHeader() {
	// Set headers with styling
	app.music_list.SetCell(0, 0, tview.NewTableCell("Title").
		SetMaxWidth(30).SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))
	app.music_list.SetCell(0, 1, tview.NewTableCell("Channel").
		SetMaxWidth(20).SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))
	app.music_list.SetCell(0, 2, tview.NewTableCell("Duration").
		SetMaxWidth(10).
		SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))

	// Fix header row
	app.music_list.SetFixed(1, 0)
}

func (app *App) performSearch(query string) {
	log.Printf("Performing search for query: %s", query)

	app.music_list.Clear()
	app.setTableHeader()

	songs, err := services.GetSongList(query, 5)
	if err != nil {
		app.music_list.SetCell(1, 0, tview.NewTableCell("Error: "+err.Error()))
		return
	}

	for i, song := range songs {
		// Store the full video object as reference in the first cell
		titleCell := tview.NewTableCell(song.Title).SetReference(&song)
		app.music_list.SetCell(i+1, 0, titleCell)
		app.music_list.SetCell(i+1, 1, tview.NewTableCell(song.Channel))
		app.music_list.SetCell(i+1, 2, tview.NewTableCell(song.Duration))
	}
}

func playMedia(url string) error {
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
				cmd := exec.Command(path, url)
				if err := cmd.Start(); err == nil {
					return nil
				}
			}
		}

		// Fall back to QuickTime Player if VLC is not available
		cmd := exec.Command("open", "-a", "QuickTime Player", url)
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
				cmd := exec.Command(player, url)
				if err := cmd.Start(); err == nil {
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
				cmd := exec.Command(path, url)
				if err := cmd.Start(); err == nil {
					return nil
				}
			}
		}
		return fmt.Errorf("no suitable media player found")

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func (app *App) playSong(song *models.Video) {
	log.Printf("Playing song: %s", song.Title)

	audioUrl, err := services.GetVideoAudioUrl(song.ID)
	if err != nil {
		log.Printf("Error getting video audio url: %v", err)
		return
	}

	log.Printf("Audio URL: %s", audioUrl)

	// Play the media using available player
	if err := playMedia(audioUrl); err != nil {
		log.Printf("Error playing media: %v", err)
		return
	}

	// Update the UI
	app.playing_song = song
	app.playing_box.Clear()
	app.playing_box.SetText("Now Playing: " + song.Title + " - " + song.Channel)
}

func main() {
	// Setup logging
	logFile, err := os.OpenFile("logs/ytview.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Initialize app
	app := NewApp()

	// Containers
	main_box := tview.NewFlex().SetDirection(tview.FlexRow)
	flex_box := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Container - Music box
	music_box := tview.NewFlex()
	music_box.SetDirection(tview.FlexRow)
	music_box.SetBorder(true)
	music_box.SetTitle("Music")
	music_box.SetTitleAlign(tview.AlignLeft)

	app.setTableHeader()

	music_box.AddItem(app.music_list, 0, 1, true)

	// Container - Player box
	player_box := tview.NewFlex().SetDirection(tview.FlexColumn)
	player_box.SetBorder(false)
	player_box.SetTitle("")

	// Set up the playing box
	app.playing_box.SetBorder(true).SetTitle(" 0:00 / 0:00 ")
	app.playing_box.SetText("No Playing Song")
	app.playing_box.SetTextColor(tcell.ColorWhite)

	button_control_box := tview.NewFlex().SetDirection(tview.FlexColumn)
	button_control_box.SetBorder(true).
		SetTitle("Control").
		SetTitleAlign(tview.AlignLeft)

	play_button := tview.NewButton("Play")
	stop_button := tview.NewButton("Stop")

	button_control_box.AddItem(play_button, 0, 1, false)
	button_control_box.AddItem(stop_button, 0, 1, false)

	player_box.AddItem(app.playing_box, 0, 5, false)
	player_box.AddItem(button_control_box, 0, 1, false)

	// Set up table selection handler
	app.music_list.SetSelectable(true, false) // Enable row selection
	app.music_list.SetSelectedFunc(func(row, column int) {
		if row > 0 { // Ignore header row
			cell := app.music_list.GetCell(row, 0)
			video, ok := cell.GetReference().(*models.Video)
			if ok {
				log.Printf("Selected video ID: %s", video.ID)
				app.playSong(video)
			}
		}
	})

	// Search box
	search_box := tview.NewInputField()
	search_box.SetBorder(true)
	search_box.SetLabel("Search: ")
	search_box.SetTitle("Search")
	search_box.SetTitleAlign(tview.AlignLeft)
	search_box.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := search_box.GetText()
			if text != "" {
				app.performSearch(text)
				app.app.SetFocus(app.music_list) // Focus directly on the table for navigation
			}
		}
	})

	// Menu
	menu := tview.NewList()
	menu.AddItem("Settings", "", 's', nil)
	menu.AddItem("Exit", "", 'q', func() {
		app.app.Stop()
	})
	menu.SetBorder(true).SetTitle("Menu")
	menu.SetTitleAlign(tview.AlignLeft)

	// Setup layout
	main_box.AddItem(search_box, 0, 1, true)
	main_box.AddItem(flex_box, 0, 6, false)
	main_box.AddItem(player_box, 0, 1, false)

	flex_box.AddItem(menu, 0, 1, false)
	flex_box.AddItem(music_box, 0, 5, false)

	if err := app.app.SetRoot(main_box, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
