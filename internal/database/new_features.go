/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ─────────────────────────────────────────
//  Welcome Settings
// ─────────────────────────────────────────

type welcomeDoc struct {
	ChatID  int64 `bson:"chat_id"`
	Enabled bool  `bson:"enabled"`
}

func SetWelcome(chatID int64, enabled bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("welcome_settings")
	filter := bson.M{"chat_id": chatID}
	update := bson.M{"$set": bson.M{"enabled": enabled}}
	_, err := col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func GetWelcome(chatID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("welcome_settings")
	var doc welcomeDoc
	err := col.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	return doc.Enabled, err
}

// ─────────────────────────────────────────
//  Keyword Filters
// ─────────────────────────────────────────

type filterDoc struct {
	ChatID   int64  `bson:"chat_id"`
	Keyword  string `bson:"keyword"`
	Response string `bson:"response"`
}

func AddFilter(chatID int64, keyword, response string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("keyword_filters")
	filter := bson.M{"chat_id": chatID, "keyword": keyword}
	update := bson.M{"$set": bson.M{"response": response}}
	_, err := col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func RemoveFilter(chatID int64, keyword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("keyword_filters")
	_, err := col.DeleteOne(ctx, bson.M{"chat_id": chatID, "keyword": keyword})
	return err
}

// GetFilters returns list of keywords only (for listing)
func GetFilters(chatID int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("keyword_filters")
	cursor, err := col.Find(ctx, bson.M{"chat_id": chatID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var keywords []string
	for cursor.Next(ctx) {
		var doc filterDoc
		if err := cursor.Decode(&doc); err == nil {
			keywords = append(keywords, doc.Keyword)
		}
	}
	return keywords, nil
}

// MatchFilter returns response if any keyword matches msg text
func MatchFilter(chatID int64, text string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("keyword_filters")
	cursor, err := col.Find(ctx, bson.M{"chat_id": chatID})
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc filterDoc
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if strings.Contains(text, doc.Keyword) {
			return doc.Response, nil
		}
	}
	return "", nil
}

// ─────────────────────────────────────────
//  Global Ban
// ─────────────────────────────────────────

type GBanEntry struct {
	UserID int64  `bson:"user_id"`
	Reason string `bson:"reason"`
}

func GBanUser(userID int64, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("gbans")
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": bson.M{"reason": reason}}
	_, err := col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

func UnGBanUser(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("gbans")
	_, err := col.DeleteOne(ctx, bson.M{"user_id": userID})
	return err
}

func IsGBanned(userID int64) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("gbans")
	var entry GBanEntry
	err := col.FindOne(ctx, bson.M{"user_id": userID}).Decode(&entry)
	if err == mongo.ErrNoDocuments {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, entry.Reason, nil
}

func GetGBans() ([]GBanEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	col := db.Collection("gbans")
	cursor, err := col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var entries []GBanEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
