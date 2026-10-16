/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"strings"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/core"
	"main/internal/database"
	"main/internal/locales"
	"main/internal/utils"
)

func init() {
	helpTexts["/settings"] = `<i>Manage this chat's playback settings.</i>

<u>Usage:</u>
<b>/settings</b> — Open the settings panel

<b>⚙️ Available toggles:</b>
• <b>Play mode</b> — restrict <code>/play</code> to admins & authorized users
• <b>Admin mode</b> — open admin commands to everyone, or keep them admin-only
• <b>Command delete</b> — auto-delete command messages after use

<b>🔒 Restrictions:</b>
• Only <b>chat admins</b> can change settings`
}

func settingsHandler(m *tg.NewMessage) error {
	chatID := m.ChannelID()

	playMode, _ := database.PlayMode(chatID)
	adminMode, _ := database.AdminMode(chatID)
	cmdDelete, _ := database.CmdDelete(chatID)
	lang, _ := database.Language(chatID)

	m.Reply(buildSettingsText(chatID, playMode, adminMode, cmdDelete, lang), &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetSettingsMarkup(chatID, playMode, adminMode, cmdDelete),
	})
	return tg.ErrEndGroup
}

func settingsCallback(cb *tg.CallbackQuery) error {
	opt := &tg.CallbackOptions{Alert: true}
	chatID := cb.ChannelID()

	isAdmin, err := utils.IsChatAdmin(cb.Client, chatID, cb.SenderID)
	if err != nil || !isAdmin {
		cb.Answer(F(chatID, "only_admin_or_auth_cb"), opt)
		return tg.ErrEndGroup
	}

	action := strings.TrimPrefix(cb.DataString(), "settings:")
	switch action {
	case "play":
		cur, _ := database.PlayMode(chatID)
		if err := database.SetPlayMode(chatID, !cur); err != nil {
			gologging.ErrorF("SetPlayMode failed for %d: %v", chatID, err)
		}
	case "admin":
		cur, _ := database.AdminMode(chatID)
		next := "admins"
		if cur == "admins" {
			next = "everyone"
		}
		if err := database.SetAdminMode(chatID, next); err != nil {
			gologging.ErrorF("SetAdminMode failed for %d: %v", chatID, err)
		}
	case "delete":
		cur, _ := database.CmdDelete(chatID)
		if err := database.SetCmdDelete(chatID, !cur); err != nil {
			gologging.ErrorF("SetCmdDelete failed for %d: %v", chatID, err)
		}
	default:
		cb.Answer(F(chatID, "unknown_action"), opt)
		return tg.ErrEndGroup
	}

	playMode, _ := database.PlayMode(chatID)
	adminMode, _ := database.AdminMode(chatID)
	cmdDelete, _ := database.CmdDelete(chatID)
	lang, _ := database.Language(chatID)

	text := buildSettingsText(chatID, playMode, adminMode, cmdDelete, lang)
	if _, err := cb.Edit(text, &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetSettingsMarkup(chatID, playMode, adminMode, cmdDelete),
	}); err != nil {
		gologging.ErrorF("Edit error: %v", err)
	}

	cb.Answer(F(chatID, "settings_updated"), opt)
	return tg.ErrEndGroup
}

func buildSettingsText(
	chatID int64,
	playMode bool,
	adminMode string,
	cmdDelete bool,
	lang string,
) string {
	playState := F(chatID, "disabled")
	if playMode {
		playState = F(chatID, "enabled")
	}

	adminState := F(chatID, "SETTINGS_ADMIN_STATE_ADMINS")
	if adminMode == "everyone" {
		adminState = F(chatID, "SETTINGS_ADMIN_STATE_EVERYONE")
	}

	deleteState := F(chatID, "disabled")
	if cmdDelete {
		deleteState = F(chatID, "enabled")
	}

	if lang == "" {
		lang = "en"
	}

	return F(chatID, "settings_menu", locales.Arg{
		"play":   playState,
		"admin":  adminState,
		"delete": deleteState,
		"lang":   lang,
	})
}
