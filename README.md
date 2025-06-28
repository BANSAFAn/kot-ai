# KOT.AI - Personal Voice Assistant

KOT.AI is a personal voice assistant for Windows, written in Go. It allows you to control your computer using voice commands and interact through a web interface.

## Features

- 🎤 Voice control with wake word activation
- 💻 Launch applications and open websites
- 🔊 Control system volume
- 📊 Get system information
- 📝 Process management
- 🌐 Web interface for interaction
- 🧠 OpenAI API integration for command processing
- 🗣️ Speech synthesis for responses

## Installation

### Download the executable file

1. Go to the [Releases](https://github.com/BANSAFAn/kot.ai/releases) page
2. Download the latest version of `kot.exe`
3. Run the file

### Build from source

```bash
# Clone the repository
git clone https://github.com/BANSAFAn/kot.ai.git
cd kot.ai

# Install dependencies
go mod download

# Build the project
go build -o kot.exe -ldflags="-H=windowsgui" .
```

### Quick Start

Run the `run.bat` script to automatically build and start the application:  
```bash
run.bat
```

## Configuration

Configuration is done through the `config.json` file, which is created on first launch in the `%USERPROFILE%\.kot.ai\` directory.

```json
{
  "assistant": {
    "name": "Kot",
    "openai_api_key": "your-api-key",
    "google_api_key": "your-api-key",
    "use_local_models": false,
    "local_model_path": "",
    "history_enabled": true,
    "history_file_path": "C:\Users\your_name\.kot.ai\history.db"
  },
  "voice": {
    "enabled": true,
    "wake_word": "kot",
    "language": "en-US",
    "voice_recognition": "google",
    "tts_provider": "google",
    "voice_threshold": 0.5,
    "silence_threshold": 0.1,
    "input_device": "",
    "output_device": ""
  },
  "ui": {
    "enabled": true,
    "ui_type": "web",
    "web_port": 8080,
    "theme": "dark",
    "start_minimized": false
  }
}
```

### Configuration Parameters

#### Assistant
- `name` - assistant name
- `openai_api_key` - OpenAI API key for command processing
- `google_api_key` - Google API key for services
- `use_local_models` - whether to use local models
- `local_model_path` - path to local models
- `history_enabled` - enable conversation history
- `history_file_path` - path to history database

#### Voice
- `enabled` - enable voice control
- `wake_word` - activation word (default "kot")
- `language` - language for speech recognition (e.g., "en-US")
- `voice_recognition` - speech recognition provider ("google" or "whisper")
- `tts_provider` - text-to-speech provider ("google" or "local")
- `voice_threshold` - voice detection threshold
- `silence_threshold` - silence detection threshold
- `input_device` - specific input device
- `output_device` - specific output device

#### UI
- `enabled` - enable user interface
- `ui_type` - interface type ("web")
- `web_port` - port for web interface
- `theme` - interface theme ("dark" or "light")
- `start_minimized` - start application minimized

## Usage

### Voice Commands

1. Activate the assistant by saying the wake word (default "kot")
2. After the sound signal, speak your command

Example commands:
- "Open browser"
- "Launch calculator"
- "Open website google.com"
- "Take a screenshot"
- "System information"
- "List processes"
- "Increase volume"
- "Decrease volume"

### Web Interface

When KOT.AI starts, it automatically launches a web server on port 8080. You can open the web interface at http://localhost:8080/

## Development

### Project Structure

```
.
├── .github/workflows/   # GitHub Actions for automatic build
├── internal/            # Internal packages
│   ├── assistant/       # Main assistant logic
│   ├── config/          # Configuration management
│   ├── system/          # System interaction
│   ├── ui/              # User interface
│   └── voice/           # Voice control
├── main.go              # Entry point
├── go.mod               # Dependencies
└── README.md            # Documentation
```

### Development Requirements

- Go 1.21 or higher
- Windows 10/11
- Microphone for voice control

## License

MIT