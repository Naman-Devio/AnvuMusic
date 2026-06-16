/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/core"
	"main/internal/locales"
)

func init() {
	helpTexts["/active"] = `<i>Show all active voice chat sessions.</i>

<u>Usage:</u>
<b>/active</b> or <b>/ac</b> — List active chats

<b>📊 Information Shown:</b>
• Total active voice chats

<b>🔒 Restrictions:</b>
• <b>Sudo users</b> only

<b>💡 Use Case:</b>
Monitor exact bot usage.`

	keys := []string{"/ac", "/activevc", "/activevoice"}
	for _, k := range keys {
		helpTexts[k] = helpTexts["/active"]
	}
}

func activeHandler(m *telegram.NewMessage) error {
	chatID := m.ChannelID()

	// Map to store unique, currently active voice chat connections
	ntgChats := make(map[int64]struct{})

	// Iterate through assistants and only count actual live NTG calls
	core.Assistants.ForEach(func(a *core.Assistant) {
		if a == nil || a.Ntg == nil {
			return
		}
		for id := range a.Ntg.Calls() {
			ntgChats[id] = struct{}{}
		}
	})

	// The exact number of active voice chats
	activeCount := len(ntgChats)

	// Send the exact count without the broken/stale logic
	msg := F(chatID, "active_chats_info", locales.Arg{
		"count": activeCount,
	})

	m.Reply(msg)
	return telegram.ErrEndGroup
}
