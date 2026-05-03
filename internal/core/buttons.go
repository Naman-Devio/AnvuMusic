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

	progress := utils.GetProgressBar(r.Position(), duration)
	progress = formatDuration(
		r.Position(),
	) + " " + progress + " " + formatDuration(
		duration,
	)

	if !queued {
		btn.AddRow(
			tg.Button.Data(progress, "progress"),
		)
	}
	btn.AddRow(
		tg.Button.Data(F(chatID, "PLAY_BTN_RESUME"), prefix+"resume"),
		tg.Button.Data(F(chatID, "PLAY_BTN_PAUSE"), prefix+"pause"),
		tg.Button.Data(F(chatID, "PLAY_BTN_SKIP"), prefix+"skip"),
	)
	btn.AddRow(
		tg.Button.Data(F(chatID, "PLAY_BTN_REPLAY"), "room:replay"),
		tg.Button.Data(F(chatID, "PLAY_BTN_STOP"), prefix+"stop"),
	)

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
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(
				F(chatID, "ADD_ME_BTN"),
				bot+"?startgroup&admin=invite_users",
			),
			tg.Button.Data(
				F(chatID, "START_HELP_BTN"),
				"help_cb",
			),
		).
		AddRow(
			tg.Button.URL(
				F(chatID, "UPDATES_BTN"),
				config.SupportChannel,
			),
			tg.Button.URL(
				F(chatID, "SUPPORT_BTN"),
				config.SupportChat,
			),
		).
		AddRow(
			tg.Button.URL(
				F(chatID, "SOURCE_BTN"),
				config.SupportChannel,
			),
			tg.Button.URL(
				F(chatID, "START_DOCS_BTN"),
				bot+"?start=pm_help",
			),
		).
		AddRow(
			tg.Button.Data(
				F(chatID, "START_REFRESH_BTN"),
				"start",
			),
		).
		Build()
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
