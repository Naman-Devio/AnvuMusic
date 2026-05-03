<h1 align="center">🎧 <b>AnvuMusic</b></h1>

<p align="center">
  <i>A blazing-fast, reliable, and feature-packed Telegram bot for streaming music in group voice chats — built with Go.</i>
</p>
<p align="center">

  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&labelColor=000000&logoColor=white" alt="Go Version">
  </a>
  <a href="https://github.com/Naman-Devio/AnvuMusic/releases">
    <img src="https://img.shields.io/github/v/release/Naman-Devio/AnvuMusic?label=Release&style=for-the-badge&labelColor=000000&color=FF9800&logo=semver&logoColor=white" alt="Release">
  </a>
  <a href="https://github.com/Naman-Devio/AnvuMusic/releases/tag/2.0.0">
    <img src="https://img.shields.io/badge/Latest-v2.0.0-FF9800?style=for-the-badge&logo=github&labelColor=000000&logoColor=white" alt="Latest Release">
  </a>
  <a href="../LICENSE">
    <img src="https://img.shields.io/badge/License-GPLv3-FF3860?style=for-the-badge&logo=gnu&labelColor=000000&logoColor=white" alt="License: GPLv3">
  </a>
  <a href="https://github.com/Naman-Devio/AnvuMusic/stargazers">
    <img src="https://img.shields.io/github/stars/Naman-Devio/AnvuMusic?style=for-the-badge&label=Stars&labelColor=000000&color=FFD700&logo=github&logoColor=white" alt="GitHub Stars">
  </a>
  <a href="https://github.com/Naman-Devio/AnvuMusic/fork">
    <img src="https://img.shields.io/github/forks/Naman-Devio/AnvuMusic?style=for-the-badge&label=Forks&labelColor=000000&color=00C853&logo=github&logoColor=white" alt="GitHub Forks">
  </a>

</p>

---

## ⭐ Star History

<a href="https://star-history.com/#Naman-Devio/AnvuMusic&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=Naman-Devio/AnvuMusic&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=Naman-Devio/AnvuMusic&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=Naman-Devio/AnvuMusic&type=Date" />
 </picture>
</a>

---

## 🚀 Quick Start


### ☁️ Deploy to Heroku (One-Click)

Click the button below to deploy **AnvuMusic** instantly on Heroku:

<a href="https://heroku.com/deploy?template=https://github.com/Naman-Devio/AnvuMusic">
  <img src="https://www.herokucdn.com/deploy/button.svg" alt="Deploy to Heroku">
</a>

---

