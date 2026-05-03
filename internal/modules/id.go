/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/utils"
)

func init() {
	helpTexts["/id"] = `<i>Get Telegram IDs of users, chats, or channels.</i>

<u>Usage:</u>
<b>/id</b>           — Your own ID + current chat ID
<b>/id</b> (reply)  — Target user's ID + info

<b>📋 Shows:</b>
• User ID, username, DC
• Chat/Channel ID
• Message ID`
}

func idHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()
	senderID := m.SenderID()

	var sb string

	if m.IsReply() {
		replied, err := m.GetReplyMessage()
		if err == nil && replied != nil && replied.Sender != nil {
			u := replied.Sender
			name := utils.FullName(u)
			uname := "N/A"
			if u.Username != "" {
				uname = "@" + u.Username
			}
			dcID := u.DcID

			sb = fmt.Sprintf(
				"<b>👤 ᴜsᴇʀ ɪɴғᴏ</b>\n\n"+
					"➤ <b>ɴᴀᴍᴇ :</b> %s\n"+
					"➤ <b>ᴜsᴇʀɴᴀᴍᴇ :</b> <code>%s</code>\n"+
					"➤ <b>ɪᴅ :</b> <code>%d</code>\n"+
					"➤ <b>ᴅᴄ :</b> <code>%d</code>\n"+
					"➤ <b>ᴍsɢ ɪᴅ :</b> <code>%d</code>",
				utils.MentionHTML(u), uname,
				u.ID, dcID, replied.ID,
			)
		} else {
			sb = "⚠️ Could not fetch replied user's info."
		}
	} else {
		chatInfo := ""
		if !m.IsPrivate() {
			chatInfo = fmt.Sprintf("\n➤ <b>ᴄʜᴀᴛ ɪᴅ :</b> <code>%d</code>", chatID)
		}
		sb = fmt.Sprintf(
			"<b>🆔 ʏᴏᴜʀ ɪɴғᴏ</b>\n\n"+
				"➤ <b>ɴᴀᴍᴇ :</b> %s\n"+
				"➤ <b>ɪᴅ :</b> <code>%d</code>%s",
			utils.MentionHTML(m.Sender), senderID, chatInfo,
		)
	}

	m.Reply(sb)
	return tg.ErrEndGroup
}
