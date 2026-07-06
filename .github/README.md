<div align="center">

<!-- ✦ REPLACE THIS with your own anime PNG hosted on catbox.moe or imgur ✦ -->
<!-- Tip: use a transparent PNG of a cute anime girl with headphones! -->
<!-- Example tool to host your PNG free → https://catbox.moe -->

<img src="https://i.ibb.co/0yph13mt/photo-2026-06-21-18-22-19-7653915802706780160.jpg" width="100%" style="border-radius:18px;" alt="AnvuMusic Banner"/>

<br/>

# 𝓐𝓷𝓿𝓾𝓜𝓾𝓼𝓲𝓬

<p>
  <i>✦ A blazing-fast, anime-spirited Telegram music bot — powered by Go & pure love for music ✦</i>
</p>

<p>
  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&labelColor=0D0D1A&logoColor=white" alt="Go Version"/>
  </a>
  <a href="https://github.com/Naman-Devio/AnvuMusic/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-GPLv3-FF6B9D?style=for-the-badge&logo=gnu&labelColor=0D0D1A&logoColor=white" alt="License"/>
  </a>
  <a href="https://t.me/ECHOWAVESUPPORT">
    <img src="https://img.shields.io/badge/Channel-@ECHOWAVESUPPORT-C3B1E1?style=for-the-badge&logo=telegram&labelColor=0D0D1A&logoColor=white" alt="Support Channel"/>
  </a>
  <a href="https://t.me/eceqt">
    <img src="https://img.shields.io/badge/Dev-@eceqt-FF9EC4?style=for-the-badge&logo=telegram&labelColor=0D0D1A&logoColor=white" alt="Developer"/>
  </a>
</p>

<p>
  <img src="https://img.shields.io/badge/MongoDB-Database-47A248?style=for-the-badge&logo=mongodb&labelColor=0D0D1A" alt="MongoDB"/>
  <img src="https://img.shields.io/badge/FFmpeg-Audio_Engine-007808?style=for-the-badge&logo=ffmpeg&labelColor=0D0D1A" alt="FFmpeg"/>
  <img src="https://img.shields.io/badge/ntgcalls-WebRTC-FF6B9D?style=for-the-badge&labelColor=0D0D1A" alt="ntgcalls"/>
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&labelColor=0D0D1A" alt="Docker"/>
</p>

<p>
  <a href="#-quick-deploy">🚀 Deploy Now</a> ·
  <a href="#-features">✨ Features</a> ·
  <a href="#-commands">📜 Commands</a> ·
  <a href="#-configuration">⚙️ Config</a> ·
  <a href="https://t.me/ECHOWAVESUPPORT">💬 Support</a>
</p>

</div>

---

<div align="center">
  <img src="https://readme-typing-svg.demolab.com?font=Nunito&weight=700&size=22&pause=1000&color=FF6B9D&center=true&vCenter=true&width=600&lines=🎵+Stream+music+in+Telegram+voice+chats;⚡+Built+with+Go+for+ultra-low+latency;🌸+Spotify+%7C+YouTube+%7C+SoundCloud+%26+more;🎧+Smart+queue+%7C+Auth+system+%7C+RTMP;💖+Made+with+love+by+Team+Echo" alt="Typing SVG"/>
</div>

---

## ✨ Features

<table>
<tr>
<td width="50%">

### 🎵 Music Playback
- **Multi-platform streaming** — YouTube, Spotify, SoundCloud, Telegram files, direct URLs & HLS streams
- **Priority-based fallback system** — Arc API → yt-dlp → always finds a way to play
- **Speed control** — 0.5× to 4.0× playback speed, live
- **Seek & position** — jump to any timestamp mid-track
- **Loop & shuffle** — loop N times or shuffle the whole queue
- **Force play** — instantly cut the queue and play now

</td>
<td width="50%">

### 📋 Queue & Controls
- **Smart queue management** — view, skip, remove, move, jump to any position
- **Pause / Resume / Mute / Unmute** with optional auto-timeout timers
- **Replay** current track from the beginning
- **Channel play** (cplay) — stream into linked channels
- **RTMP live stream** support
- **Auto-leave** on inactivity to save resources

</td>
</tr>
<tr>
<td width="50%">

### 🔒 Admin & Auth System
- **Per-chat authorized users** (up to 25) — grant non-admins play access
- **Sudo users** — elevated global privileges
- **Owner commands** — broadcast, maintenance mode, restart
- **Admin cache reload** — instant refresh after promotions
- **Leave-on-demote** — bot auto-exits if it loses admin

