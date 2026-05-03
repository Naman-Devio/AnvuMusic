/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

func AutoLeave() (bool, error) {
	state, err := getBotState()
	if err != nil {
		return false, err
	}
	return state.AutoLeave, nil
}

func SetAutoLeave(value bool) error {
	return modifyBotState(func(s *BotState) bool {
		if s.AutoLeave == value {
			return false
		}
		s.AutoLeave = value
		return true
	})
}
