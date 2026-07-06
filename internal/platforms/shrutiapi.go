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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"

	"main/internal/core"
	state "main/internal/core/models"
	"main/internal/utils"
)

const (
	PlatformShrutiApi     state.PlatformName = "ShrutiApi"
	shrutiPrimaryBaseURL                     = "https://api01.shrutibots.site"
	shrutiLegacyBaseURL                      = "https://api.shrutibots.site"
	riteshDefaultBaseURL                    = "https://web.riteshyt.in"
	onegrabDefaultBaseURL                   = "https://api.onegrab.fun"
	arcMusicDefaultBaseURL                  = "https://api.arcmusic.fun"
	xbitcodeDefaultBaseURL                  = "https://tgapi.xbitcode.com"
)

var (
	shrutiAPIKey    = "ShrutiBotsbEZWD1zDFUXrrvVZQaE9"
	riteshBaseURL   = riteshDefaultBaseURL
	riteshAPIKey    = "ritesh_free_e7839ed4ca0ae3afa8bf6b5f"
	shrutiUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
	shrutiAccept   = "application/json, text/plain, */*"

	// OneGrab API
	onegrabBaseURL = onegrabDefaultBaseURL
	onegrabAPIKey  string

	// XBitCode API - supports key rotation
	xbitcodeBaseURL = xbitcodeDefaultBaseURL
	xbitcodeKeys    = []string{
		"xbit__sD6rciPnwm7sDDLhMsbtXzA1vG3jv1g",
		"xbit_cOB0artj3kn9CL07sMoVFTEu9O3xTbTT",
		"xbit_zWU2IUXay1lTFwLeNCtm1YboGhskUz_V",
	}
	xbitcodeKeyIndex atomic.Uint64

	// ARC Music API
	arcMusicBaseURL = arcMusicDefaultBaseURL
	arcMusicAPIKey  = "ARCe00f3f51934c7b759b455b"
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

	// OneGrab config — use env var or fallback to default test key
	if key := strings.TrimSpace(os.Getenv("ONEGRAB_API_KEY")); key != "" {
		onegrabAPIKey = key
	} else {
		onegrabAPIKey = "ebf9ef_me9NNgdCX_a9G-SFZzAvAPoC2iifF6MH"
	}
	if u := strings.TrimSpace(os.Getenv("ONEGRAB_API_URL")); u != "" {
		onegrabBaseURL = strings.TrimRight(u, "/")
	}

	// XBitCode config — environment variable can override with comma-separated keys
	if keys := strings.TrimSpace(os.Getenv("XBITCODE_API_KEYS")); keys != "" {
		parts := strings.Split(keys, ",")
		var clean []string
		for _, p := range parts {
			if k := strings.TrimSpace(p); k != "" {
				clean = append(clean, k)
			}
		}
		if len(clean) > 0 {
			xbitcodeKeys = clean
		}
	}

	// ARC Music config
	if key := strings.TrimSpace(os.Getenv("ARC_MUSIC_API_KEY")); key != "" {
		arcMusicAPIKey = key
	}
	if u := strings.TrimSpace(os.Getenv("ARC_MUSIC_API_URL")); u != "" {
		arcMusicBaseURL = strings.TrimRight(u, "/")
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

// Download races all available APIs in parallel and returns the first successful result.
func (s *ShrutiApiPlatform) Download(
	ctx context.Context,
	track *state.Track,
	statusMsg *telegram.NewMessage,
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
	encodedURL := url.QueryEscape(youtubeURL)

	type apiResult struct {
		path string
		err  error
		name string
	}

	resultCh := make(chan apiResult, 7) // buffer for all API attempts
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Helper to launch an API download goroutine
	launch := func(name string, fn func() (string, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			path, err := fn()
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					gologging.WarnF("ShrutiApi [%s] failed for %s: %v", name, track.ID, err)
				}
				resultCh <- apiResult{err: err, name: name}
				return
			}
			resultCh <- apiResult{path: path, name: name}
		}()
	}

	// 1. Shruti API (primary)
	launch("Shruti", func() (string, error) {
		return s.downloadWithShruti(raceCtx, shrutiPrimaryBaseURL, encodedURL, mediaType, track, ext)
	})

	// 2. Shruti API (legacy)
	launch("ShrutiLegacy", func() (string, error) {
		return s.downloadWithShruti(raceCtx, shrutiLegacyBaseURL, encodedURL, mediaType, track, ext)
	})

	// 3. Ritesh API
	launch("Ritesh", func() (string, error) {
		return s.downloadWithRitesh(raceCtx, youtubeURL, mediaType, track, ext)
	})

	// 4. OneGrab API
	if onegrabAPIKey != "" {
	launch("OneGrab", func() (string, error) {
		return s.downloadWithOneGrab(raceCtx, encodedURL, mediaType, track, ext, statusMsg)
	})
	}

	// 5. XBitCode API
	launch("XBitCode", func() (string, error) {
		return s.downloadWithXBitCode(raceCtx, track.ID, mediaType, track, ext)
	})

	// 6. ARC Music API
	launch("ARC", func() (string, error) {
		return s.downloadWithARC(raceCtx, encodedURL, track, ext)
	})

	// Wait for all goroutines to finish, then close resultCh
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results — return first success, or collect all errors
	var errs []string
	for res := range resultCh {
		if res.err == nil {
			cancel() // cancel all other in-flight downloads
			gologging.InfoF("ShrutiApi: downloaded %s via %s -> %s", track.ID, res.name, res.path)
			return res.path, nil
		}
		if !errors.Is(res.err, context.Canceled) {
			errs = append(errs, res.name+": "+res.err.Error())
		}
	}

	if len(errs) > 0 {
		return "", fmt.Errorf("shrutiapi: all APIs failed:\n  • %s", strings.Join(errs, "\n  • "))
	}
	return "", errors.New("shrutiapi: download failed")
}

