/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

import (
	"fmt"
	"strconv"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type RTMPConfig struct {
	URL string `bson:"rtmp_url"`
	Key string `bson:"rtmp_key"`
}

type ChatSettings struct {
	ChatID             int64      `bson:"_id"`
	ChannelPlayID      int64      `bson:"cplay_id"`
	AuthUsers          []int64    `bson:"auth_users"`
	Language           string     `bson:"language"`
	RTMP               RTMPConfig `bson:"rtmp_config"`
	AssistantIndex     int        `bson:"ass_index,omitempty"`
	ThumbnailsDisabled bool       `bson:"no_thumb"`
	PlayMode           bool       `bson:"play_mode"`
	AdminMode          string     `bson:"admin_mode"`
	CmdDelete          bool       `bson:"cmd_delete"`
}

func defaultChatSettings(chatID int64) *ChatSettings {
	return &ChatSettings{
		ChatID:    chatID,
		AuthUsers: []int64{},
	}
}

func getChatSettings(chatID int64) (*ChatSettings, error) {
	cacheKey := "chat_settings_" + strconv.FormatInt(chatID, 10)
	if cached, found := dbCache.Get(cacheKey); found {
		if settings, ok := cached.(*ChatSettings); ok {
			return settings, nil
		}
	}

	ctx, cancel := ctx()
	defer cancel()

	var settings ChatSettings
	err := chatSettingsColl.FindOne(ctx, bson.M{"_id": chatID}).
		Decode(&settings)

	if err == mongo.ErrNoDocuments {
		def := defaultChatSettings(chatID)
		dbCache.Set(cacheKey, def)
		return def, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"failed to get chat settings for %d: %w",
			chatID,
			err,
		)
	}

	dbCache.Set(cacheKey, &settings)
	return &settings, nil
}

func updateChatSettings(settings *ChatSettings) error {
	cacheKey := "chat_settings_" + strconv.FormatInt(settings.ChatID, 10)

	ctx, cancel := ctx()
	defer cancel()

	_, err := chatSettingsColl.UpdateOne(
		ctx,
		bson.M{"_id": settings.ChatID},
		bson.M{"$set": settings},
		upsertOpt,
	)
	if err != nil {
		return fmt.Errorf(
			"failed to update chat settings for %d: %w",
			settings.ChatID,
			err,
		)
	}

	dbCache.Set(cacheKey, settings)
	return nil
}

func modifyChatSettings(chatID int64, fn func(*ChatSettings) bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	if fn(settings) {
		return updateChatSettings(settings)
	}

	return nil
}

// ThumbnailsDisabled returns true when thumbnails are disabled for the chat.
func ThumbnailsDisabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.ThumbnailsDisabled, nil
}

// SetThumbnailsDisabled updates the thumbnail display setting for a chat.
func SetThumbnailsDisabled(chatID int64, value bool) error {
	return modifyChatSettings(chatID, func(s *ChatSettings) bool {
		if s.ThumbnailsDisabled == value {
			return false
		}
		s.ThumbnailsDisabled = value
		return true
	})
}

// PlayMode returns whether playback is restricted to admins and authorized users.
func PlayMode(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.PlayMode, nil
}

// SetPlayMode updates the play mode setting for a chat.
func SetPlayMode(chatID int64, value bool) error {
	return modifyChatSettings(chatID, func(s *ChatSettings) bool {
		if s.PlayMode == value {
			return false
		}
		s.PlayMode = value
		return true
	})
}

// AdminMode returns the admin mode for a chat ("admins" or "everyone").
// An empty value defaults to "admins" for backwards compatibility.
func AdminMode(chatID int64) (string, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return "", err
	}
	if settings.AdminMode == "" {
		return "admins", nil
	}
	return settings.AdminMode, nil
}

// SetAdminMode updates the admin mode setting for a chat.
func SetAdminMode(chatID int64, value string) error {
	if value != "admins" && value != "everyone" {
		value = "admins"
	}
	return modifyChatSettings(chatID, func(s *ChatSettings) bool {
		if s.AdminMode == value {
			return false
		}
		s.AdminMode = value
		return true
	})
}

// CmdDelete returns whether command messages are auto-deleted in the chat.
func CmdDelete(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.CmdDelete, nil
}

// SetCmdDelete updates the command auto-delete setting for a chat.
func SetCmdDelete(chatID int64, value bool) error {
	return modifyChatSettings(chatID, func(s *ChatSettings) bool {
		if s.CmdDelete == value {
			return false
		}
		s.CmdDelete = value
		return true
	})
}
