/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package core

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/locales"
	"main/internal/utils"
)

var F func(chatID int64, key string, values ...locales.Arg) string // overwritten from main.go

func AddMeMarkup(chatID int64) tg.ReplyMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(
				F(chatID, "ADD_ME_BTN"),
				"https://t.me/"+Bot.Me().Username+"?startgroup&admin=invite_users",
			),
		).
		Build()
}

func GetCancelKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(F(chatID, "DOWNLOAD_CANCEL_BTN"), "cancel"),
		).
		Build()
}

func GetBroadcastCancelKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(F(chatID, "BROADCAST_CANCEL_BTN"), "bcast_cancel"),
		).
		Build()
}

func SuppMarkup(chatID int64) tg.ReplyMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(F(chatID, "SUPPORT_BTN"), config.SupportChat),
		).
		Build()
}

func GetStopConfirmMarkup(
	chatID int64,
	r *RoomState,
	isPaused bool,
) tg.ReplyMarkup {
	btn := tg.NewKeyboard()
	prefix := "room:"
	if r.ChannelPlayID() != 0 {
		prefix = "croom:"
	}

	if isPaused {
		btn.AddRow(
			tg.Button.Data(F(chatID, "CONFIRM_RESUME_BTN"), prefix+"resume"),
		)
	} else {
		btn.AddRow(
			tg.Button.Data(F(chatID, "CONFIRM_UNMUTE_BTN"), prefix+"unmute"),
		)
	}

	btn.AddRow(
		tg.Button.Data(F(chatID, "CONFIRM_STOP_BTN"), prefix+"stop"),
	)

	return btn.Build()
}

func GetPlayMarkup(chatID int64, r *RoomState, queued bool) tg.ReplyMarkup {
	btn := tg.NewKeyboard()
	prefix := "room:"
	if r.ChannelPlayID() != 0 {
		prefix = "croom:"
	}
	track := r.Track()
	duration := 0
	if track != nil {
		duration = track.Duration
	}

	// Progress bar row (only when not queued)
	if !queued {
		progress := utils.GetProgressBar(r.Position(), duration)
		progress = formatDuration(
			r.Position(),
		) + " " + progress + " " + formatDuration(
			duration,
		)
		btn.AddRow(
			tg.Button.Data(progress, "progress"),
		)
	}

	// Row 1: Playback controls — Resume, Pause, Skip, Stop
	btn.AddRow(
		tg.Button.Data("▷", prefix+"resume"),
		tg.Button.Data("II", prefix+"pause"),
		tg.Button.Data("‣‣I", prefix+"skip"),
		tg.Button.Data("▢", prefix+"stop"),
	)

	// Row 2: Seek back, Replay, Seek forward
	btn.AddRow(
		tg.Button.Data("↩ 15s", prefix+"seekback_15"),
		tg.Button.Data("⟳", prefix+"replay"),
		tg.Button.Data("15s ↪", prefix+"seek_15"),
	)

	// Row 3: Close button
	btn.AddRow(
		tg.Button.Data(F(chatID, "CLOSE_BTN"), "close"),
	)

	return btn.Build()
}

func GetGroupHelpKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	bot := "https://t.me/" + Bot.Me().Username
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(F(chatID, "GC_HELP_BTN"), bot+"?start=pm_help"),
			tg.Button.URL(F(chatID, "GC_UPDATES_BTN"), config.SupportChannel),
		).
		Build()
}

func GetStartMarkup(chatID int64) tg.ReplyMarkup {
	bot := "https://t.me/" + Bot.Me().Username
	kb := tg.NewKeyboard()

	// Row 1: Add to group
	kb.AddRow(
		tg.Button.URL(F(chatID, "ADD_ME_BTN"), bot+"?startgroup&admin=invite_users"),
	)

	// Row 2: Support (opens support panel), Language
	kb.AddRow(
		tg.Button.Data(F(chatID, "SUPPORT_BTN"), "support_panel"),
		tg.Button.Data(F(chatID, "LANGUAGE_BTN"), "lang"),
	)

	// Row 3: Help
	kb.AddRow(
		tg.Button.Data(F(chatID, "HELP_BTN"), "help_cb"),
	)

	return kb.Build()
}

func GetSupportMarkup(chatID int64) tg.ReplyMarkup {
	kb := tg.NewKeyboard()

	// Row 1 (2×2 grid row 1): Support Group, Updates Channel
	kb.AddRow(
		tg.Button.URL(F(chatID, "SUPPORT_BTN"), config.SupportChat),
		tg.Button.URL(F(chatID, "UPDATES_BTN"), config.SupportChannel),
	)

	// Row 2 (2×2 grid row 2): Owner, Source
	if config.OwnerID != 0 {
		kb.AddRow(
			tg.Button.URL(F(chatID, "OWNER_BTN"), "tg://user?id="+utils.IntToStr(config.OwnerID)),
			tg.Button.URL(F(chatID, "SOURCE_BTN"), config.SupportChannel),
		)
	} else {
		kb.AddRow(
			tg.Button.URL(F(chatID, "SOURCE_BTN"), config.SupportChannel),
		)
	}

	// Row 3: Back to home & Close
	kb.AddRow(
		tg.Button.Data(F(chatID, "HELP_HOME_PANEL_BTN"), "start"),
		tg.Button.Data(F(chatID, "CLOSE_BTN"), "close"),
	)

	return kb.Build()
}

func GetHelpKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_PUBLIC_BTN"),
				"help:public",
			),
			tg.Button.Data(
				F(chatID, "HELP_ADMINS_BTN"),
				"help:admins",
			),
		).
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_OWNER_BTN"),
				"help:owner",
			),
			tg.Button.Data(
				F(chatID, "HELP_SUDOERS_BTN"),
				"help:sudoers",
			),
		).
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_HOME_PANEL_BTN"),
				"start",
			),
		).
		Build()
}

func GetBackKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_BACK_CATEGORIES_BTN"),
				"help:main",
			),
			tg.Button.Data(
				F(chatID, "HELP_HOME_PANEL_BTN"),
				"start",
			),
		).
		Build()
}

func formatDuration(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s) // HH:MM:SS
	}
	return fmt.Sprintf("%02d:%02d", m, s) // MM:SS
}