</td>
<td width="50%">

### 📊 Monitoring & Tools
- **Live ping** — latency (ms), uptime, RAM, CPU & disk usage
- **Detailed stats** — Go runtime, GC, served chats/users
- **Built-in speed test** — internet bandwidth check
- **Broadcast** — send messages to all served chats/users
- **Multi-language** — locale-based i18n system (YAML)
- **Must-join** enforcement & thumbnail customization

</td>
</tr>
</table>

---

## 🎼 Platform System

> Tracks are resolved via a **priority-based fallback chain** — highest priority wins, lower ones catch failures.

| Priority | Platform | Type | Notes |
|:---:|---|---|---|
| 🥇 **100** | **Telegram** | Native | Direct audio/video files from Telegram |
| 🥈 **95** | **Spotify** | Metadata | Resolves tracks, albums & playlists → downloads via YouTube |
| 🥉 **90** | **YouTube** | Search + Meta | Global search, playlists, high-accuracy results |
| ✦ **85** | **SoundCloud** | Native | Independent & indie music via yt-dlp |
| ✦ **80** | **Arc API** | Downloader | Premium CDN-backed YouTube audio downloads |
| ✦ **65** | **DirectStream** | Fallback | HLS/M3U8, MPEG-DASH, CDN `.mp3`/`.mp4` links |
| ✦ **60** | **YT-DLP** | Universal | 1000+ sites — the final safety net |

---

## 📜 Commands

### 🌍 Public Commands

| Command | Description |
|---|---|
| `/play <query \| url>` | 🎵 Play a song from YouTube, Spotify, or other platforms |
| `/queue` | 📋 View the current queue |
| `/position` | 📍 Show playback position in the current track |
| `/ping` | 🏓 Check bot latency, uptime, CPU & RAM |
| `/help` | 📖 Show help menu |

### 🛠️ Admin Commands

| Command | Description |
|---|---|
| `/fplay <query>` | ⚡ Force play — skip queue immediately |
| `/pause [seconds]` | ⏸️ Pause playback (optional auto-resume timer) |
| `/resume` | ▶️ Resume paused playback |
| `/mute [seconds]` | 🔇 Mute (optional auto-unmute timer) |
| `/unmute` | 🔊 Unmute playback |
| `/seek <seconds>` | ⏩ Seek to a specific timestamp |
| `/speed <0.5–4.0>` | 🐇 Set playback speed |
| `/loop <count>` | 🔁 Loop current track N times |
| `/shuffle` | 🔀 Toggle shuffle mode |
| `/skip` | ⏭️ Skip to next track in queue |
| `/stop` | ⏹️ Stop playback and clear queue |
| `/clear` | 🗑️ Clear the entire queue |
| `/remove <index>` | ❌ Remove a track from the queue |
| `/move <from> <to>` | ↕️ Move a track to a new queue position |
| `/jump <position>` | 🎯 Jump to a position in the current track |
| `/replay` | 🔄 Replay the current track from beginning |
| `/addauth <user>` | ✅ Grant a user play permission |
| `/delauth <user>` | ❎ Revoke a user's play permission |
| `/authlist` | 📃 List all authorized users in this chat |
| `/reload` | 🔃 Reload admin cache |
| `/cplay` | 📡 Channel play mode |
| `/bug` | 🐛 Report a bug directly to the dev |

### 👑 Owner / Sudo Commands

| Command | Description |
|---|---|
| `/addsudo <user>` | ⚡ Add a sudo user |
| `/delsudo <user>` | 🚫 Remove a sudo user |
| `/sudolist` | 📃 List all sudo users |
| `/maintenance <on/off>` | 🔧 Toggle maintenance mode |
| `/broadcast <msg>` | 📣 Broadcast a message to all served chats |
| `/stats` | 📊 Full system & bot statistics |
| `/speedtest` | 🌐 Run a server speed test |
| `/restart` | 🔁 Restart the bot |
| `/eval <code>` | 🧑‍💻 Evaluate Go code (dev only) |

---

## ⚙️ Configuration

All configuration is done via environment variables (`.env` file or host env).

### 🔴 Required

