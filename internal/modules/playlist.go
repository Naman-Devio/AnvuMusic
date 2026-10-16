/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"

	state "main/internal/core/models"
	"main/internal/database"
	"main/internal/locales"
	"main/internal/platforms"
)

const (
	maxUserPlaylists = 10
	maxPlaylistName  = 40
)

func init() {
	helpTexts["/createplaylist"] = `<i>Create a new personal playlist.</i>

<u>Usage:</u>
<b>/createplaylist [name]</b> — Create a playlist

<b>⚙️ Notes:</b>
• You can have up to 10 playlists
• Names are capped at 40 characters

<b>💡 Example:</b>
<code>/createplaylist Chill Vibes</code>

<b>💡 Tip:</b>
Share the returned <code>tgpl_...</code> ID with <code>/play</code> to play the whole playlist.`

	helpTexts["/deleteplaylist"] = `<i>Delete one of your playlists.</i>

<u>Usage:</u>
<b>/deleteplaylist [playlist id]</b> — Delete a playlist

<b>⚙️ Notes:</b>
• You can only delete playlists you created

<b>💡 Example:</b>
<code>/deleteplaylist tgpl_abc123</code>`

	helpTexts["/addtoplaylist"] = `<i>Add a track to one of your playlists.</i>

<u>Usage:</u>
<b>/addtoplaylist [playlist id] [song url]</b> — Add a track

<b>⚙️ Notes:</b>
• Duplicate tracks are skipped automatically

<b>💡 Example:</b>
<code>/addtoplaylist tgpl_abc123 https://youtu.be/dQw4w9WgXcQ</code>`

	helpTexts["/removefromplaylist"] = `<i>Remove a track from one of your playlists.</i>

<u>Usage:</u>
<b>/removefromplaylist [playlist id] [song number or url]</b> — Remove a track

<b>💡 Examples:</b>
<code>/removefromplaylist tgpl_abc123 2</code>
<code>/removefromplaylist tgpl_abc123 https://youtu.be/dQw4w9WgXcQ</code>`

	helpTexts["/playlistinfo"] = `<i>View the contents of a playlist.</i>

<u>Usage:</u>
<b>/playlistinfo [playlist id]</b> — Show playlist details

<b>💡 Example:</b>
<code>/playlistinfo tgpl_abc123</code>`

	helpTexts["/myplaylists"] = `<i>List all of your playlists.</i>

<u>Usage:</u>
<b>/myplaylists</b> — Show your playlists

<b>💡 Tip:</b>
Play a whole playlist with <code>/play tgpl_xxx</code>.`
}

func createPlaylistHandler(m *tg.NewMessage) error {
	userID := m.SenderID()
	args := strings.TrimSpace(m.Args())

	if args == "" {
		m.Reply(F(m.ChannelID(), "playlist_create_usage"))
		return tg.ErrEndGroup
	}

	playlists, err := database.GetUserPlaylists(userID)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_fetch_failed"))
		return tg.ErrEndGroup
	}
	if len(playlists) >= maxUserPlaylists {
		m.Reply(F(m.ChannelID(), "playlist_limit_reached"))
		return tg.ErrEndGroup
	}

	if len([]rune(args)) > maxPlaylistName {
		args = string([]rune(args)[:maxPlaylistName])
	}

	id, err := database.CreatePlaylist(args, userID)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_create_failed", locales.Arg{
			"error": err.Error(),
		}))
		return tg.ErrEndGroup
	}

	m.Reply(F(m.ChannelID(), "playlist_created", locales.Arg{
		"name": html.EscapeString(args),
		"id":   id,
	}))
	return tg.ErrEndGroup
}

func deletePlaylistHandler(m *tg.NewMessage) error {
	userID := m.SenderID()
	id := strings.TrimSpace(m.Args())

	if id == "" {
		m.Reply(F(m.ChannelID(), "playlist_delete_usage"))
		return tg.ErrEndGroup
	}

	playlist, err := database.GetPlaylist(id)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_not_found"))
		return tg.ErrEndGroup
	}
	if playlist.UserID != userID {
		m.Reply(F(m.ChannelID(), "playlist_not_owner"))
		return tg.ErrEndGroup
	}

	if err := database.DeletePlaylist(id, userID); err != nil {
		m.Reply(F(m.ChannelID(), "playlist_delete_failed", locales.Arg{
			"error": err.Error(),
		}))
		return tg.ErrEndGroup
	}

	m.Reply(F(m.ChannelID(), "playlist_deleted", locales.Arg{
		"name": html.EscapeString(playlist.Name),
	}))
	return tg.ErrEndGroup
}

func addToPlaylistHandler(m *tg.NewMessage) error {
	userID := m.SenderID()
	parts := strings.SplitN(m.Args(), " ", 2)
	if len(parts) != 2 {
		m.Reply(F(m.ChannelID(), "playlist_add_usage"))
		return tg.ErrEndGroup
	}

	id := strings.TrimSpace(parts[0])
	songURL := strings.TrimSpace(parts[1])

	playlist, err := database.GetPlaylist(id)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_not_found"))
		return tg.ErrEndGroup
	}
	if playlist.UserID != userID {
		m.Reply(F(m.ChannelID(), "playlist_not_owner"))
		return tg.ErrEndGroup
	}

	tracks, err := platforms.GetTracksFromURL(songURL, false)
	if err != nil || len(tracks) == 0 {
		m.Reply(F(m.ChannelID(), "playlist_invalid_url"))
		return tg.ErrEndGroup
	}

	track := tracks[0]
	song := database.PlaylistSong{
		URL:      track.URL,
		Name:     track.Title,
		TrackID:  track.ID,
		Duration: track.Duration,
		Platform: string(track.Source),
	}

	if err := database.AddSongToPlaylist(id, song); err != nil {
		m.Reply(F(m.ChannelID(), "playlist_add_failed", locales.Arg{
			"error": err.Error(),
		}))
		return tg.ErrEndGroup
	}

	m.Reply(F(m.ChannelID(), "playlist_song_added", locales.Arg{
		"name": html.EscapeString(song.Name),
		"pl":   html.EscapeString(playlist.Name),
	}))
	return tg.ErrEndGroup
}