### Prerequisites
- Go 1.25 or higher
- MongoDB (Cloud or Local)
- Telegram Bot Token (from [@BotFather](https://t.me/BotFather))
- API ID & Hash (from [my.telegram.org](https://my.telegram.org))
- Assistant Account Session String

### Installation

1. **Clone Repository**
```bash
git clone https://github.com/Naman-Devio/AnvuMusic.git
cd AnvuMusic
```

2. **Install Dependencies**
```bash
bash install.sh && go mod tidy
```

3. **Configure Environment**
```bash
cp sample.env .env
# Edit .env with your credentials
```

4. **Get Required Credentials**
- **Bot Token**: Message [@BotFather](https://t.me/BotFather), use `/newbot`
- **API ID & Hash**: Visit [my.telegram.org](https://my.telegram.org)
- **Session String**: Use [@StringFatherBot](https://t.me/StringFatherBot) or online generator
- **MongoDB**: Free tier at [MongoDB Atlas](https://www.mongodb.com/cloud/atlas)

5. **Start the Bot**
```bash
go run ./cmd/app
```

---

## ⚙️ Configuration

All configuration is managed through environment variables. For detailed configuration instructions, see:

📖 **[Configuration Guide](../internal/config/README.md)**

### Key Variables
| Variable | Required | Purpose |
|----------|----------|---------|
| `API_ID` | ✅ | Telegram API ID |
| `API_HASH` | ✅ | Telegram API Hash |
| `TOKEN` | ✅ | Bot token from @BotFather |
| `MONGO_DB_URI` | ✅ | MongoDB connection string |
| `STRING_SESSIONS` | ✅ | Assistant account session strings |
| `OWNER_ID` | ❌ | Your Telegram User ID |
| `LOGGER_ID` | ❌ | Log channel ID |

See **[Configuration Reference](../internal/config/README.md)** for complete list of variables with examples.

---

## 📚 Commands

### User Commands
| Command | Description |
|---------|-------------|
| `/play <query>` | Play a song from YouTube or Telegram |
| `/queue` | Show current queue |
| `/position` | Show current track position |
| `/help` | Show command help |
| `/ping` | Check bot status |

### Admin Commands
| Command | Description |
|---------|-------------|
| `/pause [seconds]` | Pause for playback (optionally auto-resume) |
| `/resume` | Resume paused track |
| `/mute [seconds]` | Mute playback (optionally auto-unmute) |
| `/unmute` | Unmute playback |
| `/seek <seconds>` | Seek to specific position |
| `/loop <count>` | Loop track N times |
| `/shuffle` | Toggle shuffle mode |
| `/speed <speed>` | Set playback speed (0.5-4.0x) |
| `/skip` | Skip to next track |
| `/fplay <query>` | Force play (skip queue) |
| `/clear` | Clear entire queue |
| `/remove <index>` | Remove track from queue |
| `/move <from> <to>` | Move track in queue |
| `/jump <position>` | Jump to position in current track |
| `/replay` | Replay current track |
| `/addauth <user>` | Grant user playback permission |
| `/delauth <user>` | Revoke user playback permission |
| `/authlist` | List authorized users |
| `/reload` | Reload admin cache |
| `/cplay` | Channel play mode |

### Owner Commands
| Command | Description |
|---------|-------------|
| `/addsudo <user>` | Add sudo user |
| `/delsudo <user>` | Remove sudo user |
| `/sudolist` | List all sudo users |
| `/maintenance <on/off>` | Toggle maintenance mode |
| `/broadcast <message>` | Broadcast to all chats |
| `/stats` | Show bot statistics |
| `/restart` | Restart bot |

---

## 🎼 Platform System

AnvuMusic uses a **modular platform system** to support multiple music sources:

### Supported Platforms
1. **Telegram** (Priority: 100) — in-chat audio/video
2. **Spotify** (Priority: 95) — metadata + YouTube fallback for playback
3. **YouTube** (Priority: 90) — search and metadata
4. **SoundCloud / Shruti API** (Priority: 85) — native catalogue hooks where configured
5. **YT-DLP** (Priority: 60) — universal download fallback
6. **DirectStream** (Priority: 10) — direct URLs (MP3, HLS/M3U8, etc.)

### How It Works
- When you request a song, the bot checks each platform by **priority**
- **First valid platform handles the request**
- Automatic fallback if one method fails
- Seamless track fetching and downloading

📖 **[Platform System Guide](../internal/platforms/README.md)** - Learn how to add custom platforms

---

## 📖 Documentation

- **[Configuration Guide](../internal/config/README.md)** - All environment variables explained
- **[Platform System](../internal/platforms/README.md)** - How platforms work and add custom sources
- **[Database Layer](../internal/database/README.md)** - MongoDB schemas, queries, and bot data management
- **[Modules System](../internal/modules/README.md)** - Command handlers, permissions, and feature modules
---

## 🏗️ Project Structure

```
AnvuMusic/
├── .github/
│   └── README.md              # Main documentation (you are here)
├── cmd/app/
│   └── main.go                # Entry point
│   └──...
├── internal/
│   ├── config/                # Configuration management
│   │   ├── config.go
│   │   └── README.md          # Detailed config guide
│   ├── core/                  # Core bot logic
│   │   ├── clients.go         # Bot & assistant initialization
│   │   ├── room_state.go      # Playback state management
│   │   ├── chat_state.go      # Chat & assistant state
│   │   └── ...
│   ├── database/              # MongoDB operations
│   │   ├── bot_state.go
│   │   ├── chat_settings.go
│   │   └── ...
│   ├── modules/               # Command handlers
│   │   ├── play.go            # Play command
│   │   ├── queue.go           # Queue management
│   │   └── ...
│   ├── platforms/             # Music sources
│   │   ├── youtube.go         # YouTube integration
│   │   ├── telegram.go        # Telegram media
│   │   ├── ytdlp.go           # YT-DLP downloader
│   │   └── ...
│   ├── locales/               # Multi-language support
│   │   ├── en.yml            # English
│   │   └── ...
│   ├── utils/                 # Utility functions
│   ├── cookies/               # YouTube cookie files
│   └── ...
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
```

---

## 🐛 Bug Reports & Features

Found a bug? Have a feature request?

- **Use `/bug` command** in the bot to report directly
- **Join our [Support Chat](https://t.me/+t4FSZuy7t_VmOWE9)** for discussions
- **Open an [Issue on GitHub](https://github.com/Naman-Devio/AnvuMusic/issues)**

---

## 🤝 Contributing

Contributions are welcome! Here's how to help:

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Commit changes** (`git commit -m 'Add amazing feature'`)
4. **Push to branch** (`git push origin feature/amazing-feature`)
5. **Open a Pull Request**

### Adding a New Platform
See **[Platform System Guide](../internal/platforms/README.md#-how-to-add-a-new-platform)** for step-by-step instructions.

---

## 📜 License

This project is licensed under the **GNU General Public License v3.0** - see the [LICENSE](LICENSE) file for details.

```
AnvuMusic — A Telegram bot that streams music into group voice chats
Copyright (C) 2026 Team Echo

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
```

---

## 🙌 Credits

- **EchoWave · Anvu** — current bot line and configuration defaults
- **Contributors**: everyone who sends fixes and features
- **Libraries**: [gogram](https://github.com/AmarnathCJD/gogram), [ntgcalls](https://github.com/pytgcalls/ntgcalls), and the rest of the `go.mod` graph

---

## 📞 Support

- **Telegram Support**: [support chat](https://t.me/+t4FSZuy7t_VmOWE9)
- **Updates channel**: [EchoWave](https://t.me/ECHOWAVESUPPORT)
- **GitHub Issues**: [Report bugs](https://github.com/Naman-Devio/AnvuMusic/issues)

---

## ⚡ Performance Tips

1. **Use multiple assistant accounts** - Distributes load across accounts
2. **Set appropriate limits** - Adjust `QUEUE_LIMIT` and `DURATION_LIMIT`
3. **Enable auto-leave** - Removes bot from inactive chats automatically
4. **Use MongoDB Atlas** - Better performance than local MongoDB
5. **Set up logger** - Monitor errors and optimize accordingly

---

**Happy Streaming! 🎶**
