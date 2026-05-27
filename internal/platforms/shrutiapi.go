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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	state "main/internal/core/models"
)

const (
	PlatformShrutiApi    state.PlatformName = "ShrutiApi"
	shrutiPrimaryBaseURL                     = "https://api.shrutibots.site"
	shrutiLegacyBaseURL                      = "https://shrutibots.site"
	riteshDefaultBaseURL                    = "https://api.riteshyt.in"
)

var (
	shrutiAPIKey    = "ShrutiBotslKqAAhXsyOVUPWb7EmIg"
	riteshBaseURL   = riteshDefaultBaseURL
	riteshAPIKey    = "ritesh_free_3349aed8ab6e1bcd3e51999c"
	shrutiUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	shrutiAccept   = "application/json, text/plain, */*"
)

type ShrutiApiPlatform struct {
	name   state.PlatformName
	client *http.Client
}

func init() {
	if key := strings.TrimSpace(os.Getenv("SHRUTI_API_KEY")); key != "" {
		shrutiAPIKey = key
	}
	if apiURL := strings.TrimSpace(os.Getenv("API_URL")); apiURL != "" {
		riteshBaseURL = strings.TrimRight(apiURL, "/")
	}
	if key := strings.TrimSpace(os.Getenv("API_KEY")); key != "" {
		riteshAPIKey = key
	}
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
	baseURLs := []string{shrutiPrimaryBaseURL, shrutiLegacyBaseURL}

	var lastErr error
	for _, baseURL := range baseURLs {
		path, err := s.downloadWithBase(ctx, baseURL, youtubeURL, mediaType, track, ext)
		if err == nil {
			if baseURL != shrutiPrimaryBaseURL {
				gologging.WarnF("ShrutiApi: recovered via legacy endpoint for %s", track.ID)
			}
			return path, nil
		}
		lastErr = err
		if errors.Is(err, context.Canceled) {
			return "", err
		}
		gologging.WarnF("ShrutiApi: failed on %s for %s: %v", baseURL, track.ID, err)
	}

	if path, err := s.downloadWithRitesh(ctx, youtubeURL, mediaType, track, ext); err == nil {
		gologging.InfoF("ShrutiApi: recovered via Ritesh fallback for %s", track.ID)
		return path, nil
	} else if !errors.Is(err, context.Canceled) {
		gologging.WarnF("ShrutiApi: Ritesh fallback failed for %s: %v", track.ID, err)
		if lastErr != nil {
			return "", fmt.Errorf("%w; ritesh fallback: %v", lastErr, err)
		}
		return "", err
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New("shrutiapi: download failed")
}

func (s *ShrutiApiPlatform) downloadWithBase(
	ctx context.Context,
	baseURL string,
	youtubeURL string,
	mediaType string,
	track *state.Track,
	ext string,
) (string, error) {
	encodedURL := url.QueryEscape(youtubeURL)
	endpoint := fmt.Sprintf(
		"%s/download?url=%s&type=%s&api_key=%s",
		baseURL,
		encodedURL,
		mediaType,
		url.QueryEscape(shrutiAPIKey),
	)

	gologging.DebugF("ShrutiApi: requesting token for %s (%s) via %s", track.ID, mediaType, baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: build request: %w", err)
	}
	req.Header.Set("User-Agent", shrutiUserAgent)
	req.Header.Set("Accept", shrutiAccept)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("shrutiapi: token HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: read token response: %w", err)
	}

	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		path := getPath(track, ext)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			return "", fmt.Errorf("shrutiapi: write file: %w", err)
		}
		if !fileExists(path) {
			return "", errors.New("shrutiapi: empty file after download")
		}
		gologging.InfoF("ShrutiApi: downloaded %s -> %s", track.ID, path)
		return path, nil
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("shrutiapi: decode token response: %w", err)
	}

	if status, _ := data["status"].(string); status != "success" {
		return "", fmt.Errorf("shrutiapi: status=%s", status)
	}

	token, _ := data["download_token"].(string)
	if token == "" {
		return "", errors.New("shrutiapi: no download_token in response")
	}

	streamURL := fmt.Sprintf(
		"%s/stream/%s?token=%s&type=%s",
		baseURL,
		track.ID,
		token,
		mediaType,
	)
	gologging.DebugF("ShrutiApi: streaming from %s", streamURL)

	sreq, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: build stream request: %w", err)
	}
	sreq.Header.Set("User-Agent", shrutiUserAgent)
	sreq.Header.Set("Accept", shrutiAccept)

	sresp, err := s.client.Do(sreq)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: stream request failed: %w", err)
	}
	defer sresp.Body.Close()

	if sresp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sresp.Body)
		return "", fmt.Errorf("shrutiapi: stream HTTP %d: %s", sresp.StatusCode, strings.TrimSpace(string(body)))
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

func (s *ShrutiApiPlatform) downloadWithRitesh(
	ctx context.Context,
	youtubeURL string,
	mediaType string,
	track *state.Track,
	ext string,
) (string, error) {
	safeQuery := url.QueryEscape(youtubeURL)
	var endpoint string
	if riteshAPIKey != "" {
		endpoint = fmt.Sprintf(
			"%s/downloads/%s/%s%s",
			riteshBaseURL,
			riteshAPIKey,
			safeQuery,
			ext,
		)
	} else {
		endpoint = fmt.Sprintf(
			"%s/downloads/stream?query=%s&dl_type=%s",
			riteshBaseURL,
			safeQuery,
			mediaType,
		)
	}

	gologging.DebugF("ShrutiApi: requesting fallback Ritesh download for %s (%s) via %s", track.ID, mediaType, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: build ritesh request: %w", err)
	}
	req.Header.Set("User-Agent", shrutiUserAgent)
	req.Header.Set("Accept", shrutiAccept)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: ritesh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("shrutiapi: ritesh HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	path := getPath(track, ext)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("shrutiapi: ritesh create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("shrutiapi: ritesh write file: %w", err)
	}

	if !fileExists(path) {
		return "", errors.New("shrutiapi: ritesh empty file after download")
	}

	gologging.InfoF("ShrutiApi: downloaded %s -> %s", track.ID, path)
	return path, nil
}
