/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package platforms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	state "main/internal/core/models"
)

const (
	PlatformShrutiApi state.PlatformName = "ShrutiApi"
	shrutiBaseURL                        = "https://shrutibots.site"
)

type ShrutiApiPlatform struct {
	name   state.PlatformName
	client *http.Client
}

func init() {
	Register(85, &ShrutiApiPlatform{
		name:   PlatformShrutiApi,
		client: &http.Client{Timeout: 90 * time.Second},
	})
}

func (s *ShrutiApiPlatform) Name() state.PlatformName { return s.name }

func (s *ShrutiApiPlatform) CanGetTracks(_ string) bool { return false }

func (s *ShrutiApiPlatform) GetTracks(_ string, _ bool) ([]*state.Track, error) {
	return nil, errors.New("shrutiapi is a download-only platform")
}

func (s *ShrutiApiPlatform) CanDownload(source state.PlatformName) bool {
	return source == PlatformYouTube
}

func (s *ShrutiApiPlatform) CanSearch() bool { return false }

func (s *ShrutiApiPlatform) Search(_ string, _ bool) ([]*state.Track, error) { return nil, nil }

func (s *ShrutiApiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	_ *telegram.NewMessage,
) (string, error) {
	// Check local cache first
	if f := findFile(track); f != "" {
		gologging.Debug("ShrutiApi: cache hit -> " + f)
		return f, nil
	}

	mediaType := "audio"
	ext := ".webm"
	if track.Video {
		mediaType = "video"
		ext = ".mkv"
	}

	youtubeURL := "https://www.youtube.com/watch?v=" + track.ID
	endpoint := fmt.Sprintf("%s/download?url=%s&type=%s", shrutiBaseURL, youtubeURL, mediaType)

	gologging.DebugF("ShrutiApi: requesting token for %s (%s)", track.ID, mediaType)

	// Step 1: get download token
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("shrutiapi: token HTTP %d", resp.StatusCode)
	}

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("shrutiapi: decode token response: %w", err)
	}

	if status, _ := data["status"].(string); status != "success" {
		return "", fmt.Errorf("shrutiapi: status=%s", status)
	}

	token, _ := data["download_token"].(string)
	if token == "" {
		return "", errors.New("shrutiapi: no download_token in response")
	}

	// Step 2: stream the file
	streamURL := fmt.Sprintf("%s/stream/%s?token=%s&type=%s", shrutiBaseURL, track.ID, token, mediaType)
	gologging.DebugF("ShrutiApi: streaming from %s", streamURL)

	sreq, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: build stream request: %w", err)
	}

	sresp, err := s.client.Do(sreq)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: stream request failed: %w", err)
	}
	defer sresp.Body.Close()

	if sresp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("shrutiapi: stream HTTP %d", sresp.StatusCode)
	}

	path := getPath(track, ext)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, sresp.Body); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("shrutiapi: write file: %w", err)
	}

	if !fileExists(path) {
		return "", errors.New("shrutiapi: empty file after download")
	}

	gologging.InfoF("ShrutiApi: downloaded %s -> %s", track.ID, path)
	return path, nil
}
