/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/locales"
	"main/internal/utils"
)

func init() {
	helpTexts["/autoplay"] = `<i>Toggle autoplay for this chat.</i>

<u>Usage:</u>
<b>/autoplay</b> — Show current autoplay state
<b>/autoplay on</b> — Enable autoplay
<b>/autoplay off</b> — Disable autoplay

<b>⚙️ Behavior:</b>
• When the queue runs empty, the bot automatically plays a recommended track (YouTube Mix) based on the last played song
• Autoplay keeps playing related tracks until it is turned off or playback is stopped

<b>🔒 Restrictions:</b>
• Only <b>chat admins</b> or <b>authorized users</b> can use this

<b>💡 Examples:</b>
<code>/autoplay on</code> — Enable autoplay
<code>/autoplay off</code> — Disable autoplay

<b>⚠️ Note:</b>
Autoplay only kicks in after the current queue has finished.`
}

func autoplayHandler(m *telegram.NewMessage) error {
	return handleAutoplay(m, false)
}

func handleAutoplay(m *telegram.NewMessage, cplay bool) error {
	arg := strings.ToLower(m.Args())

	r, err := getEffectiveRoom(m, cplay)
	if err != nil {
		m.Reply(err.Error())
		return telegram.ErrEndGroup
	}
	chatID := m.ChannelID()

	if !r.IsActiveChat() {
		m.Reply(F(chatID, "room_no_active"))
		return telegram.ErrEndGroup
	}

	r.Parse()

	if arg == "" {
		state := F(chatID, "disabled")
		cmd := getCommand(m) + " on"
		if r.Autoplay() {
			state = F(chatID, "enabled")
			cmd = getCommand(m) + " off"
		}

		m.Reply(F(chatID, "autoplay_current_state", locales.Arg{
			"state": state,
			"cmd":   cmd,
		}))
		return telegram.ErrEndGroup
	}

	var newState bool
	switch arg {
	case "on", "enable", "true", "1":
		newState = true
	case "off", "disable", "false", "0":
		newState = false
	}

	r.SetAutoplay(newState)

	state := F(chatID, "disabled")
	if newState {
		state = F(chatID, "enabled")
	}

	m.Reply(F(chatID, "autoplay_updated", locales.Arg{
		"state": state,
		"user":  utils.MentionHTML(m.Sender),
	}))

	return telegram.ErrEndGroup
}
