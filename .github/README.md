<div align="center">

<img src="https://i.ibb.co/t6z06g1/photo-2026-07-06-16-43-28-7659456606119458816.jpg" width="560" style="max-width: 100%; height: auto; border-radius: 12px; box-shadow: 0 6px 24px rgba(0,0,0,0.15);" alt="AnvuMusic Banner"/>

# AnvuMusic

*A high-performance Telegram music bot powered by Go, built for seamless voice chat streaming.*

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-GPLv3-FF6B9D?style=flat-square&logo=gnu&logoColor=white)](https://github.com/Naman-Devio/AnvuMusic/blob/main/LICENSE)
[![Support Channel](https://img.shields.io/badge/Channel-@ECHOWAVESUPPORT-blue?style=flat-square&logo=telegram&logoColor=white)](https://t.me/ECHOWAVESUPPORT)
[![Developer](https://img.shields.io/badge/Dev-@eceqt-FF9EC4?style=flat-square&logo=telegram&logoColor=white)](https://t.me/eceqt)

[Features](#-features) • [Platforms](#-platform-system) • [Commands](#-commands) • [Configuration](#-configuration) • [Deployment](#-quick-deploy) • [Credits](#-credits--acknowledgements)

</div>

---

## ✨ Features

- **Multi-Source Streaming** — YouTube, Spotify, SoundCloud, Telegram files, direct audio streams, and HLS/M3U8 links.
- **Fast Stream Resolvers** — Direct high-speed streaming via Yuki API & Shruti API with automatic failover to yt-dlp.
- **Playback Controls** — Live speed control (0.5× to 4.0×), seeking, looping, pause/resume, mute/unmute, and force play.
- **Queue Management** — Smart queue with skip, remove, move, jump, shuffle, and clear actions.
- **Access Control** — Chat-level authorized users (up to 25) and global sudo/owner administration.
- **Performance & Stats** — Real-time latency checks (`/ping`), system memory/CPU monitoring, speedtest, and broadcast tools.

---

## 🎼 Platform System

Tracks are resolved through a priority-based fallback chain:

| Priority | Platform | Type | Notes |
|:---:|---|---|---|
| 🥇 **100** | **Telegram** | Native | Direct audio & video files from Telegram messages |
| 🥈 **95** | **Spotify** | Metadata | Resolves tracks, albums & playlists → streams via YouTube |
| 🥉 **90** | **YouTube** | Search & Metadata | Global query search, playlist resolution & video playback |
| ✦ **85** | **Yuki / Shruti API** | Downloader | High-speed direct streaming endpoints for YouTube tracks |
| ✦ **85** | **SoundCloud** | Native | Music playback from SoundCloud |
| ✦ **65** | **DirectStream** | Fallback | Direct `.mp3`, `.mp4`, HLS, and M3U8 web streams |
| ✦ **60** | **YT-DLP** | Universal | Universal safety net supporting 1000+ websites |

---

## 📜 Commands

### Public Commands

| Command | Description |
|---|---|
| `/play <query \| url>` | Play a song from YouTube, Spotify, or supported URLs |
| `/vplay <query \| url>` | Play a video track in the voice chat |
| `/queue` | View the current track queue |
| `/position` | Check current playback position |
| `/ping` | Check bot latency, system uptime, CPU & RAM |
| `/help` | Show command menu and usage instructions |

### Admin Commands

| Command | Description |
|---|---|
| `/fplay <query>` | Force play a track immediately (skips current song) |
| `/pause` / `/resume` | Pause or resume playback |
| `/mute` / `/unmute` | Mute or unmute the voice chat stream |
| `/seek <seconds>` | Seek forward to a specific timestamp |
| `/speed <0.5–4.0>` | Adjust live playback speed |
| `/loop <count>` | Loop the current track N times |
| `/shuffle` | Toggle queue shuffle |
| `/skip` | Skip to the next track in queue |
| `/stop` | Stop playback and clear the queue |
| `/clear` | Clear upcoming tracks from the queue |
| `/remove <index>` | Remove a specific track from the queue |
| `/move <from> <to>` | Reorder a track in the queue |
| `/replay` | Replay the current track from the beginning |
| `/addauth <user>` | Grant a user play permissions in the group |
| `/delauth <user>` | Revoke a user's play permissions |
| `/authlist` | List authorized users for the chat |
| `/cplay` | Stream music into a linked channel |
| `/reload` | Refresh the admin cache |

### Owner & Sudo Commands

| Command | Description |
|---|---|
| `/addsudo` / `/delsudo` | Add or remove global sudo users |
| `/sudolist` | View all sudo users |
| `/maintenance <on/off>` | Toggle maintenance mode |
| `/broadcast <msg>` | Send a broadcast message to all chats |
| `/stats` | View runtime, memory, and database stats |
| `/speedtest` | Run an internet bandwidth speedtest |
| `/restart` | Restart the bot process |

---

## ⚙️ Configuration

Set these variables in your `.env` file or environment:

### Required Variables

| Variable | Description |
|---|---|
| `API_ID` | Telegram API ID from [my.telegram.org](https://my.telegram.org) |
| `API_HASH` | Telegram API Hash from [my.telegram.org](https://my.telegram.org) |
| `TOKEN` | Bot token from [@BotFather](https://t.me/BotFather) |
| `MONGO_DB_URI` | MongoDB connection URI |
| `STRING_SESSIONS` | Assistant Pyrogram/Telethon string session(s) |
| `OWNER_ID` | Telegram user ID of the bot owner |

### Optional Variables & External APIs

| Variable | Default | Description |
|---|---|---|
| `LOGGER_ID` | `0` | Log channel / group ID for bot logs |
| `YUKI_API_KEY` | `""` | Yuki API Key for fast stream downloads (via [@MeowApiRobot](https://t.me/MeowApiRobot)) |
| `SHRUTI_API_KEY` | `""` | Shruti API Key for streaming (via [@SHRUTIAPIBOT](https://t.me/SHRUTIAPIBOT)) |
| `SPOTIFY_CLIENT_ID` | `""` | Spotify app client ID for Spotify link resolution |
| `SPOTIFY_CLIENT_SECRET` | `""` | Spotify app client secret |
| `DEFAULT_LANG` | `en` | Default bot language |
| `DURATION_LIMIT` | `3600` | Maximum track duration in seconds |
| `QUEUE_LIMIT` | `10` | Maximum songs per chat queue |
| `MAX_AUTH_USERS` | `25` | Maximum authorized users per group |
| `SUPPORT_CHAT` | — | Support group invite link |
| `SUPPORT_CHANNEL` | `https://t.me/ECHOWAVESUPPORT` | Official channel link |
| `START_IMG_URL` | — | Custom image URL for `/start` |
| `PING_IMG_URL` | — | Custom image URL for `/ping` |

---

## 🚀 Quick Deploy

### One-Click Heroku Deploy

[![Deploy to Heroku](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy?template=https://github.com/Naman-Devio/AnvuMusic)

---

### Docker Setup

```bash
# 1. Clone repository
git clone https://github.com/Naman-Devio/AnvuMusic.git
cd AnvuMusic

# 2. Setup environment
cp sample.env .env
nano .env

# 3. Build and launch
docker build -t anvumusic .
docker run --env-file .env anvumusic
```

---

### Manual Setup (VPS / Local)

**Prerequisites:** Go 1.25+, FFmpeg, yt-dlp, MongoDB

```bash
# 1. Clone repository
git clone https://github.com/Naman-Devio/AnvuMusic.git
cd AnvuMusic

# 2. Install dependencies
bash install.sh

# 3. Configure environment
cp sample.env .env
nano .env

# 4. Run the bot
go run ./cmd/app
```

---

## 💖 Credits & Acknowledgements

Special thanks to the open-source projects and developers that make AnvuMusic possible:

| Component | Resource / Author |
|---|---|
| 🏛️ **Base Repository** | [Ashok Sahu (TgMusicBot)](https://github.com/AshokShau/TgMusicBot) |
| ⚡ **Yuki Stream API** | [@Z0iiw](https://t.me/Z0iiw) via [@MeowApiRobot](https://t.me/MeowApiRobot) |
| 🎵 **Shruti Stream API** | [@Yaduwanshi_Nand](https://t.me/Yaduwanshi_Nand) via [@SHRUTIAPIBOT](https://t.me/SHRUTIAPIBOT) |
| 🚀 **MTProto Engine** | [gogram](https://github.com/amarnathcjd/gogram) by AmarnathCJD |
| 🎙️ **Voice & WebRTC** | [ntgcalls](https://github.com/pytgcalls/ntgcalls) by pytgcalls team |
| 🧑‍💻 **Developer** | [@eceqt](https://t.me/eceqt) |
| 📢 **Support & Updates** | [@ECHOWAVESUPPORT](https://t.me/ECHOWAVESUPPORT) |

---

## 📜 License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
