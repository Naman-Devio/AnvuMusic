/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"
	"strings"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/database"
	"main/internal/utils"
)

func init() {
	helpTexts["/gban"] = `<i>Globally ban a user across all chats served by this bot.</i>

<u>Usage:</u>
<b>/gban [user_id/reply] [reason]</b>

<b>⚙️ Behavior:</b>
• Bans user from all served chats
• Logs ban to owner/logger

<b>🔒 Restrictions:</b>
• Owner / Sudo only`

	helpTexts["/ungban"] = `<i>Remove a global ban.</i>

<u>Usage:</u>
<b>/ungban [user_id/reply]</b>`

	helpTexts["/gbans"] = `<i>List all globally banned users.</i>`
}

func gbanHandler(m *tg.NewMessage) error {
	target, err := utils.ExtractUserObj(m)
	if err != nil {
		m.Reply("⚠️ " + err.Error())
		return tg.ErrEndGroup
	}

	reason := "No reason provided"
	args := strings.SplitN(m.Args(), " ", 2)
	if len(args) >= 2 {
		reason = strings.TrimSpace(args[1])
	}

	if err := database.GBanUser(target.ID, reason); err != nil {
		gologging.ErrorF("GBan failed user=%d: %v", target.ID, err)
		m.Reply("❌ Failed to globally ban user.")
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf(
		"🔨 <b>Globally Banned:</b> %s\n"+
			"➤ <b>ID:</b> <code>%d</code>\n"+
			"➤ <b>Reason:</b> <i>%s</i>",
		utils.MentionHTML(target), target.ID, reason,
	))
	return tg.ErrEndGroup
}

func ungbanHandler(m *tg.NewMessage) error {
	target, err := utils.ExtractUserObj(m)
	if err != nil {
		m.Reply("⚠️ " + err.Error())
		return tg.ErrEndGroup
	}

	if err := database.UnGBanUser(target.ID); err != nil {
		m.Reply("❌ Failed to remove global ban.")
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf(
		"✅ <b>Global ban lifted</b> for %s (<code>%d</code>).",
		utils.MentionHTML(target), target.ID,
	))
	return tg.ErrEndGroup
}

func gbansListHandler(m *tg.NewMessage) error {
	bans, err := database.GetGBans()
	if err != nil {
		m.Reply("❌ Failed to fetch gban list.")
		return tg.ErrEndGroup
	}

	if len(bans) == 0 {
		m.Reply("✅ <b>No globally banned users.</b>")
		return tg.ErrEndGroup
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>🔨 Globally Banned Users (%d):</b>\n\n", len(bans)))
	for i, b := range bans {
		if i >= 20 {
			sb.WriteString(fmt.Sprintf("<i>...and %d more.</i>", len(bans)-20))
			break
		}
		sb.WriteString(fmt.Sprintf("  %d. <code>%d</code> — <i>%s</i>\n", i+1, b.UserID, b.Reason))
	}

	m.Reply(sb.String())
	return tg.ErrEndGroup
}
