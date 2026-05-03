/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"
	"html"

	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/locales"
	"main/internal/utils"
)

func init() {
	helpTexts["/replay"] = `Restart the current track from the beginning.

<u>Usage:</u>
<b>/replay</b> — Restart current track

<b>⚙️ Behavior:</b>
• Resets position to 0:00
• Maintains speed setting
• Continues playback immediately

<b>🔒 Restrictions:</b>
• Only <b>chat admins</b> or <b>authorized users</b> can use this
`
}

func replayHandler(m *telegram.NewMessage) error {
	return handleReplay(m, false)
}

func creplayHandler(m *telegram.NewMessage) error {
	return handleReplay(m, true)
}

func handleReplay(m *telegram.NewMessage, cplay bool) error {
	chatID := m.ChannelID()

	r, err := getEffectiveRoom(m, cplay)
	if err != nil {
		m.Reply(err.Error())
		return telegram.ErrEndGroup
	}

	if !r.IsActiveChat() {
		m.Reply(F(chatID, "room_no_active"))
		return telegram.ErrEndGroup
	}
	t := r.Track()

	if err := r.Replay(); err != nil {
		m.Reply(F(chatID, "replay_failed", locales.Arg{
			"error": err,
		}))
	} else {
		trackTitle := html.EscapeString(utils.ShortTitle(t.Title, 25))
		totalDuration := formatDuration(t.Duration)
		m.Reply(F(chatID, "replay_success", locales.Arg{
			"title":    trackTitle,
			"duration": totalDuration,
			"speed":    fmt.Sprintf("%.2f", r.Speed()),
		}))
	}

	return telegram.ErrEndGroup
}
