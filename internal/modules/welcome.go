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

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/database"
	"main/internal/utils"
)

// welcomeEnabled tracks per-chat welcome toggle; backed by DB via database package
var (
	welcomeMu sync.RWMutex
)

func init() {
	helpTexts["/welcome"] = `<i>Toggle welcome/goodbye messages for new members.</i>

<u>Usage:</u>
<b>/welcome on</b>  — Enable welcome messages
<b>/welcome off</b> — Disable welcome messages
<b>/welcome</b>     — Check current status

<b>✨ Features:</b>
• Auto-greets new members with name mention
• Sends goodbye when a member leaves
• Can be customized per chat`

	helpTexts["/goodbye"] = helpTexts["/welcome"]
}

func welcomeHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	arg := m.Args()

	switch arg {
	case "on", "enable", "true", "1":
		if err := database.SetWelcome(chatID, true); err != nil {
			m.Reply(fmt.Sprintf("❌ Failed to enable welcome: <code>%s</code>", err.Error()))
			return tg.ErrEndGroup
		}
		m.Reply("✅ <b>Welcome messages enabled!</b>\n\nI'll greet new members when they join.")
	case "off", "disable", "false", "0":
		if err := database.SetWelcome(chatID, false); err != nil {
			m.Reply(fmt.Sprintf("❌ Failed to disable welcome: <code>%s</code>", err.Error()))
			return tg.ErrEndGroup
		}
		m.Reply("🔕 <b>Welcome messages disabled.</b>")
	default:
		enabled, err := database.GetWelcome(chatID)
		if err != nil {
			m.Reply("⚠️ Could not fetch welcome status.")
			return tg.ErrEndGroup
		}
		status := "🔴 <b>Disabled</b>"
		if enabled {
			status = "🟢 <b>Enabled</b>"
		}
		m.Reply(fmt.Sprintf(
			"<b>Welcome Status:</b> %s\n\n"+
				"Use <code>/welcome on</code> or <code>/welcome off</code> to toggle.",
			status,
		))
	}

	return tg.ErrEndGroup
}

// handleNewMember is called from chat action handler when a user joins
func handleNewMember(client *tg.Client, chatID int64, user *tg.UserObj) {
	enabled, err := database.GetWelcome(chatID)
	if err != nil || !enabled {
		return
	}

	mention := utils.MentionHTMLFromUser(user)
	msg := fmt.Sprintf(
		"🎉 <b>ᴡᴇʟᴄᴏᴍᴇ</b> %s <b>ᴛᴏ ᴛʜᴇ ɢʀᴏᴜᴘ!</b>\n\n"+
			"<i>✨ ʜᴏᴘᴇ ʏᴏᴜ ᴇɴᴊᴏʏ ʏᴏᴜʀ sᴛᴀʏ ɪɴ ʜᴇʀᴇ ~</i>",
		mention,
	)

	if _, err := client.SendMessage(chatID, msg); err != nil {
		gologging.ErrorF("Failed to send welcome msg in chat %d: %v", chatID, err)
	}
}

// handleLeftMember is called from chat action handler when a user leaves
func handleLeftMember(client *tg.Client, chatID int64, user *tg.UserObj) {
	enabled, err := database.GetWelcome(chatID)
	if err != nil || !enabled {
		return
	}

	mention := utils.MentionHTMLFromUser(user)
	msg := fmt.Sprintf(
		"👋 <b>%s</b> <i>ʜᴀs ʟᴇꜰᴛ ᴛʜᴇ ɢʀᴏᴜᴘ.</i>",
		mention,
	)

	if _, err := client.SendMessage(chatID, msg); err != nil {
		gologging.ErrorF("Failed to send goodbye msg in chat %d: %v", chatID, err)
	}
}
