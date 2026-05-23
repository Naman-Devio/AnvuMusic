/*
 * ○ Thumbnail toggle handler
 */

package modules

import (
    "strings"

    "github.com/Laky-64/gologging"
    tg "github.com/amarnathcjd/gogram/telegram"

    "main/internal/database"
    "main/internal/locales"
    "main/internal/utils"
)

func init() {
    helpTexts["nothumb"] = `<i>Enable or disable thumbnail/artwork display in playback messages.</i>

Usage:
<b>/nothumb</b> — Show current status
<b>/nothumb enable</b> — Disable thumbnails
<b>/nothumb disable</b> — Enable thumbnails`
}

func nothumbHandler(m *tg.NewMessage) error {
    args := strings.Fields(m.Text())
    chatID := m.ChannelID()

    current, err := database.ThumbnailsDisabled(chatID)
    if err != nil {
        m.Reply(F(chatID, "nothumb_fetch_fail"))
        return tg.ErrEndGroup
    }

    status := F(chatID, utils.IfElse(current, "disabled", "enabled"))

    if len(args) < 2 {
        m.Reply(F(chatID, "nothumb_status", locales.Arg{"action": status}))
        return tg.ErrEndGroup
    }

    newState, parseErr := utils.ParseBool(args[1])
    if parseErr != nil {
        m.Reply(F(chatID, "invalid_bool"))
        return tg.ErrEndGroup
    }

    if newState == current {
        m.Reply(F(chatID, "nothumb_already", locales.Arg{"action": status}))
        return tg.ErrEndGroup
    }

    if err := database.SetThumbnailsDisabled(chatID, newState); err != nil {
        gologging.ErrorF("Failed to set thumbnails: %v", err)
        m.Reply(F(chatID, "nothumb_update_fail"))
        return tg.ErrEndGroup
    }

    newStatus := F(chatID, utils.IfElse(newState, "disabled", "enabled"))
    m.Reply(F(chatID, "nothumb_updated", locales.Arg{"action": newStatus}))
    return tg.ErrEndGroup
}