// ── Shruti API ──────────────────────────────────────────────

func (s *ShrutiApiPlatform) downloadWithShruti(
	ctx context.Context,
	baseURL, encodedURL, mediaType string,
	track *state.Track,
	ext string,
) (string, error) {
	endpoint := fmt.Sprintf(
		"%s/download?url=%s&type=%s&api_key=%s",
		baseURL, encodedURL, mediaType, url.QueryEscape(shrutiAPIKey),
	)

	gologging.DebugF("ShrutiApi: requesting token for %s (%s) via %s", track.ID, mediaType, baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", shrutiUserAgent)
	req.Header.Set("Accept", shrutiAccept)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	// Direct binary response
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		path := getPath(track, ext)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			return "", fmt.Errorf("write file: %w", err)
		}
		if !fileExists(path) {
			return "", errors.New("empty file after download")
		}
		return path, nil
	}

	// JSON response with download_token
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if status, _ := data["status"].(string); status != "success" {
		return "", fmt.Errorf("status=%s", status)
	}

	token, _ := data["download_token"].(string)
	if token == "" {
		return "", errors.New("no download_token in response")
	}

	streamURL := fmt.Sprintf("%s/stream/%s?token=%s&type=%s", baseURL, track.ID, token, mediaType)
	gologging.DebugF("ShrutiApi: streaming from %s", streamURL)

	sreq, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", fmt.Errorf("build stream request: %w", err)
	}
	sreq.Header.Set("User-Agent", shrutiUserAgent)
	sreq.Header.Set("Accept", shrutiAccept)

	sresp, err := s.client.Do(sreq)
	if err != nil {
		return "", fmt.Errorf("stream request failed: %w", err)
	}
	defer sresp.Body.Close()

	if sresp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sresp.Body)
		return "", fmt.Errorf("stream HTTP %d: %s", sresp.StatusCode, strings.TrimSpace(string(body)))
	}

	return s.saveToFile(sresp.Body, track, ext)
}

// ── Ritesh API ──────────────────────────────────────────────

func (s *ShrutiApiPlatform) downloadWithRitesh(
	ctx context.Context,
	youtubeURL, mediaType string,
	track *state.Track,
	ext string,
) (string, error) {
	safeQuery := url.QueryEscape(youtubeURL)
	var endpoint string
	if riteshAPIKey != "" {
		endpoint = fmt.Sprintf("%s/downloads/%s/%s%s", riteshBaseURL, riteshAPIKey, safeQuery, ext)
	} else {
		endpoint = fmt.Sprintf("%s/downloads/stream?query=%s&dl_type=%s", riteshBaseURL, safeQuery, mediaType)
	}

	gologging.DebugF("ShrutiApi: Ritesh download for %s via %s", track.ID, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build ritesh request: %w", err)
	}
	req.Header.Set("User-Agent", shrutiUserAgent)
	req.Header.Set("Accept", shrutiAccept)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ritesh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ritesh HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return s.saveToFile(resp.Body, track, ext)
}

// ── OneGrab API ─────────────────────────────────────────────

var onegrabTelegramRegex = regexp.MustCompile(`https?://t\.me/([a-zA-Z0-9_]{4,})/(\d+)`)

