# ytview

A terminal-based YouTube music player written in Go, featuring a TUI (Text User Interface) powered by tview.

## Features

- 🎵 Search and play YouTube music videos
- 🎨 Terminal user interface
- ⌨️ Keyboard-driven controls
- ⏯️ Play/Pause functionality
- 🕒 Real-time duration and progress display
- 📺 Channel information display

## Prerequisites

- Go 1.24.2 or higher
- yt-dlp (included in tools/yt-dlp.exe)
- A working internet connection

## Installation

```bash
# Clone the repository
git clone https://github.com/sangnt1552314/ytview.git

# Change to the project directory
cd ytview

# Install dependencies
go mod download

# Build the project
go build -o ytview ./cmd/main.go
```

## Usage

1. Start the application:
```bash
./ytview
```

2. Use the following keyboard shortcuts:
   - Type to search for music
   - Arrow keys to navigate
   - Enter to play selected track
   - Space to play/pause
   - Ctrl+C to quit

## Dependencies

- [github.com/gdamore/tcell/v2](https://github.com/gdamore/tcell) - Terminal handling
- [github.com/rivo/tview](https://github.com/rivo/tview) - Terminal UI library
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - YouTube video downloading and processing
- [github.com/joho/godotenv](https://github.com/joho/godotenv) - Environment variable management
- [github.com/kkdai/youtube](https://github.com/kkdai/youtube) - YouTube video data extraction

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

[sangnt1552314](https://github.com/sangnt1552314)