/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

func IsLoggerEnabled() (bool, error) {
	state, err := getBotState()
	if err != nil {
		return false, err
	}
	return state.LoggerEnabled, nil
}

func SetLoggerEnabled(enabled bool) error {
	return modifyBotState(func(s *BotState) bool {
		if s.LoggerEnabled == enabled {
			return false
		}
		s.LoggerEnabled = enabled
		return true
	})
}