func removeFromPlaylistHandler(m *tg.NewMessage) error {
	userID := m.SenderID()
	parts := strings.SplitN(m.Args(), " ", 2)
	if len(parts) != 2 {
		m.Reply(F(m.ChannelID(), "playlist_remove_usage"))
		return tg.ErrEndGroup
	}

	id := strings.TrimSpace(parts[0])
	identifier := strings.TrimSpace(parts[1])

	playlist, err := database.GetPlaylist(id)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_not_found"))
		return tg.ErrEndGroup
	}
	if playlist.UserID != userID {
		m.Reply(F(m.ChannelID(), "playlist_not_owner"))
		return tg.ErrEndGroup
	}

	var trackID string
	if idx, err := strconv.Atoi(identifier); err == nil {
		if idx < 1 || idx > len(playlist.Songs) {
			m.Reply(F(m.ChannelID(), "playlist_invalid_index"))
			return tg.ErrEndGroup
		}
		trackID = playlist.Songs[idx-1].TrackID
	} else {
		for _, song := range playlist.Songs {
			if song.URL == identifier || song.TrackID == identifier {
				trackID = song.TrackID
				break
			}
		}
	}

	if trackID == "" {
		m.Reply(F(m.ChannelID(), "playlist_song_not_found"))
		return tg.ErrEndGroup
	}

	if err := database.RemoveSongFromPlaylist(id, trackID); err != nil {
		m.Reply(F(m.ChannelID(), "playlist_remove_failed", locales.Arg{
			"error": err.Error(),
		}))
		return tg.ErrEndGroup
	}

	m.Reply(F(m.ChannelID(), "playlist_song_removed", locales.Arg{
		"name": html.EscapeString(playlist.Name),
	}))
	return tg.ErrEndGroup
}

func playlistInfoHandler(m *tg.NewMessage) error {
	id := strings.TrimSpace(m.Args())

	if id == "" {
		m.Reply(F(m.ChannelID(), "playlist_info_usage"))
		return tg.ErrEndGroup
	}

	playlist, err := database.GetPlaylist(id)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_not_found"))
		return tg.ErrEndGroup
	}

	var b strings.Builder
	for i, song := range playlist.Songs {
		if i >= 15 {
			b.WriteString(F(m.ChannelID(), "playlist_more_songs", locales.Arg{
				"count": len(playlist.Songs) - 15,
			}))
			break
		}
		b.WriteString(fmt.Sprintf(
			"%d. <a href=\"%s\">%s</a> (%s)\n",
			i+1,
			html.EscapeString(song.URL),
			html.EscapeString(song.Name),
			formatDuration(song.Duration),
		))
	}

	owner := fmt.Sprintf("<a href=\"tg://user?id=%d\">%d</a>", playlist.UserID, playlist.UserID)

	m.Reply(F(m.ChannelID(), "playlist_info", locales.Arg{
		"name":  html.EscapeString(playlist.Name),
		"owner": owner,
		"count": len(playlist.Songs),
		"songs": b.String(),
	}), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

func myPlaylistsHandler(m *tg.NewMessage) error {
	userID := m.SenderID()

	playlists, err := database.GetUserPlaylists(userID)
	if err != nil {
		m.Reply(F(m.ChannelID(), "playlist_fetch_failed"))
		return tg.ErrEndGroup
	}
	if len(playlists) == 0 {
		m.Reply(F(m.ChannelID(), "playlist_none"))
		return tg.ErrEndGroup
	}

	var b strings.Builder
	for _, pl := range playlists {
		b.WriteString(fmt.Sprintf(
			"• <b>%s</b> (<code>%s</code>) — %d songs\n",
			html.EscapeString(pl.Name),
			pl.ID,
			len(pl.Songs),
		))
	}

	m.Reply(F(m.ChannelID(), "playlist_my", locales.Arg{
		"list": b.String(),
	}), &tg.SendOptions{ParseMode: "HTML"})
	return tg.ErrEndGroup
}

// loadPlaylistTracks converts a stored playlist into playable tracks.
func loadPlaylistTracks(id string, video bool) ([]*state.Track, error) {
	playlist, err := database.GetPlaylist(id)
	if err != nil {
		return nil, err
	}
	if len(playlist.Songs) == 0 {
		return nil, errors.New("playlist is empty")
	}

	tracks := make([]*state.Track, 0, len(playlist.Songs))
	for _, song := range playlist.Songs {
		tracks = append(tracks, &state.Track{
			ID:       song.TrackID,
			Title:    song.Name,
			Duration: song.Duration,
			URL:      song.URL,
			Video:    video,
			Source:   state.PlatformName(song.Platform),
		})
	}
	return tracks, nil
}