func (s *ShrutiApiPlatform) downloadWithOneGrab(
	ctx context.Context,
	encodedURL, mediaType string,
	track *state.Track,
	ext string,
	statusMsg *telegram.NewMessage,
) (string, error) {
	key := onegrabAPIKey
	if key == "" {
		key = os.Getenv("ONEGRAB_API_KEY")
	}

	endpoint := fmt.Sprintf("%s/api/track?url=%s&api_key=%s", onegrabBaseURL, encodedURL, key)
	if track.Video {
		endpoint += "&video=true"
	}

	gologging.DebugF("ShrutiApi: OneGrab download for %s", track.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build onegrab request: %w", err)
	}
	req.Header.Set("X-API-Key", key)
	req.Header.Set("User-Agent", shrutiUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("onegrab request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("onegrab HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read onegrab response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("decode onegrab response: %w", err)
	}

	cdnurl, _ := data["cdnurl"].(string)
	if cdnurl == "" {
		return "", errors.New("onegrab: no cdnurl in response")
	}

	gologging.DebugF("ShrutiApi: OneGrab cdnurl: %s", cdnurl[:min(len(cdnurl), 120)])

	// If cdnurl is a Telegram message link, download via bot client (YukkiMusic approach)
	if onegrabTelegramRegex.MatchString(cdnurl) {
		gologging.InfoF("OneGrab: cdnurl is Telegram link, downloading via bot: %s", cdnurl[:min(len(cdnurl), 80)])
		pm := utils.GetProgress(statusMsg)
		path, err := s.downloadFromTelegram(ctx, cdnurl, track, ext, pm)
		if err != nil {
			return "", fmt.Errorf("onegrab: telegram download failed: %w", err)
		}
		return path, nil
	}

	// Direct HTTP download
	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cdnurl, nil)
	if err != nil {
		return "", fmt.Errorf("onegrab: build cdn request: %w", err)
	}
	dlReq.Header.Set("User-Agent", shrutiUserAgent)

	dlResp, err := s.client.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("onegrab: cdn request failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("onegrab: cdn HTTP %d", dlResp.StatusCode)
	}

	return s.saveToFile(dlResp.Body, track, ext)
}

// ── XBitCode API ────────────────────────────────────────────

func (s *ShrutiApiPlatform) downloadWithXBitCode(
	ctx context.Context,
	videoID, mediaType string,
	track *state.Track,
	ext string,
) (string, error) {
	endpoint := fmt.Sprintf("%s/info/%s", xbitcodeBaseURL, videoID)
	gologging.DebugF("ShrutiApi: XBitCode download for %s", track.ID)

	// Key rotation: try each key starting from last successful index
	startIdx := int(xbitcodeKeyIndex.Load() % uint64(len(xbitcodeKeys)))
	lastErr := errors.New("no keys available")

	for i := 0; i < len(xbitcodeKeys); i++ {
		idx := (startIdx + i) % len(xbitcodeKeys)
		key := xbitcodeKeys[idx]

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = fmt.Errorf("build xbitcode request: %w", err)
			continue
		}
		req.Header.Set("x-api-key", key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", shrutiUserAgent)

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("xbitcode request failed: %w", err)
			continue
		}

		// If auth/rate-limit error, try next key
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("xbitcode HTTP %d (key exhausted)", resp.StatusCode)
			// Advance the global index past this failed key
			xbitcodeKeyIndex.Add(1)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("xbitcode HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read xbitcode response: %w", err)
			continue
		}

		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			lastErr = fmt.Errorf("decode xbitcode response: %w", err)
			continue
		}

		if status, _ := data["status"].(string); status != "success" {
			lastErr = fmt.Errorf("xbitcode status=%s", status)
			continue
		}

		// Pick audio_url or video_url based on media type
		var dlURL string
		if track.Video {
			dlURL, _ = data["video_url"].(string)
		} else {
			dlURL, _ = data["audio_url"].(string)
		}
		if dlURL == "" {
			lastErr = errors.New("xbitcode: no audio/video_url in response")
			continue
		}

		// Advance key index past the successful key for next call
		xbitcodeKeyIndex.Store(uint64((idx + 1) % len(xbitcodeKeys)))

		// Download from the returned URL
		dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
		if err != nil {
			return "", fmt.Errorf("xbitcode: build dl request: %w", err)
		}
		dlReq.Header.Set("User-Agent", shrutiUserAgent)

		dlResp, err := s.client.Do(dlReq)
		if err != nil {
			return "", fmt.Errorf("xbitcode: dl request failed: %w", err)
		}
		defer dlResp.Body.Close()

		if dlResp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("xbitcode: dl HTTP %d", dlResp.StatusCode)
		}

		return s.saveToFile(dlResp.Body, track, ext)
	}

	return "", lastErr
}

// ── ARC Music API ───────────────────────────────────────────

