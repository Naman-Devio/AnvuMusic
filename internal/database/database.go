/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

import (
	"context"
	"time"

	"github.com/Laky-64/gologging"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"main/internal/utils"
)

var (
	client           *mongo.Client
	database         *mongo.Database
	settingsColl     *mongo.Collection
	chatSettingsColl *mongo.Collection

	logger  = gologging.GetLogger("Database")
	dbCache = utils.NewCache[string, any](60 * time.Minute)
)

func Init(mongoURL string) (func(), error) {
	var err error
	logger.Debug("Initializing MongoDB...")
	client, err = mongo.Connect(options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, err
	}

	logger.Debug("Successfully connected to MongoDB.")

	// If you already use a live DB named "ArcMusic", point this string to that name or run a one-time rename so data is not left behind.
	database = client.Database("AnvuMusic")
	settingsColl = database.Collection("bot_settings")
	chatSettingsColl = database.Collection("chat_settings")

	migrateData()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(ctx); err != nil {
			logger.Error("Error while disconnecting MongoDB: %v", err)
		} else {
			logger.Info("MongoDB disconnected successfully")
		}
	}, nil
}
