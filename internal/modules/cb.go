/*
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package modules

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/core"
	"main/internal/database"
	"main/internal/locales"
	"main/internal/platforms"
	"main/internal/utils"
)

// Action handlers map for cleaner dispatch
type actionHandler func(*tg.CallbackQuery, *core.RoomState, int64) error

var actionHandlers = map[string]actionHandler{
	"pause":    handlePauseAction,
	"resume":   handleResumeAction,
	"replay":   handleReplayAction,
	"skip":     handleSkipAction,
	"stop":     handleStopAction,
	"mute":     handleMuteAction,
	"unmute":   handleUnmuteAction,
	"autoplay": handleAutoplayAction,
	"playlist": handleAddToPlaylistAction,
	"settings": handleSettingsAction,
	"back":     handleSettingsBackAction,
}

func cancelHandler(cb *tg.CallbackQuery) error {
	chatID := cb.ChannelID()
	opt := &tg.CallbackOptions{Alert: true}

	if !checkAdminOrAuth(cb, chatID, opt) {
		return tg.ErrEndGroup
	}

	if cancel, ok := downloadCancels[chatID]; ok {
		cancel()
		delete(downloadCancels, chatID)
		cb.Answer(F(chatID, "download_cancelled"), opt)
	} else {
		cb.Answer(F(chatID, "no_download_to_cancel"), opt)
	}
	return tg.ErrEndGroup
}

func closeHandler(cb *tg.CallbackQuery) error {
	cb.Answer("")
	cb.Delete()
	return tg.ErrEndGroup
}

func emptyCBHandler(cb *tg.CallbackQuery) error {
	cb.Answer("")
	return tg.ErrEndGroup
}

func roomHandle(cb *tg.CallbackQuery) error {
	opt := &tg.CallbackOptions{Alert: true}
	data := cb.DataString()

	// Parse action type
	action := strings.TrimPrefix(data, "croom:")
	action = strings.TrimPrefix(action, "room:")

	if action == "" {
		gologging.WarnF("Missing action in data: %s", data)
		cb.Answer(F(cb.ChannelID(), "invalid_request"), opt)
		return tg.ErrEndGroup
	}

	chatID := cb.ChannelID()

	// Handle cplay mode
	if strings.HasPrefix(cb.DataString(), "croom:") {
		realChatID, err := database.LinkedChannel(chatID)
		if err != nil {
			gologging.ErrorF(
				"Failed to get chat ID for cplay ID %d: %v",
				chatID,
				err,
			)
			cb.Answer(F(chatID, "room_not_linked"), opt)
			return tg.ErrEndGroup
		}
		chatID = realChatID
	}

	// Get room
	r, err := getRoomForCallback(chatID)
	if err != nil {
		if strings.Contains(err.Error(), "no active room") {
			cb.Answer(F(cb.ChannelID(), "room_not_active_cb"), opt)
			editMessage(cb, F(cb.ChannelID(), "room_no_active"))
		} else {
			cb.Answer(err.Error(), opt)
		}
		return tg.ErrEndGroup
	}

	// Check permissions
	if !checkAdminOrAuth(cb, chatID, opt) {
		return tg.ErrEndGroup
	}

	// Flood control
	if !checkFloodControl(cb, chatID, opt) {
		return tg.ErrEndGroup
	}

	// Handle seek actions
	if strings.HasPrefix(action, "seek") {
		return handleSeekAction(cb, r, action, opt)
	}

	// Dispatch to handler
	if handler, ok := actionHandlers[action]; ok {
		return handler(cb, r, chatID)
	}

	gologging.WarnF("Unknown callback type: %s", action)
	cb.Answer(F(cb.ChannelID(), "unknown_action"), opt)
	return tg.ErrEndGroup
}

// Helper functions

func getRoomForCallback(chatID int64) (*core.RoomState, error) {
	ass, err := core.Assistants.ForChat(chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant: %w", err)
	}

	r, ok := core.GetRoom(chatID, ass, false)
	if !ok || !r.IsActiveChat() {
		return nil, fmt.Errorf("no active room")
	}

	return r, nil
}

func checkAdminOrAuth(
	cb *tg.CallbackQuery,
	chatID int64,
	opt *tg.CallbackOptions,
) bool {
	isAdmin, err := utils.IsChatAdmin(cb.Client, chatID, cb.SenderID)
	if err != nil || !isAdmin {
		cb.Answer(F(cb.ChannelID(), "only_admin_or_auth_cb"), opt)
		return false
	}
	return true
}

func checkFloodControl(
	cb *tg.CallbackQuery,
	chatID int64,
	opt *tg.CallbackOptions,
) bool {
	key := fmt.Sprintf("room:%d:%d", cb.Sender.ID, chatID)
	if remaining := utils.GetFlood(key); remaining > 0 {
		cb.Answer(F(cb.ChannelID(), "flood_seconds", locales.Arg{
			"duration": int(remaining.Seconds()),
		}), opt)
		return false
	}
	utils.SetFlood(key, 5*time.Second)
	return true
}

func editMessage(cb *tg.CallbackQuery, text string) {
	if _, err := cb.Edit(text); err != nil {
		gologging.ErrorF("Edit error: %v", err)
	}
}

func replyToCallback(cb *tg.CallbackQuery, text string) {
	msg, err := cb.GetMessage()
	if err != nil {
		return
	}
	msg.Reply(text)
}

// Action handlers

func handlePauseAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	gologging.InfoF("Callback → pause, chatID=%d", chatID)

	if r.IsPaused() {
		remaining := r.RemainingResumeDuration()
		msg := utils.IfElse(
			remaining > 0,
			F(cb.ChannelID(), "room_already_paused_auto", locales.Arg{
				"duration": formatDuration(int(remaining.Seconds())),
			}),
			F(cb.ChannelID(), "room_already_paused"),
		)
		cb.Answer(msg, opt)
		return tg.ErrEndGroup
	}

	if _, err := r.Pause(); err != nil {
		gologging.ErrorF("Pause failed: %v", err)
		cb.Answer(F(cb.ChannelID(), "room_pause_failed", locales.Arg{
			"error": err.Error(),
		}), opt)
		return tg.ErrEndGroup
	}

	if r.IsMuted() {
		r.Unmute()
	}

	cb.Answer(F(cb.ChannelID(), "cb_pause_success", locales.Arg{
		"position": formatDuration(r.Position()),
	}), opt)

	updatePlaybackMessage(cb, r, "paused")
	return tg.ErrEndGroup
}

func handleResumeAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	gologging.InfoF("Callback → resume, chatID=%d", chatID)

	if !r.IsPaused() {
		cb.Answer(F(cb.ChannelID(), "cb_already_playing"), opt)
		return tg.ErrEndGroup
	}

	if _, err := r.Resume(); err != nil {
		gologging.ErrorF("Resume failed: %v", err)
		cb.Answer(F(cb.ChannelID(), "cb_resume_failed"), opt)
		return tg.ErrEndGroup
	}

	cb.Answer(F(cb.ChannelID(), "cb_resume_success", locales.Arg{
		"position": formatDuration(r.Position()),
	}), opt)

	updatePlaybackMessage(cb, r, "playing")
	return tg.ErrEndGroup
}

func handleReplayAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	gologging.InfoF("Callback → replay, chatID=%d", chatID)

	statusMsg, err := cb.Respond(F(cb.ChannelID(), "cb_replaying"))
	if err != nil {
		gologging.ErrorF("Failed to send replay message: %v", err)
		return tg.ErrEndGroup
	}

	if err := r.Replay(); err != nil {
		gologging.ErrorF("Replay failed: %v", err)
		utils.EOR(statusMsg, F(cb.ChannelID(), "replay_failed", locales.Arg{
			"error": err.Error(),
		}))
		cb.Answer(F(cb.ChannelID(), "cb_replay_failed"), opt)
		return tg.ErrEndGroup
	}

	track := r.Track()
	trackTitle := html.EscapeString(utils.ShortTitle(track.Title, 25))

	msgText := F(cb.ChannelID(), "stream_now_playing", locales.Arg{
		"url":        track.URL,
		"title":      trackTitle,
		"duration":   formatDuration(track.Duration),
		"by":         track.Requester,
		"credit_url": config.SupportChannel,
	})

	cb.Answer(F(cb.ChannelID(), "cb_replay_success"), opt)

	optSend := &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetPlayMarkup(cb.ChannelID(), r, false),
	}
	if media := resolveTrackMedia(cb.ChannelID(), track); media != "" {
		optSend.Media = media
	}

	statusMsg, _ = utils.EOR(statusMsg, msgText, optSend)
	r.SetStatusMsg(statusMsg)

	editMessage(cb, F(cb.ChannelID(), "cb_replay_edited", locales.Arg{
		"user": utils.MentionHTML(cb.Sender),
	}))
	return tg.ErrEndGroup
}

func handleSkipAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	gologging.InfoF("Callback → skip, chatID=%d", chatID)

	if len(r.Queue()) == 0 {
		cleanupRoomMessages(r)
		core.DeleteRoom(r.ChatID())
		editMessage(cb, F(cb.ChannelID(), "skip_stopped", locales.Arg{
			"user": utils.MentionHTML(cb.Sender),
		}))
		cb.Answer(F(cb.ChannelID(), "cb_skip_queue_empty"), opt)
		return tg.ErrEndGroup
	}
	r.SetLoop(0)
	t := r.NextTrack()

	statusMsg, err := cb.Respond(F(cb.ChannelID(), "stream_downloading_next"))
	if err != nil {
		gologging.ErrorF("Failed to send message: %v", err)
	}

	path, err := platforms.Download(context.Background(), t, statusMsg)
	if err != nil {
		gologging.ErrorF("Download failed for %s: %v", t.URL, err)
		utils.EOR(
			statusMsg,
			F(cb.ChannelID(), "stream_download_fail", locales.Arg{
				"error": err.Error(),
			}),
		)
		cb.Answer(F(cb.ChannelID(), "cb_skip_download_failed"), opt)
		cleanupRoomMessages(r)
		core.DeleteRoom(r.ChatID())

		return tg.ErrEndGroup
	}

	if err := r.Play(t, path); err != nil {
		gologging.ErrorF("Play error: %v", err)
		utils.EOR(statusMsg, F(cb.ChannelID(), "stream_play_fail"))
		cb.Answer(F(cb.ChannelID(), "cb_skip_play_failed"), opt)
		cleanupRoomMessages(r)
		core.DeleteRoom(r.ChatID())

		return tg.ErrEndGroup
	}

	cb.Answer(F(cb.ChannelID(), "cb_skip_success"), opt)
	cb.Delete()

	title := utils.ShortTitle(t.Title, 25)
	safeTitle := html.EscapeString(title)

	msgText := F(cb.ChannelID(), "stream_now_playing", locales.Arg{
		"url":        t.URL,
		"title":      safeTitle,
		"duration":   formatDuration(t.Duration),
		"by":         t.Requester,
		"credit_url": config.SupportChannel,
	})

	sendOpt := &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetPlayMarkup(cb.ChannelID(), r, false),
	}
	if media := resolveTrackMedia(cb.ChannelID(), t); media != "" {
		sendOpt.Media = media
	}

	statusMsg, _ = utils.EOR(statusMsg, msgText, sendOpt)
	replyToCallback(cb, F(cb.ChannelID(), "cb_skip_edited", locales.Arg{
		"user": utils.MentionHTML(cb.Sender),
	}))

	r.SetStatusMsg(statusMsg)
	return tg.ErrEndGroup
}

func handleStopAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	gologging.InfoF("Callback → stop, chatID=%d", chatID)

	cleanupRoomMessages(r)
	core.DeleteRoom(r.ChatID())

	cb.Answer(F(cb.ChannelID(), "cb_stop_success"), opt)
	editMessage(cb, F(cb.ChannelID(), "stopped", locales.Arg{
		"user": utils.MentionHTML(cb.Sender),
	}))

	return tg.ErrEndGroup
}

func handleMuteAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	if r.IsMuted() {
		remaining := r.RemainingUnmuteDuration()
		msg := utils.IfElse(
			remaining > 0,
			F(cb.ChannelID(), "mute_already_muted_with_time", locales.Arg{
				"duration": formatDuration(int(remaining.Seconds())),
			}),
			F(cb.ChannelID(), "mute_already_muted"),
		)
		cb.Answer(msg, opt)
		return tg.ErrEndGroup
	}

	if _, err := r.Mute(); err != nil {
		cb.Answer(F(cb.ChannelID(), "mute_failed", locales.Arg{
			"error": err.Error(),
		}), opt)
		return tg.ErrEndGroup
	}

	cb.Answer(F(cb.ChannelID(), "cb_mute_success"), opt)
	updatePlaybackMessage(cb, r, "muted")
	return tg.ErrEndGroup
}

func handleUnmuteAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	if !r.IsMuted() {
		cb.Answer(F(cb.ChannelID(), "unmute_already"), opt)
		return tg.ErrEndGroup
	}

	if _, err := r.Unmute(); err != nil {
		cb.Answer(F(cb.ChannelID(), "unmute_failed", locales.Arg{
			"error": err.Error(),
		}), opt)
		return tg.ErrEndGroup
	}

	cb.Answer(F(cb.ChannelID(), "cb_unmute_success"), opt)
	updatePlaybackMessage(cb, r, "playing")
	return tg.ErrEndGroup
}

// renderPlaybackPanel re-renders the now-playing panel, keeping the ⚙️ settings
// view open while its countdown is running and falling back to the normal
// control panel otherwise. Used by callbacks, the countdown ticker and the
// room monitor so the two views never fight each other.
func renderPlaybackPanel(r *core.RoomState) {
	if r == nil || r.IsDestroyed() {
		return
	}
	msg := r.StatusMsg()
	if msg == nil {
		return
	}

	var markup tg.ReplyMarkup
	if r.InSettingsView() {
		markup = core.GetPlaybackSettingsMarkup(r.EffectiveChatID(), r)
	} else {
		markup = core.GetPlayMarkup(r.EffectiveChatID(), r, false)
	}

	if _, err := msg.Edit(msg.Text(), &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	}); err != nil {
		gologging.ErrorF("Edit error: %v", err)
	}
}

// refreshSettingsView resets the ⚙️ settings view window back to 5 seconds and
// schedules its auto-close. Used when an action is taken while the view is open
// (autoplay toggle, playlist picker) so the user can keep using it.
func refreshSettingsView(r *core.RoomState) {
	if r == nil || !r.InSettingsView() {
		return
	}
	until := time.Now().Add(core.SettingsViewWindow).Unix()
	r.SetData("settings_until", until)
	scheduleSettingsClose(r, until)
}

// scheduleSettingsClose restores the normal control panel with a single delayed
// edit once the ⚙️ settings view window expires. No per-second edits, so no
// flood risk. A newer ⚙️ tap (newer settings_until) makes older timers exit
// quietly.
func scheduleSettingsClose(r *core.RoomState, until int64) {
	time.AfterFunc(core.SettingsViewWindow, func() {
		if r.IsDestroyed() {
			return
		}
		ok, v := r.GetData("settings_until")
		if !ok {
			return
		}
		u, isInt := v.(int64)
		if !isInt || u != until {
			return // superseded by a newer ⚙️ tap
		}
		r.DeleteData("settings_until")
		renderPlaybackPanel(r)
	})
}

// handleSettingsAction opens the ⚙️ settings view on the now-playing panel,
// replacing the control rows with autoplay/playlist + back.
func handleSettingsAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	until := time.Now().Add(core.SettingsViewWindow).Unix()
	r.SetData("settings_until", until)

	cb.Answer(F(cb.ChannelID(), "cb_settings_extras_shown"), opt)

	renderPlaybackPanel(r)
	scheduleSettingsClose(r, until)

	return tg.ErrEndGroup
}

// handleSettingsBackAction closes the ⚙️ settings view and returns the panel to
// the normal control rows.
func handleSettingsBackAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	r.DeleteData("settings_until")
	cb.Answer(F(cb.ChannelID(), "cb_settings_back"), opt)

	renderPlaybackPanel(r)

	return tg.ErrEndGroup
}

func handleAutoplayAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	next := !r.Autoplay()
	r.SetAutoplay(next)

	if next {
		cb.Answer(F(cb.ChannelID(), "cb_autoplay_enabled"), opt)
	} else {
		cb.Answer(F(cb.ChannelID(), "cb_autoplay_disabled"), opt)
	}

	// Refresh the panel so the button reflects the new state. If the ⚙️ settings
	// view is open, keep it open with a reset countdown so the user can toggle
	// both actions.
	refreshSettingsView(r)
	renderPlaybackPanel(r)

	return tg.ErrEndGroup
}

// playlistPickerRooms remembers which room chat a user's playlist picker was
// opened from (userID → room chatID). This keeps the picker working for
// channel-play setups, where the panel lives in the group but the room is
// keyed by the linked channel.
var playlistPickerRooms = make(map[int64]int64)

// handleAddToPlaylistAction opens a playlist picker so the caller can choose
// which playlist to save the currently playing track into.
func handleAddToPlaylistAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	chatID int64,
) error {
	opt := &tg.CallbackOptions{Alert: true}

	track := r.Track()
	if track == nil || track.ID == "" {
		cb.Answer(F(cb.ChannelID(), "playlist_nothing_to_add"), opt)
		return tg.ErrEndGroup
	}

	playlists, err := database.GetUserPlaylists(cb.SenderID)
	if err != nil {
		cb.Answer(F(cb.ChannelID(), "playlist_fetch_failed"), opt)
		return tg.ErrEndGroup
	}

	playlistPickerRooms[cb.SenderID] = chatID

	// Keep the ⚙️ settings view open with a reset countdown while the user picks
	// a playlist, so the panel is still in settings view when they come back.
	refreshSettingsView(r)

	title := html.EscapeString(utils.ShortTitle(track.Title, 40))
	key := "playlist_picker_title"
	if len(playlists) == 0 {
		key = "playlist_picker_empty"
	}

	if _, err := cb.Respond(F(cb.ChannelID(), key, locales.Arg{
		"title": title,
	}), &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetPlaylistPickerMarkup(cb.ChannelID(), playlists),
	}); err != nil {
		gologging.ErrorF("Failed to send playlist picker: %v", err)
		return tg.ErrEndGroup
	}

	cb.Answer(F(cb.ChannelID(), "playlist_picker_open"), opt)
	return tg.ErrEndGroup
}

// playlistPickCallback saves the currently playing track into the playlist
// chosen from the picker (or a newly created one via "plist:create").
func playlistPickCallback(cb *tg.CallbackQuery) error {
	opt := &tg.CallbackOptions{Alert: true}
	chatID := cb.ChannelID()
	userID := cb.SenderID

	roomChatID, ok := playlistPickerRooms[userID]
	delete(playlistPickerRooms, userID)
	if !ok {
		roomChatID = chatID
	}

	r, err := getRoomForCallback(roomChatID)
	if err != nil {
		cb.Answer(F(chatID, "room_not_active_cb"), opt)
		return tg.ErrEndGroup
	}

	track := r.Track()
	if track == nil || track.ID == "" {
		cb.Answer(F(chatID, "playlist_nothing_to_add"), opt)
		_, _ = cb.Delete()
		return tg.ErrEndGroup
	}

	data := strings.TrimPrefix(cb.DataString(), "plist:")

	var playlistID string
	if data == "create" {
		playlists, err := database.GetUserPlaylists(userID)
		if err != nil {
			cb.Answer(F(chatID, "playlist_fetch_failed"), opt)
			return tg.ErrEndGroup
		}
		if len(playlists) >= maxUserPlaylists {
			cb.Answer(F(chatID, "playlist_limit_reached"), opt)
			return tg.ErrEndGroup
		}

		playlistID, err = database.CreatePlaylist("My Playlist", userID)
		if err != nil {
			cb.Answer(F(chatID, "playlist_create_failed", locales.Arg{
				"error": err.Error(),
			}), opt)
			return tg.ErrEndGroup
		}
	} else {
		playlist, err := database.GetPlaylist(data)
		if err != nil {
			cb.Answer(F(chatID, "playlist_not_found"), opt)
			_, _ = cb.Delete()
			return tg.ErrEndGroup
		}
		if playlist.UserID != userID {
			cb.Answer(F(chatID, "playlist_not_owner"), opt)
			return tg.ErrEndGroup
		}
		playlistID = data
	}

	song := database.PlaylistSong{
		URL:      track.URL,
		Name:     track.Title,
		TrackID:  track.ID,
		Duration: track.Duration,
		Platform: string(track.Source),
	}

	if err := database.AddSongToPlaylist(playlistID, song); err != nil {
		cb.Answer(F(chatID, "playlist_add_failed", locales.Arg{
			"error": err.Error(),
		}), opt)
		return tg.ErrEndGroup
	}

	playlist, err := database.GetPlaylist(playlistID)
	if err == nil {
		cb.Answer(F(chatID, "playlist_added_to", locales.Arg{
			"name": html.EscapeString(song.Name),
			"pl":   html.EscapeString(playlist.Name),
		}), opt)
	}

	_, _ = cb.Delete()
	return tg.ErrEndGroup
}

func handleSeekAction(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	action string,
	opt *tg.CallbackOptions,
) error {
	parts := strings.Split(action, "_")
	if len(parts) != 2 {
		cb.Answer(F(cb.ChannelID(), "invalid_request"), opt)
		return tg.ErrEndGroup
	}

	seconds, err := strconv.Atoi(parts[1])
	if err != nil {
		cb.Answer(F(cb.ChannelID(), "invalid_request"), opt)
		return tg.ErrEndGroup
	}

	isBackward := strings.HasPrefix(action, "seekback_")

	if isBackward {
		if r.Position() <= seconds {
			r.Seek(-int(r.Position()))
		} else {
			r.Seek(-seconds)
		}
		cb.Answer(F(cb.ChannelID(), "cb_seekback_success", locales.Arg{
			"seconds": seconds,
		}), opt)
		replyToCallback(cb, F(cb.ChannelID(), "cb_seekback_edited", locales.Arg{
			"seconds": seconds,
			"user":    utils.MentionHTML(cb.Sender),
		}))
	} else {
		if (r.Track().Duration - r.Position()) <= seconds {
			cb.Answer(F(cb.ChannelID(), "cb_seek_near_end", locales.Arg{
				"seconds": seconds,
			}), opt)
			return tg.ErrEndGroup
		}
		r.Seek(seconds)
		cb.Answer(F(cb.ChannelID(), "cb_seek_success", locales.Arg{
			"seconds": seconds,
		}), opt)
		replyToCallback(cb, F(cb.ChannelID(), "cb_seek_edited", locales.Arg{
			"seconds": seconds,
			"user":    utils.MentionHTML(cb.Sender),
		}))
	}

	return tg.ErrEndGroup
}

func updatePlaybackMessage(
	cb *tg.CallbackQuery,
	r *core.RoomState,
	state string,
) {
	track := r.Track()

	if track == nil {
		return
	}
	safeTitle := html.EscapeString(utils.ShortTitle(track.Title, 25))
	mention := utils.MentionHTML(cb.Sender)

	var msgText string
	switch state {
	case "paused":
		msgText = F(cb.ChannelID(), "cb_pause_message", locales.Arg{
			"url":      track.URL,
			"title":    safeTitle,
			"position": formatDuration(r.Position()),
			"duration": formatDuration(track.Duration),
			"user":     mention,
		})
	case "playing":
		msgText = F(cb.ChannelID(), "cb_resume_message", locales.Arg{
			"url":      track.URL,
			"title":    safeTitle,
			"duration": formatDuration(track.Duration),
			"user":     mention,
		})
	case "muted":
		msgText = F(cb.ChannelID(), "cb_mute_message", locales.Arg{
			"url":   track.URL,
			"title": safeTitle,
			"user":  mention,
		})
	}

	editMessage(cb, msgText)

	if _, err := cb.Edit(msgText, &tg.SendOptions{
		ParseMode:   "HTML",
		ReplyMarkup: core.GetPlayMarkup(cb.ChannelID(), r, false),
	}); err != nil {
		gologging.ErrorF("Edit error: %v", err)
	}
}
