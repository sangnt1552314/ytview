package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sangnt1552314/ytview/internal/models"
	"github.com/sangnt1552314/ytview/internal/services"
)

type App struct {
	app            *tview.Application
	music_list     *tview.Table
	playing_song   *models.Video
	playing_url    string
	playing_box    *tview.TextView
	control_button *tview.Button
	timer          *time.Timer
	start_time     time.Time
	duration       time.Duration
	elapsed        time.Duration
}

func NewApp() *App {
	button := tview.NewButton("▶️ Play")
	button.SetActivatedStyle(tcell.Style{}.Background(tcell.ColorBlack))
	button.SetStyle(tcell.Style{}.Background(tcell.ColorBlack))

	return &App{
		app:            tview.NewApplication(),
		music_list:     tview.NewTable(),
		playing_box:    tview.NewTextView().SetTextAlign(tview.AlignCenter),
		control_button: button,
	}
}

func (app *App) setMusicTableHeader() {
	// Set headers with styling
	app.music_list.SetCell(0, 0, tview.NewTableCell("Title").
		SetMaxWidth(18).SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))
	app.music_list.SetCell(0, 1, tview.NewTableCell("Channel").
		SetMaxWidth(9).SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))
	app.music_list.SetCell(0, 2, tview.NewTableCell("Duration").
		SetMaxWidth(5).
		SetSelectable(false).
		SetTextColor(tcell.ColorYellow).
		SetAttributes(tcell.AttrBold))

	// Fix header row
	app.music_list.SetFixed(1, 0)
}

func (app *App) performSearch(query string, maxResults int) {
	app.music_list.Clear()
	app.setMusicTableHeader()

	songs, err := services.GetSongListYtDlp(query, maxResults)

	if err != nil {
		app.music_list.SetCell(1, 0, tview.NewTableCell("Error: "+err.Error()))
		return
	}

	for i, song := range songs {
		duration := formatDuration(parseDuration(song.Duration))
		titleCell := tview.NewTableCell(song.Title).SetReference(&song)

		app.music_list.SetCell(i+1, 0, titleCell)
		app.music_list.SetCell(i+1, 1, tview.NewTableCell(song.Channel))
		app.music_list.SetCell(i+1, 2, tview.NewTableCell(duration)) // Use formatted duration
	}
}

func (app *App) initMusicData(maxResults int) {
	// Show loading message
	app.music_list.SetCell(1, 0, tview.NewTableCell("Loading trending songs..."))

	// Run the data fetching in a goroutine
	go func() {
		songs, err := services.GetTrendingSongListYtDlp(maxResults)

		// Use QueueUpdateDraw to safely update UI from goroutine
		app.app.QueueUpdateDraw(func() {
			app.music_list.Clear()
			app.setMusicTableHeader()

			if err != nil {
				app.music_list.SetCell(1, 0, tview.NewTableCell("Error: "+err.Error()))
				return
			}

			for i, song := range songs {
				duration := formatDuration(parseDuration(song.Duration))
				titleCell := tview.NewTableCell(song.Title).SetReference(&song)

				app.music_list.SetCell(i+1, 0, titleCell)
				app.music_list.SetCell(i+1, 1, tview.NewTableCell(song.Channel))
				app.music_list.SetCell(i+1, 2, tview.NewTableCell(duration))
			}
		})
	}()
}

func (app *App) updateControlButton() {
	state := services.GetPlayerState()
	if state == "playing" {
		app.control_button.SetLabel("⏸️ Pause")
		app.playing_box.SetTextColor(tcell.ColorGreen)
		app.playing_box.SetTitleColor(tcell.ColorGreen)
		app.start_time = time.Now().Add(-app.elapsed)
		if app.timer != nil {
			app.timer.Reset(time.Second)
		}
	} else {
		app.control_button.SetLabel("▶️ Play")
		app.playing_box.SetTextColor(tcell.ColorYellow)
		app.playing_box.SetTitleColor(tcell.ColorYellow)
		if state == "paused" {
			app.elapsed = time.Since(app.start_time)
		}
		if state == "stopped" {
			app.elapsed = app.duration
		}
	}
	app.updateTimeDisplay()
}

