/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"main/internal/config"
	"main/internal/core"
	"main/internal/database"
	"main/internal/utils"
)

const startSticker = "CAACAgUAAyEFAAS-Y9n1AAJio2lnx-7YpGWYAAGKW3ILDhBG7Gl0qgACoxsAAnmZoFUj0zVj_zPVix4E"

var loadingFrames = []string{
	"😈 <b>ᴅɪηɢ ᴅᴏηɢ.</b>",
	"😈 <b>ᴅɪηɢ ᴅᴏηɢ..</b>",
	"😈 <b>ᴅɪηɢ ᴅᴏηɢ...</b>",
	"😎 <b>sᴛᴧʀᴛɪηɢ.</b>",
	"😎 <b>sᴛᴧʀᴛɪηɢ..</b>",
	"😎 <b>sᴛᴧʀᴛɪηɢ...</b>",
	"💖 <b>ʜєʏ ʙᴧʙʏ!</b>",
	"🌺 <b>ᴀɴᴠᴜ ꭙ ϻᴜsɪᴄ ♪\nsᴛᴧʀᴛed!</b>",
}

func init() {
	helpTexts["/start"] = `<i>Start the bot and show main menu.</i>`
}

func buildStartCaptionFor(user *tg.UserObj, bot *tg.UserObj) string {
	uptime := formatUptime(time.Since(config.StartTime))

	cpuStr := "N/A"
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		cpuStr = fmt.Sprintf("%.1f%%", pct[0])
	}

	ramStr := "N/A"
	if v, err := mem.VirtualMemory(); err == nil {
		ramStr = fmt.Sprintf("%.1f%%", v.UsedPercent)
	}

	storageStr := "N/A"
	if d, err := disk.Usage("/"); err == nil {
		storageStr = fmt.Sprintf("%.1f%%", d.UsedPercent)
	}

	userMention := utils.MentionHTML(user)
	botMention := utils.MentionHTML(bot)

	return fmt.Sprintf(
		"╔══〔 <b>ɪɴғᴏʀᴍᴀᴛɪᴏɴ</b> 〕══●\n"+
			"║ ➤ <b>ʜᴇʏ,</b> %s\n"+
			"║ ➤ <b>ɪ ᴀᴍ</b> %s\n"+
			"╚══════════════════●\n\n"+
			"<blockquote>▸ <b>ᴛʜᴇ ᴍᴏsᴛ ᴘᴏᴡᴇʀғᴜʟ &amp; ғᴀsᴛᴇsᴛ ᴍᴜsɪᴄ ᴘʟᴀʏᴇʀ ʙᴏᴛ!</b></blockquote>\n"+
			"<blockquote>"+
			"❖ <b>ᴜᴘᴛɪᴍᴇ :</b> <code>%s</code>\n"+
			"❖ <b>sᴛᴏʀᴀɢᴇ :</b> <code>%s</code>\n"+
			"❖ <b>ᴄᴘᴜ ʟᴏᴀᴅ :</b> <code>%s</code>\n"+
			"❖ <b>ʀᴀᴍ ᴜsᴀɢᴇ :</b> <code>%s</code>"+
			"</blockquote>\n"+
			"●══════════════════●\n"+
			"<blockquote>✦ <b>ᴘᴏᴡᴇʀᴇᴅ ʙʏ »</b> <a href=\""+config.SupportChannel+"\">EchoWave · Anvu</a></blockquote>\n"+
			"●══════════════════●",
		userMention, botMention,
		uptime, storageStr, cpuStr, ramStr,
	)
}

func buildStartCaption(m *tg.NewMessage) string {
	return buildStartCaptionFor(m.Sender, m.Client.Me())
}

func startHandler(m *tg.NewMessage) error {
	if m.ChatType() != tg.EntityUser {
		database.AddServedChat(m.ChannelID())
		m.Reply(F(m.ChannelID(), "start_group"))
		return tg.ErrEndGroup
	}

	if config.MustJoin != "" && !checkMustJoin(m) {
		return tg.ErrEndGroup
	}

	arg := m.Args()
	database.AddServedUser(m.ChannelID())

	if arg != "" {
		gologging.Info("Got Start parameter: " + arg + " in ChatID: " + utils.IntToStr(m.ChannelID()))
	}

	switch arg {
	case "pm_help":
		helpHandler(m)
	default:
		// 1. Loading animation
		loading, err := m.Reply(loadingFrames[0])
		if err == nil && loading != nil {
			for _, frame := range loadingFrames[1:] {
				time.Sleep(100 * time.Millisecond)
				loading.Edit(frame)
			}
			time.Sleep(100 * time.Millisecond)
			loading.Delete()
		}

		// 2. Sticker
		m.ReplyMedia(startSticker, &tg.MediaOptions{})

		// 3. Image + caption + buttons
		caption := buildStartCaption(m)
		_, err = m.RespondMedia(config.StartImage, &tg.MediaOptions{
			Caption:     caption,
			NoForwards:  true,
			ReplyMarkup: core.GetStartMarkup(m.ChannelID()),
		})
		if err != nil {
			gologging.Error("[start] media reply failed: " + err.Error())
			m.Respond(caption, &tg.SendOptions{
				NoForwards:  true,
				ReplyMarkup: core.GetStartMarkup(m.ChannelID()),
			})
		}
	}

	if config.LoggerID != 0 && isLoggerEnabled() {
		uName := "N/A"
		if m.Sender.Username != "" {
			uName = "@" + m.Sender.Username
		}
		msg := fmt.Sprintf(
			"▶️ <b>%s started the bot</b>\n\n<b>Username:</b> %s\n<b>ID:</b> <code>%d</code>",
			utils.MentionHTML(m.Sender), uName, m.SenderID(),
		)
		if _, err := m.Client.SendMessage(config.LoggerID, msg); err != nil {
			gologging.Error("Failed to send logger_bot_started msg, Err: " + err.Error())
		}
	}
	return tg.ErrEndGroup
}

func startCB(cb *tg.CallbackQuery) error {
	cb.Answer("")
	caption := buildStartCaptionFor(cb.Sender, cb.Client.Me()) + "\n\n" + F(cb.ChannelID(), "start_cb_panel")
	sendOpt := &tg.SendOptions{
		ReplyMarkup: core.GetStartMarkup(cb.ChannelID()),
		NoForwards:  true,
	}
	if config.StartImage != "" {
		sendOpt.Media = config.StartImage
	}
	cb.Edit(caption, sendOpt)
	return tg.ErrEndGroup
}