| Variable | Description |
|---|---|
| `API_ID` | Telegram API ID from [my.telegram.org](https://my.telegram.org) |
| `API_HASH` | Telegram API Hash from [my.telegram.org](https://my.telegram.org) |
| `TOKEN` | Bot token from [@BotFather](https://t.me/BotFather) |
| `MONGO_DB_URI` | MongoDB connection URI |
| `STRING_SESSIONS` | One or more assistant session strings (space/comma separated) |
| `LOGGER_ID` | Chat ID where bot logs are sent |

### 🟡 Optional

| Variable | Default | Description |
|---|---|---|
| `OWNER_ID` | `0` | Your Telegram user ID |
| `SPOTIFY_CLIENT_ID` | — | Spotify app client ID |
| `SPOTIFY_CLIENT_SECRET` | — | Spotify app client secret |
| `DEFAULT_LANG` | `en` | Default bot language |
| `DURATION_LIMIT` | `3600` | Max track duration in seconds |
| `QUEUE_LIMIT` | `10` | Max tracks per queue |
| `MAX_AUTH_USERS` | `25` | Max authorized users per chat |
| `LEAVE_ON_DEMOTED` | `false` | Leave chat if bot is demoted |
| `SUPPORT_CHAT` | — | Your support group URL |
| `SUPPORT_CHANNEL` | `https://t.me/ECHOWAVESUPPORT` | Your channel URL |
| `COOKIES_LINK` | — | Space-separated batbin.me URLs for YT cookies |
| `START_IMG_URL` | catbox image | Start message image URL |
| `PING_IMG_URL` | catbox image | Ping message image URL |
| `MUST_JOIN` | `ECHOWAVESUPPORT` | Required channel to use the bot |
| `SET_CMDS` | `true` | Auto-set bot commands on startup |
| `PORT` | `8000` | HTTP port (for health checks) |

> 💡 **Tip:** `STRING_SESSIONS` supports multiple sessions separated by spaces or commas — the bot load-balances across them automatically!

---

## 🚀 Quick Deploy

### ☁️ One-Click Heroku Deploy

Click below to deploy **AnvuMusic** instantly on Heroku (no server needed!):

<a href="https://heroku.com/deploy?template=https://github.com/Naman-Devio/AnvuMusic">
  <img src="https://www.herokucdn.com/deploy/button.svg" alt="Deploy to Heroku" height="40"/>
</a>

---

### 🐳 Docker

```bash
# Clone the repo
git clone https://github.com/Naman-Devio/AnvuMusic.git
cd AnvuMusic

# Copy and fill in your environment variables
cp sample.env .env
nano .env

# Build and run
docker build -t anvumusic .
docker run --env-file .env anvumusic
```

---

### 🖥️ Manual (VPS / Local)

**Prerequisites:** Go 1.25+, FFmpeg, yt-dlp, MongoDB, ntgcalls

```bash
# 1. Clone the repository
git clone https://github.com/Naman-Devio/AnvuMusic.git
cd AnvuMusic

# 2. Run the automated installer (handles Go, FFmpeg, yt-dlp, ntgcalls)
bash install.sh

# 3. Copy and edit your environment file
cp sample.env .env
nano .env   # fill in your credentials

# 4. Download Go dependencies
go mod tidy

# 5. Start the bot 🎶
go run ./cmd/app
```

> 🎓 **Get your credentials:**
> - **Bot Token** → [@BotFather](https://t.me/BotFather) → `/newbot`
> - **API ID & Hash** → [my.telegram.org](https://my.telegram.org/apps)
> - **Session String** → [STRING](https://t.me/dragonstringbot)
> - **MongoDB URI** → [MongoDB Atlas](https://www.mongodb.com/cloud/atlas) (free tier)
> - **Spotify Keys** → [developer.spotify.com](https://developer.spotify.com/dashboard)

---

## 🏗️ Project Structure

```
AnvuMusic/
│
├── 📁 cmd/app/
│   └── main.go                  ← Entry point & boot sequence
│
├── 📁 internal/
│   ├── 📁 config/               ← All env variables & validation
│   ├── 📁 core/                 ← Bot clients, room state, queue engine
│   │   ├── clients.go           ← Bot & assistant initialization
│   │   ├── room_state.go        ← Playback state per chat
│   │   ├── room_queue.go        ← Queue management engine
│   │   └── buttons.go           ← Inline keyboard builders
│   ├── 📁 modules/              ← Command handlers (one file per command)
│   │   ├── play.go · queue.go · skip.go · pause.go · seek.go
│   │   ├── auth.go · sudoers.go · broadcast.go · stats.go
│   │   └── rtmp_stream.go · monitor.go · speedtest.go · eval.go
│   ├── 📁 platforms/            ← Music source integrations
│   │   ├── youtube.go · spotify.go · soundcloud.go
│   │   ├── arcapi.go · ytdlp.go · directstream.go · telegram.go
│   │   └── registry.go          ← Priority-based platform router
│   ├── 📁 database/             ← MongoDB collections & queries
│   ├── 📁 locales/              ← i18n YAML files (en.yml, ...)
│   ├── 📁 utils/                ← Shared helpers & flood control
│   └── 📁 cookies/              ← YouTube cookie rotation
│
├── 📁 ntgcalls/                 ← Go bindings for ntgcalls (WebRTC engine)
├── 📁 ubot/                     ← Userbot/assistant voice call logic
│
├── Dockerfile · heroku.yml · Procfile · install.sh
└── go.mod · go.sum
```

---

## 🎨 Anime Corner

> *"Even the loneliest night is less lonely with music playing."*
> — AnvuMusic, probably 🌙

<!-- 
  ✦ HOW TO ADD YOUR ANIME PNG ✦
  
  1. Find a cute anime girl with headphones (transparent PNG preferred!)
     → Great free sources: https://www.pixiv.net  |  https://safebooru.org
     → Or generate one with any AI image tool
  
  2. Upload it to https://catbox.moe (free, permanent hosting)
  
  3. Replace the img src at the very TOP of this README with your catbox.moe URL
  
  4. For a side-mascot effect, you can wrap sections in an HTML table like this:
  
  <table><tr>
    <td><img src="YOUR_ANIME_PNG_URL" width="200"/></td>
    <td> ... your content ... </td>
  </tr></table>
-->

---

## 🧰 Tech Stack

| Layer | Technology |
|---|---|
| **Language** | Go 1.25+ |
| **Telegram MTProto** | [gogram](https://github.com/amarnathcjd/gogram) |
| **Voice Calls / WebRTC** | [ntgcalls](https://github.com/pytgcalls/ntgcalls) (Go bindings) |
| **Database** | MongoDB via `mongo-driver/v2` |
| **Music Download** | yt-dlp + Arc API + SoundCloud |
| **Spotify Metadata** | `zmb3/spotify` |
| **Audio Processing** | FFmpeg |
| **HTTP Client** | `resty.dev/v3` |
| **System Monitoring** | `gopsutil/v3` |
| **Config** | `.env` via `godotenv` + YAML locales |
| **Logging** | `gologging` + Charmbracelet lipgloss |

---

## 🤝 Contributing

Contributions are very welcome! 💖

1. **Fork** the repository
2. **Create your branch** — `git checkout -b feature/cool-new-thing`
3. **Commit your changes** — `git commit -m 'Add cool new thing'`
4. **Push to your branch** — `git push origin feature/cool-new-thing`
5. **Open a Pull Request** 🎉

> Want to add a new music platform? See the [Platform System guide](./internal/platforms/README.md#-how-to-add-a-new-platform) — it's plug-and-play!

---

## 🐛 Bug Reports

- Use `/bug` inside the bot to send a report directly to the dev team
- Or join our **[Support Channel](https://t.me/ECHOWAVESUPPORT)** — we reply fast!
- GitHub Issues are open too 🙏

---

## 📜 License

AnvuMusic is open-source under the **GNU General Public License v3.0**.

```
AnvuMusic — A high-performance Telegram music bot built with Go
Copyright (C) 2026 Team Echo

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
```

---

## 💖 Credits & Support

<div align="center">

| 👤 Role | 🔗 Link |
|---|---|
| 🧑‍💻 **Developer** | [@eceqt](https://t.me/eceqt) |
| 📢 **Updates Channel** | [@ECHOWAVESUPPORT](https://t.me/ECHOWAVESUPPORT) |
| 🏗️ **Core Library** | [gogram](https://github.com/amarnathcjd/gogram) by AmarnathCJD |
| 🎙️ **Voice Engine** | [ntgcalls](https://github.com/pytgcalls/ntgcalls) by pytgcalls team |

<br/>

*If AnvuMusic brings joy to your server, please consider giving the repo a* ⭐ *— it means the world! 🌸*

<br/>

<img src="https://capsule-render.vercel.app/api?type=waving&color=gradient&customColorList=12,20,24&height=100&section=footer&text=AnvuMusic&fontSize=28&fontColor=ffffff&animation=fadeIn&fontAlignY=70" width="100%"/>

</div>