func (s *ShrutiApiPlatform) downloadWithARC(
	ctx context.Context,
	encodedURL string,
	track *state.Track,
	ext string,
) (string, error) {
	isVideo := "false"
	if track.Video {
		isVideo = "true"
	}

	// Step 1: Initiate download
	initEndpoint := fmt.Sprintf(
		"%s/youtube/v2/download?query=%s&isVideo=%s&api_key=%s",
		arcMusicBaseURL, encodedURL, isVideo, arcMusicAPIKey,
	)

	gologging.DebugF("ShrutiApi: ARC Music initiating for %s", track.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, initEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("arc: build init request: %w", err)
	}
	req.Header.Set("User-Agent", shrutiUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("arc: init request failed: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("arc: read init response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("arc: init HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var initData map[string]any
	if err := json.Unmarshal(body, &initData); err != nil {
		return "", fmt.Errorf("arc: decode init response: %w", err)
	}

	jobID, _ := initData["job_id"].(string)
	if jobID == "" {
		return "", fmt.Errorf("arc: no job_id in response")
	}

	// Step 2: Poll for completion
	pollURL := fmt.Sprintf("%s/youtube/jobStatus", arcMusicBaseURL)
	pollTicker := time.NewTicker(3 * time.Second)
	defer pollTicker.Stop()

	for attempt := 0; attempt < 30; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-pollTicker.C:
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodGet, pollURL+"?job_id="+url.QueryEscape(jobID)+"&api_key="+url.QueryEscape(arcMusicAPIKey), nil)
		if err != nil {
			continue
		}
		pollReq.Header.Set("User-Agent", shrutiUserAgent)

		pollResp, err := s.client.Do(pollReq)
		if err != nil {
			continue
		}

		pollBody, err := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()
		if err != nil {
			continue
		}

		var pollData map[string]any
		if err := json.Unmarshal(pollBody, &pollData); err != nil {
			continue
		}

		job, _ := pollData["job"].(map[string]any)
		if job == nil {
			continue
		}

		jobStatus, _ := job["status"].(string)
		switch strings.ToLower(jobStatus) {
		case "done":
			result, _ := job["result"].(map[string]any)
			if result == nil {
				return "", errors.New("arc: no result in job")
			}
			publicURL, _ := result["public_url"].(string)
			if publicURL == "" {
				return "", errors.New("arc: no public_url in job result")
			}

			dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, publicURL, nil)
			if err != nil {
				return "", fmt.Errorf("arc: build download request: %w", err)
			}
			dlReq.Header.Set("User-Agent", shrutiUserAgent)

			dlResp, err := s.client.Do(dlReq)
			if err != nil {
				return "", fmt.Errorf("arc: download request failed: %w", err)
			}
			defer dlResp.Body.Close()

			if dlResp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("arc: download HTTP %d", dlResp.StatusCode)
			}

			return s.saveToFile(dlResp.Body, track, ext)

		case "failed", "error":
			return "", fmt.Errorf("arc: job failed: %v", pollData)
		}
	}

	return "", errors.New("arc: max retries exceeded")
}

// ── OneGrab Telegram CDN download (like YukkiMusic's fallenapi.go) ──

func (s *ShrutiApiPlatform) downloadFromTelegram(
	ctx context.Context,
	dlURL string,
	track *state.Track,
	ext string,
	pm *telegram.ProgressManager,
) (string, error) {
	matches := onegrabTelegramRegex.FindStringSubmatch(dlURL)
	if len(matches) < 3 {
		return "", fmt.Errorf("invalid telegram download url: %s", dlURL)
	}

	username := matches[1]
	messageID, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", fmt.Errorf("invalid message ID: %v", err)
	}

	msg, err := core.Bot.GetMessageByID(username, int32(messageID))
	if err != nil {
		return "", fmt.Errorf("failed to fetch Telegram message: %w", err)
	}

	path := getPath(track, ext)
	dOpts := &telegram.DownloadOptions{
		FileName: path,
		Ctx:      ctx,
	}
	if pm != nil {
		dOpts.ProgressManager = pm
	}

	_, err = msg.Download(dOpts)
	if err != nil {
		os.Remove(path)
		return "", fmt.Errorf("telegram download failed: %w", err)
	}

	if !fileExists(path) {
		os.Remove(path)
		return "", errors.New("empty file after telegram download")
	}

	gologging.InfoF("OneGrab: downloaded %s via Telegram CDN -> %s", track.ID, path)
	return path, nil
}

// ── Helpers ─────────────────────────────────────────────────

func (s *ShrutiApiPlatform) saveToFile(
	reader io.Reader,
	track *state.Track,
	ext string,
) (string, error) {
	path := getPath(track, ext)
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("write file: %w", err)
	}

	if !fileExists(path) {
		os.Remove(path)
		return "", errors.New("empty file after download")
	}

	gologging.InfoF("ShrutiApi: downloaded %s -> %s", track.ID, path)
	return path, nil
}
