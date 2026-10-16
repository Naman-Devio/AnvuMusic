/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// PlaylistSong represents a single track stored inside a playlist.
type PlaylistSong struct {
	URL      string `bson:"url"`
	Name     string `bson:"name"`
	TrackID  string `bson:"track_id"`
	Duration int    `bson:"duration"`
	Platform string `bson:"platform"`
}

// Playlist represents a user's personal playlist.
type Playlist struct {
	ID     string         `bson:"_id"`
	Name   string         `bson:"name"`
	UserID int64          `bson:"user_id"`
	Songs  []PlaylistSong `bson:"songs"`
}

// generatePlaylistID generates a unique playlist ID (tgpl_ + random hex).
func generatePlaylistID() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return "tgpl_" + hex.EncodeToString(b)
}

// CreatePlaylist creates a new playlist for a user and returns its ID.
func CreatePlaylist(name string, userID int64) (string, error) {
	id := generatePlaylistID()
	playlist := Playlist{
		ID:     id,
		Name:   name,
		UserID: userID,
		Songs:  []PlaylistSong{},
	}

	ctx, cancel := ctx()
	defer cancel()

	if _, err := playlistsColl.InsertOne(ctx, playlist); err != nil {
		return "", err
	}
	return id, nil
}

// GetPlaylist fetches a playlist by its ID.
func GetPlaylist(id string) (*Playlist, error) {
	ctx, cancel := ctx()
	defer cancel()

	var playlist Playlist
	if err := playlistsColl.FindOne(ctx, bson.M{"_id": id}).Decode(&playlist); err != nil {
		return nil, err
	}
	return &playlist, nil
}

// DeletePlaylist deletes a playlist, but only if it belongs to the given user.
func DeletePlaylist(id string, userID int64) error {
	ctx, cancel := ctx()
	defer cancel()

	res, err := playlistsColl.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// playlistHasSong reports whether a track ID already exists in the playlist.
func playlistHasSong(id, trackID string) bool {
	playlist, err := GetPlaylist(id)
	if err != nil {
		return false
	}
	for _, song := range playlist.Songs {
		if song.TrackID == trackID {
			return true
		}
	}
	return false
}

// AddSongToPlaylist adds a song to a playlist, skipping duplicates.
func AddSongToPlaylist(id string, song PlaylistSong) error {
	if song.TrackID == "" || playlistHasSong(id, song.TrackID) {
		return nil
	}

	ctx, cancel := ctx()
	defer cancel()

	_, err := playlistsColl.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$push": bson.M{"songs": song}},
	)
	return err
}

// RemoveSongFromPlaylist removes a song from a playlist by its track ID.
func RemoveSongFromPlaylist(id, trackID string) error {
	ctx, cancel := ctx()
	defer cancel()

	res, err := playlistsColl.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$pull": bson.M{"songs": bson.M{"track_id": trackID}}},
	)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("track not found in playlist")
	}
	return nil
}

// GetUserPlaylists returns all playlists belonging to a user.
func GetUserPlaylists(userID int64) ([]Playlist, error) {
	ctx, cancel := ctx()
	defer cancel()

	cursor, err := playlistsColl.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var playlists []Playlist
	if err := cursor.All(ctx, &playlists); err != nil {
		return nil, err
	}
	return playlists, nil
}
