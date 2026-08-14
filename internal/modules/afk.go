/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/utils"
)

type afkEntry struct {
	Since time.Time
}

var (
	afkMu    sync.RWMutex
	afkStore = make(map[int64]*afkEntry) // userID → entry
)

func init() {
	helpTexts["/afk"] = `<i>Mark yourself as AFK (Away From Keyboard).</i>	<u>Usage:</u>
<b>/afk</b> — Go AFK

<b>⚙️ Behavior:</b>
• Others are notified if they mention you while AFK
• You are automatically unmarked when you send a message`
}

func afkHandler(m *tg.NewMessage) error {
	if m.Sender == nil {
		return tg.ErrEndGroup
	}

	userID := m.SenderID()

	afkMu.Lock()
	afkStore[userID] = &afkEntry{
		Since: time.Now(),
	}
	afkMu.Unlock()

	mention := utils.MentionHTML(m.Sender)
	text := fmt.Sprintf("😴 %s <b>is now AFK.</b>", mention)

	m.Reply(text)
	return tg.ErrEndGroup
}

// checkAFK is called from message handler to auto-unmark and notify on mention
func checkAFK(m *tg.NewMessage) {
	if m.Sender == nil {
		return
	}

	senderID := m.SenderID()

	// Auto-unmark if the AFK user sends a message
	afkMu.Lock()
	if entry, ok := afkStore[senderID]; ok {
		delete(afkStore, senderID)
		afkMu.Unlock()
		elapsed := formatUptime(time.Since(entry.Since))
		mention := utils.MentionHTML(m.Sender)
		m.Reply(fmt.Sprintf(
			"👋 %s <b>is back!</b> <i>(was AFK for %s)</i>",
			mention, elapsed,
		))
		return
	}
	afkMu.Unlock()

	// Notify if a mentioned user is AFK
	if m.Message == nil {
		return
	}

	for _, entity := range m.Message.Entities {
		var mentionedID int64
		switch e := entity.(type) {
		case *tg.MessageEntityMentionName:
			mentionedID = e.UserID
		}
		if mentionedID == 0 {
			continue
		}

		afkMu.RLock()
		entry, ok := afkStore[mentionedID]
		afkMu.RUnlock()

		if !ok {
			continue
		}

		elapsed := formatUptime(time.Since(entry.Since))
		text := fmt.Sprintf(
			"<b>💤 That user is AFK</b> <i>(for %s)</i>",
			elapsed,
		)
		m.Reply(text)
		break
	}
}
