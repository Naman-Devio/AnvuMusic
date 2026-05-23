/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"fmt"
	"runtime"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"main/internal/config"
	"main/internal/core"
	"main/internal/utils"
)

const aliveSticker = "CAACAgUAAyEFAAS-Y9n1AAJio2lnx-7YpGWYAAGKW3ILDhBG7Gl0qgACoxsAAnmZoFUj0zVj_zPVix4E"

func init() {
	helpTexts["/alive"] = `<i>Check if the bot is alive and kicking.</i>

<u>Usage:</u>
<b>/alive</b> — Shows bot status with system info

<b>📊 Shows:</b>
• Bot version & uptime
• CPU & RAM snapshot
• Go runtime info
• Active voice sessions`
}

func aliveHandler(m *tg.NewMessage) error {
	cpuStr := "N/A"
	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		cpuStr = fmt.Sprintf("%.1f%%", pct[0])
	}

	ramStr := "N/A"
	if v, err := mem.VirtualMemory(); err == nil {
		ramStr = fmt.Sprintf("%.1f%%", v.UsedPercent)
	}

	uptime := formatUptime(time.Since(config.StartTime))
	activeRooms := core.ActiveRoomsCount()
	goVer := runtime.Version()

	caption := fmt.Sprintf(
		"╔══〔 <b>ᴀɴᴠᴜ × ᴍᴜsɪᴄ</b> 〕══●\n"+
			"║ ➤ <b>sᴛᴀᴛᴜs :</b> <code>✅ ᴏɴʟɪɴᴇ</code>\n"+
			"║ ➤ <b>ᴘɪɴɢ :</b> <code>ᴀʟɪᴠᴇ ✓</code>\n"+
			"╠══════════════════●\n"+
			"║ ➤ <b>ʙᴏᴛ :</b> %s\n"+
			"║ ➤ <b>ᴜᴘᴛɪᴍᴇ :</b> <code>%s</code>\n"+
			"║ ➤ <b>ᴄᴘᴜ :</b> <code>%s</code>\n"+
			"║ ➤ <b>ʀᴀᴍ :</b> <code>%s</code>\n"+
			"║ ➤ <b>ᴀᴄᴛɪᴠᴇ ᴠᴄ :</b> <code>%d</code>\n"+
			"║ ➤ <b>ɢᴏ :</b> <code>%s</code>\n"+
			"╚══════════════════●",
		utils.MentionHTML(m.Client.Me()),
		uptime, cpuStr, ramStr,
		activeRooms, goVer,
	)

	// Send animated sticker first
	m.ReplyMedia(aliveSticker, &tg.MediaOptions{})

	// Then send info card with buttons
	m.Respond(caption, &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.SuppMarkup(m.ChannelID()),
	})

	return tg.ErrEndGroup
}