func (app *App) playSong(song *models.Video) {
	if app.timer != nil {
		app.timer.Stop()
	}

	audioUrl, err := services.GetVideoAudioUrlYtDlp(song.ID)
	if err != nil {
		log.Printf("Error getting video audio url: %v", err)
		return
	}

	if err := services.PlayMedia(audioUrl); err != nil {
		log.Printf("Error playing media: %v", err)
		return
	}

	app.playing_song = song
	app.playing_url = audioUrl
	app.duration = parseDuration(song.Duration)
	app.start_time = time.Now()
	app.elapsed = 0

	// Create and start the timer
	app.timer = time.NewTimer(time.Second)
	go func() {
		for range app.timer.C {
			app.app.QueueUpdateDraw(func() {
				app.updateTimeDisplay()
			})
			app.timer.Reset(time.Second)
		}
	}()

	if services.GetPlayerState() == "stopped" {
		app.app.QueueUpdateDraw(func() {
			app.control_button.SetLabel("▶️ Play")
			app.playing_box.SetTextColor(tcell.ColorYellow)
			app.playing_box.SetTitleColor(tcell.ColorYellow)
			app.elapsed = app.duration
			app.updateTimeDisplay()
		})
		if app.timer != nil {
			app.timer.Stop()
		}
	}

	app.playing_box.Clear()
	app.playing_box.SetText("Now Playing: " + song.Title + " - " + song.Channel)
	app.updateControlButton()
	app.updateTimeDisplay()
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

func parseDuration(dur string) time.Duration {
	parts := strings.Split(dur, ":")
	if len(parts) == 1 {
		// Only seconds
		sec, _ := strconv.Atoi(parts[0])
		return time.Duration(sec) * time.Second
	} else if len(parts) == 2 {
		// Minutes:Seconds
		min, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return time.Duration(min)*time.Minute + time.Duration(sec)*time.Second
	} else if len(parts) == 3 {
		// Hours:Minutes:Seconds
		hour, _ := strconv.Atoi(parts[0])
		min, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.Atoi(parts[2])
		return time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute + time.Duration(sec)*time.Second
	}
	return 0
}

func (app *App) updateTimeDisplay() {
	if app.playing_song == nil {
		app.playing_box.SetTitle(" 0:00 / 0:00 ")
		return
	}

	var elapsed time.Duration
	state := services.GetPlayerState()

	if state == "playing" {
		elapsed = time.Since(app.start_time)
		app.playing_box.SetTextColor(tcell.ColorGreen)
		app.playing_box.SetTitleColor(tcell.ColorGreen)
		if app.timer != nil {
			app.timer.Reset(time.Second)
		}
	} else if state == "paused" {
		elapsed = app.elapsed
		app.playing_box.SetTextColor(tcell.ColorYellow)
		app.playing_box.SetTitleColor(tcell.ColorYellow)
	} else if state == "stopped" {
		elapsed = app.duration // Show full duration when stopped
		app.playing_box.SetTextColor(tcell.ColorYellow)
		app.playing_box.SetTitleColor(tcell.ColorYellow)
		app.control_button.SetLabel("▶️ Play")
		if app.timer != nil {
			app.timer.Stop()
		}
	}

	if elapsed > app.duration {
		elapsed = app.duration
	}

	title := fmt.Sprintf(" %s / %s ",
		formatDuration(elapsed),
		formatDuration(app.duration))

	app.playing_box.SetTitle(title)
}

func main() {
	// Ensure logs directory exists
	if err := os.MkdirAll("storage/logs", 0755); err != nil {
		panic(fmt.Errorf("failed to create logs directory: %w", err))
	}

	// Setup logging
	logFile, err := os.OpenFile("storage/logs/ytview.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Setup signal handling for cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-c
		log.Printf("Received signal %v, cleaning up...", sig)
		services.Cleanup()
		os.Exit(0)
	}()

	// Initialize app
	app := NewApp()

	// Add input capture to handle Ctrl+C and 'q' globally
	app.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC || event.Rune() == 'q' {
			if app.timer != nil {
				app.timer.Stop()
			}
			services.Cleanup()
			app.app.Stop()
			return nil
		}
		return event
	})

	// Containers
	main_box := tview.NewFlex()
	main_box.SetDirection(tview.FlexRow)
	main_box.SetFullScreen(true)

	flex_box := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Container - Music box
	music_box := tview.NewFlex()
	music_box.SetDirection(tview.FlexRow)
	music_box.SetBorder(true)
	music_box.SetTitle("Music")
	music_box.SetTitleAlign(tview.AlignLeft)

	app.setMusicTableHeader()
	app.initMusicData(5)

	music_box.AddItem(app.music_list, 0, 1, true)

	// Container - Playlist box
	playlist_box := tview.NewFlex()
	playlist_box.SetDirection(tview.FlexRow)
	playlist_box.SetBorder(true)
	playlist_box.SetTitle("Playlist")
	playlist_box.SetTitleAlign(tview.AlignLeft)

	// Container - Content box
	content_box := tview.NewFlex().SetDirection(tview.FlexRow)
	content_box.AddItem(music_box, 0, 1, false)
	content_box.AddItem(playlist_box, 0, 1, false)

	// Container - Player box
	player_box := tview.NewFlex().SetDirection(tview.FlexColumn)
	player_box.SetBorder(false)
	player_box.SetTitle("")

	// Set up the playing box
	app.playing_box.SetBorder(true).
		SetTitle(" 0:00 / 0:00 ").
		SetTitleColor(tcell.ColorYellow)
	app.playing_box.SetText("No Playing Song")
	app.playing_box.SetTextColor(tcell.ColorYellow)

	button_control_box := tview.NewFlex()
	button_control_box.SetBorder(true)

	app.control_button.SetSelectedFunc(func() {
		if app.playing_song == nil {
			return
		}

		state := services.GetPlayerState()
		if state == "playing" {
			services.PauseMedia()
		} else {
			if services.IsMediaFinished() {
				app.playSong(app.playing_song)
			} else {
				services.PlayMedia(app.playing_url)
			}
		}
		app.updateControlButton()
	})

	button_control_box.AddItem(app.control_button, 0, 1, true)

	player_box.AddItem(app.playing_box, 0, 5, false)
	player_box.AddItem(button_control_box, 0, 1, false)

	// Set up table selection handler
	app.music_list.SetSelectable(true, false) // Enable row selection
	app.music_list.SetSelectedFunc(func(row, column int) {
		if row > 0 { // Ignore header row
			cell := app.music_list.GetCell(row, 0)
			video, ok := cell.GetReference().(*models.Video)
			if ok {
				app.playSong(video)
			}
		}
	})

	// Header box
	header_box := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Status Box
	status_box := tview.NewTextView()
	status_box.SetBorder(true)
	status_box.SetTitle("Status")
	status_box.SetTitleAlign(tview.AlignLeft)
	status_box.SetText("...")

	// Search box
	search_box := tview.NewInputField()
	search_box.SetBorder(true)
	search_box.SetTitle("Search")
	search_box.SetFieldBackgroundColor(tcell.ColorNone)
	search_box.SetFieldTextColor(tcell.ColorWhite)
	search_box.SetTitleAlign(tview.AlignLeft)
	search_box.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := search_box.GetText()
			if text != "" {
				app.performSearch(text, 5)
				app.app.SetFocus(app.music_list) // Focus directly on the table for navigation
			}
		}
	})

	// Set up header box
	header_box.AddItem(search_box, 0, 1, false)
	// header_box.AddItem(status_box, 0, 1, false)

	// Menu
	menu := tview.NewList()
	menu.AddItem("Settings", "", 's', nil)
	menu.AddItem("Exit", "", 'q', func() {
		if app.timer != nil {
			app.timer.Stop()
		}
		services.Cleanup()
		app.app.Stop()
	})
	menu.SetBorder(true).SetTitle("Menu")
	menu.SetTitleAlign(tview.AlignLeft)

	// Setup layout
	main_box.AddItem(header_box, 0, 1, true)
	main_box.AddItem(flex_box, 0, 6, false)
	main_box.AddItem(player_box, 0, 1, false)

	flex_box.AddItem(menu, 0, 1, false)
	flex_box.AddItem(content_box, 0, 5, false)

	if err := app.app.
		SetRoot(main_box, true).
		EnableMouse(true).
		Run(); err != nil {
		panic(err)
	}
}
