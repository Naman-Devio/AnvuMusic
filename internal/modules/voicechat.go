/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/core"
	"main/internal/database"
	"main/internal/utils"
)

func handleActions(m *telegram.NewMessage) error {
	if !isValidChatType(m) {
		warnAndLeave(m.Client, m.ChannelID())
		return telegram.ErrEndGroup
	}

	if action, ok := m.Action.(*telegram.MessageActionGroupCall); ok {
		return handleVoiceChatAction(m, action)
	}

	return telegram.ErrEndGroup
}

func handleVoiceChatAction(
	m *telegram.NewMessage,
	action *telegram.MessageActionGroupCall,
) error {
	if isMaint, _ := database.IsMaintenanceEnabled(); isMaint {
		return telegram.ErrEndGroup
	}

	chatID := m.ChannelID()
	isActive := action.Duration == 0

	go clearRTMPState(chatID)
	s, err := core.GetChatState(chatID)
	if err != nil {
		gologging.ErrorF("Failed to get chat state for %d: %v", chatID, err)
		return telegram.ErrEndGroup
	}

	s.SetVoiceChatActive(isActive)

	actionStr := utils.IfElse(isActive, "started", "ended")
	gologging.DebugF("Voice chat %s in %d", actionStr, chatID)
	return telegram.ErrEndGroup
}

func isValidChatType(m *telegram.NewMessage) bool {
	return m.ChatType() != telegram.EntityChat ||
		(m.Channel != nil && m.Channel.Megagroup)
}
