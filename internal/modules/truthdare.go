/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
)

const (
	truthAPIURL = "https://api.truthordarebot.xyz/v1/truth"
	dareAPIURL  = "https://api.truthordarebot.xyz/v1/dare"
)

var todHTTPClient = &http.Client{Timeout: 8 * time.Second}

func init() {
	helpTexts["/truth"] = `<i>Get a random Truth question to answer.</i>

<u>Usage:</u>
<b>/truth</b> — Fetch a random truth question

<b>🎮 Game Tip:</b>
Use with /dare to play Truth or Dare in your group!`

	helpTexts["/dare"] = `<i>Get a random Dare challenge.</i>

<u>Usage:</u>
<b>/dare</b> — Fetch a random dare challenge`
}

func fetchToDQuestion(apiURL string) (string, error) {
	resp, err := todHTTPClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Question string `json:"question"`
		ID       string `json:"id"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Question == "" {
		return "", fmt.Errorf("empty response")
	}

	return result.Question, nil
}

func truthHandler(m *tg.NewMessage) error {
	q, err := fetchToDQuestion(truthAPIURL)
	if err != nil {
		m.Reply("❌ Could not fetch a truth question. Try again in a bit!")
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf(
		"🔮 <b>ᴛʀᴜᴛʜ :</b>\n\n"+
			"<blockquote>%s</blockquote>\n\n"+
			"<i>↩️ Reply to answer!</i>",
		q,
	))
	return tg.ErrEndGroup
}

func dareHandler(m *tg.NewMessage) error {
	q, err := fetchToDQuestion(dareAPIURL)
	if err != nil {
		m.Reply("❌ Could not fetch a dare challenge. Try again in a bit!")
		return tg.ErrEndGroup
	}

	m.Reply(fmt.Sprintf(
		"🎯 <b>ᴅᴀʀᴇ :</b>\n\n"+
			"<blockquote>%s</blockquote>\n\n"+
			"<i>⚡ Do it or skip!</i>",
		q,
	))
	return tg.ErrEndGroup
}
