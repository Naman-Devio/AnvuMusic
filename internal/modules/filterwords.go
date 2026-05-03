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
)

func init() {
	helpTexts["/filter"] = `<i>Set keyword auto-replies for your group.</i>

<u>Usage:</u>
<b>/filter [keyword] [response]</b> — Add a filter
<b>/filters</b>                    — List all filters
<b>/stopfilter [keyword]</b>       — Remove a filter

<b>💡 Examples:</b>
<code>/filter hello Hi there! Welcome!</code>
<code>/filter rules Please read #rules channel</code>

<b>⚙️ Notes:</b>
• Keyword matching is case-insensitive
• Supports partial word match
• Only admins/authorized users can add filters`

	helpTexts["/filters"] = `<i>List all keyword filters set in this chat.</i>

<u>Usage:</u>
<b>/filters</b> — Show all filters`

	helpTexts["/stopfilter"] = `<i>Remove a keyword filter.</i>

<u>Usage:</u>
<b>/stopfilter [keyword]</b> — Remove specific filter`
}

func filterAddHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	args := strings.SplitN(m.Args(), " ", 2)

	if len(args) < 2 || args[0] == "" || args[1] == "" {
		m.Reply("⚠️ Usage: <code>/filter [keyword] [response]</code>")
		return tg.ErrEndGroup
	}

	keyword := strings.ToLower(strings.TrimSpace(args[0]))
	response := strings.TrimSpace(args[1])

	if err := database.AddFilter(chatID, keyword, response); err != nil {
		gologging.ErrorF("AddFilter failed chat=%d: %v", chatID, err)
		m.Reply("❌ Failed to save filter.")
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf(
		"✅ <b>Filter added!</b>\n\n"+
			"➤ <b>Keyword:</b> <code>%s</code>\n"+
			"➤ <b>Response:</b> <i>%s</i>",
		keyword, response,
	))
	return tg.ErrEndGroup
}

func filtersListHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()

	filters, err := database.GetFilters(chatID)
	if err != nil || len(filters) == 0 {
		m.Reply("📭 <b>No filters set in this chat.</b>")
		return tg.ErrEndGroup
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Filters in this chat (%d):</b>\n\n", len(filters)))
	for i, kw := range filters {
		sb.WriteString(fmt.Sprintf("  %d. <code>%s</code>\n", i+1, kw))
	}

	m.Reply(sb.String())
	return tg.ErrEndGroup
}

func filterStopHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	keyword := strings.ToLower(strings.TrimSpace(m.Args()))

	if keyword == "" {
		m.Reply("⚠️ Usage: <code>/stopfilter [keyword]</code>")
		return tg.ErrEndGroup
	}

	if err := database.RemoveFilter(chatID, keyword); err != nil {
		m.Reply(fmt.Sprintf("❌ Could not remove filter: <code>%s</code>", err.Error()))
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf("✅ Filter <code>%s</code> removed.", keyword))
	return tg.ErrEndGroup
}

// checkFilters is called from message watcher to auto-reply on keyword match
func checkFilters(m *tg.NewMessage) {
	if m.Sender == nil || m.Message == nil {
		return
	}

	chatID := m.ChannelID()
	text := strings.ToLower(m.Text())
	if text == "" {
		return
	}

	response, err := database.MatchFilter(chatID, text)
	if err != nil || response == "" {
		return
	}

	m.Reply(response)
}
