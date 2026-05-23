/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/core"
)

// checkMustJoin returns true if user is a member, false and sends join prompt if not.
func checkMustJoin(m *tg.NewMessage) bool {
	if config.MustJoin == "" {
		return true
	}

	peer, err := core.Bot.ResolvePeer(config.MustJoin)
	if err != nil {
		return true // can't resolve channel, let through
	}

	var chatID int64
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		chatID = p.ChannelID
	case *tg.InputPeerChat:
		chatID = p.ChatID
	default:
		return true
	}

	_, err = core.Bot.GetChatMember(chatID, m.SenderID())
	if err != nil {
		errStr := err.Error()
		// only block if Telegram explicitly says user is not a participant
		if !strings.Contains(errStr, "USER_NOT_PARTICIPANT") &&
			!strings.Contains(errStr, "PARTICIPANT_ID_INVALID") {
			return true
		}
	} else {
		return true
	}

	// Not a member — build join link
	username := strings.TrimPrefix(config.MustJoin, "@")
	link := "https://t.me/" + username

	m.RespondMedia(config.StartImage, &tg.MediaOptions{
		Caption: "» ʏᴏᴜ ɴᴇᴇᴅ ᴛᴏ ᴊᴏɪɴ ᴏᴜʀ ᴄʜᴀɴɴᴇʟ ꜰɪʀsᴛ ᴛᴏ ᴜsᴇ ᴍᴇ!\n\nᴀꜰᴛᴇʀ ᴊᴏɪɴɪɴɢ, sᴇɴᴅ /start ᴀɢᴀɪɴ.",
		ReplyMarkup: tg.NewKeyboard().AddRow(
			tg.Button.URL("ᴊᴏɪɴ ᴄʜᴀɴɴᴇʟ", link),
		).Build(),
	})
	return false
}
