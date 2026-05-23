/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var pasteHTTPClient = &http.Client{Timeout: 10 * time.Second}

func init() {
	helpTexts["/paste"] = `<i>Upload text to a paste service and get a shareable link.</i>

<u>Usage:</u>
<b>/paste [text]</b>   — Paste provided text
<b>/paste</b> (reply) — Paste replied message text

<b>💡 Use Cases:</b>
• Share long messages cleanly
• Upload logs for debugging
• Share code snippets`
}

func pasteHandler(m *tg.NewMessage) error {
	text := strings.TrimSpace(m.Args())

	if text == "" && m.IsReply() {
		replied, err := m.GetReplyMessage()
		if err == nil && replied != nil {
			text = replied.Text()
		}
	}

	if text == "" {
		m.Reply("⚠️ Provide text to paste, or reply to a message.")
		return tg.ErrEndGroup
	}

	processing, _ := m.Reply("📤 <b>Uploading to paste service...</b>")

	url, err := uploadToPaste(text)
	if err != nil {
		gologging.ErrorF("Paste upload failed: %v", err)
		if processing != nil {
			processing.Edit("❌ <b>Failed to upload paste.</b> Try again later.")
		}
		return tg.ErrEndGroup
	}

	result := fmt.Sprintf(
		"✅ <b>Pasted successfully!</b>\n\n"+
			"🔗 <code>%s</code>",
		url,
	)

	if processing != nil {
		processing.Edit(result)
	} else {
		m.Reply(result)
	}

	return tg.ErrEndGroup
}

func uploadToPaste(content string) (string, error) {
	// Using batbin (compatible with haste-server protocol)
	payload := map[string]string{"content": content}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://batbin.me/api/v2/paste", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pasteHTTPClient.Do(req)
	if err != nil {
		return tryFallbackPaste(content)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Success bool   `json:"success"`
		Link    string `json:"link"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil || !result.Success {
		return tryFallbackPaste(content)
	}

	return result.Link, nil
}

func tryFallbackPaste(content string) (string, error) {
	// Fallback: nekobin
	payload := map[string]interface{}{
		"document": map[string]string{"content": content},
	}
	body, _ := json.Marshal(payload)

	resp, err := pasteHTTPClient.Post("https://nekobin.com/api/documents", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("all paste services failed")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			Key string `json:"key"`
		} `json:"result"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil || !result.OK {
		return "", fmt.Errorf("nekobin failed too")
	}

	return "https://nekobin.com/" + result.Result.Key, nil
}
