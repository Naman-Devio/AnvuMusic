/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package utils

import (
	"fmt"
	"html"

	tg "github.com/amarnathcjd/gogram/telegram"
)

// FullName returns trimmed first+last name of a user
func FullName(u *tg.UserObj) string {
	if u == nil {
		return "Unknown"
	}
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	if name == "" {
		return "User"
	}
	return name
}

// MentionHTMLFromUser creates a mention link from a UserObj
func MentionHTMLFromUser(u *tg.UserObj) string {
	return MentionHTML(u)
}

// HtmlEscape escapes a string for HTML output
func HtmlEscape(s string) string {
	return html.EscapeString(s)
}

// ExtractUserObj returns the *tg.UserObj from a reply or user-id arg.
// It wraps the existing ExtractUser (which returns int64) and fetches full info.
func ExtractUserObj(m *tg.NewMessage) (*tg.UserObj, error) {
	if m == nil {
		return nil, fmt.Errorf("nil message")
	}

	// If it's a reply, grab the sender of the replied message directly
	if m.IsReply() {
		replied, err := m.GetReplyMessage()
		if err != nil {
			return nil, fmt.Errorf("could not fetch replied message: %w", err)
		}
		if replied != nil && replied.Sender != nil {
			return replied.Sender, nil
		}
	}

	// Fall back to resolving from int64 ID
	userID, err := ExtractUser(m)
	if err != nil {
		return nil, err
	}

	peer, err := m.Client.ResolvePeer(userID)
	if err != nil {
		return nil, fmt.Errorf("could not resolve user %d: %w", userID, err)
	}

	inputUser, ok := peer.(*tg.InputPeerUser)
	if !ok {
		return nil, fmt.Errorf("resolved peer is not a user")
	}

	users, err := m.Client.UsersGetUsers([]tg.InputUser{
		&tg.InputUserObj{UserID: inputUser.UserID, AccessHash: inputUser.AccessHash},
	})
	if err != nil || len(users) == 0 {
		return nil, fmt.Errorf("could not fetch user info")
	}

	u, ok := users[0].(*tg.UserObj)
	if !ok {
		return nil, fmt.Errorf("unexpected user type")
	}
	return u, nil
}
