/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package platforms

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Laky-64/gologging"
	"github.com/amarnathcjd/gogram/telegram"
	"resty.dev/v3"

	state "main/internal/core/models"
	"main/internal/database"
	"main/internal/utils"
)

// TODO: NOT TESTED YET

type platformEntry struct {
	platform state.Platform
	priority int
}

type PlatformRegistry struct {
	platforms []platformEntry
	mu        sync.RWMutex
}

var (
	registry = &PlatformRegistry{
		platforms: make([]platformEntry, 0),
	}
	rc = resty.New()
)

// Register adds a platform to the registry with given priority
func Register(priority int, p state.Platform) {
	registry.mu.Lock()
	defer registry.mu.Unlock()

	registry.platforms = append(registry.platforms, platformEntry{p, priority})
	sort.Slice(registry.platforms, func(i, j int) bool {
		return registry.platforms[i].priority > registry.platforms[j].priority
	})
}

// GetOrderedPlatforms returns all platforms sorted by priority
func GetOrderedPlatforms() []state.Platform {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	res := make([]state.Platform, len(registry.platforms))
	for i, e := range registry.platforms {
		res[i] = e.platform
	}
	return res
}

func findPlatform(url string) state.Platform {
	for _, p := range GetOrderedPlatforms() {
		if p.CanGetTracks(url) {
			return p
		}
	}
	return nil
}

// GetTracks extracts tracks from the given query or message context
func GetTracks(m *telegram.NewMessage, video bool) ([]*state.Track, error) {
	gologging.Debug("GetTracks called | video: " + strconv.FormatBool(video))

	// 1. URL Processing
	if urls, _ := utils.ExtractURLs(m); len(urls) > 0 {
		gologging.Debug("URLs detected in message: " + strconv.Itoa(len(urls)))
		tracks, errs := processURLs(urls, video)
		if len(tracks) > 0 {
			gologging.Info("Returning tracks from URLs")
			return tracks, nil
		}

		if !hasPlayableReply(m) {
			return nil, combineErrors("no supported platform for given URL(s)", errs)
		}
		gologging.Debug("URL extraction failed, falling back to reply media check")
	}

	// 2. Query/Search Processing
	if query := m.Args(); query != "" {
		gologging.Info("Processing search query: " + query)
		tracks, err := processSearchQuery(query, video)
		if err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}

	// 3. Reply Chain Processing
	if m.IsReply() {
		return processReplyChain(m)
	}

	gologging.Info("No tracks found after checking URLs, Query, and Replies")
	return nil, errors.New("no tracks found")
}

func SearchTracks(query string, video bool) ([]*state.Track, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("empty query")
	}

	if p := findPlatform(query); p != nil && p.Name() != PlatformYouTube {
		tracks, err := p.Search(query, video)
		if err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}

	gologging.Info("Searching YouTube with query: " + query)
	tracks, err := yt.GetTracks(query, video)
	if err != nil {
		gologging.Error("YouTube search failed: " + err.Error())
		return nil, err
	}
	return tracks, nil
}

func processURLs(urls []string, video bool) ([]*state.Track, []string) {
	var allTracks []*state.Track
	var errs []string

	for _, url := range urls {
		gologging.Info("Processing URL: " + url)
		p := findPlatform(url)
		if p == nil {
			errMsg := "No platform found for URL: " + url
			gologging.Error(errMsg)
			errs = append(errs, errMsg)
			continue
		}

		gologging.Debug("Matched platform [" + string(p.Name()) + "] for URL: " + url)
		tracks, err := p.GetTracks(url, video)
		if err != nil {
			if strings.Contains(err.Error(), "failed to extract metadata") {
				gologging.Debug("Silent skip: metadata extraction failed for " + url)
				continue
			}
			errMsg := string(p.Name()) + ": " + err.Error()
			gologging.Error(errMsg)
			errs = append(errs, errMsg)
			continue
		}

		gologging.Info("Tracks found: " + strconv.Itoa(len(tracks)))
		allTracks = append(allTracks, tracks...)
	}
	return allTracks, errs
}

func processSearchQuery(query string, video bool) ([]*state.Track, error) {
	if p := findPlatform(query); p != nil && p.Name() != PlatformYouTube {
		gologging.Debug("Query matches specific platform: " + string(p.Name()))
		tracks, err := p.GetTracks(query, video)
		if err == nil && len(tracks) > 0 {
			gologging.Info("Query handled by platform: " + string(p.Name()))
			return tracks, nil
		}
	}

	gologging.Info("Searching YouTube with query: " + query)
	tracks, err := yt.GetTracks(query, video)
	if err != nil {
		gologging.Error("YouTube search failed: " + err.Error())
		return nil, err
	}

	if len(tracks) > 0 {
		gologging.Info("YouTube search successful, returning top result")
		return []*state.Track{tracks[0]}, nil
	}

	gologging.Debug("YouTube search returned 0 results for: " + query)
	return nil, nil
}

func processReplyChain(m *telegram.NewMessage) ([]*state.Track, error) {
	gologging.Debug("Message is a reply, resolving media chain...")
	target, isVideo, err := findMediaInReply(m)
	if err != nil {
		gologging.Info("Reply chain does not contain valid media")
		return nil, err
	}

	tg := &TelegramPlatform{}
	track, err := tg.GetTracksByMessage(target)
	if err != nil {
		gologging.Error("Failed to get track from Telegram reply: " + err.Error())
		return nil, err
	}

	track.Video = isVideo
	if isVideo {
		noThumb, err := database.ThumbnailsDisabled(m.ChannelID())
		if err != nil || !noThumb {

			gologging.Debug(
				"Reply media is video, handling thumbnail for ID: " + track.ID,
			)
			downloadThumbnail(target, track)
		}
	}

	gologging.Info("Returning track from Telegram reply")
	return []*state.Track{track}, nil
}

// Download attempts to download a track using available downloaders
func Download(
	ctx context.Context,
	track *state.Track,
	statusMsg *telegram.NewMessage,
) (string, error) {
	gologging.Debug(
		"Download requested for track: " + track.ID + " | Source: " + string(
			track.Source,
		),
	)

	if track == nil {
		return "", errors.New("download failed: nil track")
	}

	if track.Source == PlatformYouTube {
		return downloadYouTubeTrack(ctx, track, statusMsg)
	}

	var errs []string

	platforms := GetOrderedPlatforms()
	for _, p := range platforms {
		if !p.CanDownload(track.Source) {
			gologging.Debug(
				"Platform [" + string(
					p.Name(),
				) + "] cannot download source: " + string(
					track.Source,
				),
			)
			continue
		}

		gologging.Debug("Attempting download with platform: " + string(p.Name()))
		path, err := p.Download(ctx, track, statusMsg)
		if err == nil {
			gologging.Info("Download successful via " + string(p.Name()) + " -> " + path)
			return path, nil
		}

		if errors.Is(err, context.Canceled) {
			gologging.Debug("Download canceled by context (user/system request)")
			return "", err
		}

		errMsg := string(p.Name()) + ": " + err.Error()
		gologging.Error("Download failed with " + errMsg)
		errs = append(errs, errMsg)
	}

	if len(errs) > 0 {
		return "", combineErrors("Multiple download errors occurred", errs)
	}

	return "", errors.New("no downloader available for source: " + string(track.Source))
}

func downloadYouTubeTrack(
	ctx context.Context,
	track *state.Track,
	statusMsg *telegram.NewMessage,
) (string, error) {
	if f := findFile(track); f != "" {
		gologging.Debug("YouTube download cache hit -> " + f)
		return f, nil
	}

	candidates := []state.Platform{}
	for _, p := range GetOrderedPlatforms() {
		if p.Name() == PlatformShrutiApi || p.Name() == PlatformYtDlp {
			if p.CanDownload(track.Source) {
				candidates = append(candidates, p)
			}
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("no YouTube downloaders available")
	}

	type result struct {
		path   string
		err    error
		name   string
	}

	resultCh := make(chan result, len(candidates))
	raceCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, p := range candidates {
		platform := p
		go func() {
			path, err := platform.Download(raceCtx, track, statusMsg)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					resultCh <- result{err: err, name: string(platform.Name())}
					return
				}
				gologging.WarnF("Download failed via %s: %v", platform.Name(), err)
				resultCh <- result{err: err, name: string(platform.Name())}
				return
			}
			resultCh <- result{path: path, name: string(platform.Name())}
		}()
	}

	var errs []string
	for i := 0; i < len(candidates); i++ {
		res := <-resultCh
		if res.err == nil {
			cancel()
			gologging.Info("Download successful via " + res.name + " -> " + res.path)
			return res.path, nil
		}
		if errors.Is(res.err, context.Canceled) {
			return "", res.err
		}
		errs = append(errs, res.name+": "+res.err.Error())
	}

	if len(errs) > 0 {
		return "", combineErrors("Multiple download errors occurred", errs)
	}

	return "", errors.New("no YouTube downloader succeeded")
}

// --- Helpers ---

func findMediaInReply(m *telegram.NewMessage) (*telegram.NewMessage, bool, error) {
	curr, err := m.GetReplyMessage()
	if err != nil {
		gologging.Error("Failed to fetch initial reply: " + err.Error())
		return nil, false, fmt.Errorf("failed to get replied message: %w", err)
	}

	for i := 0; i < 2; i++ {
		gologging.Debug(
			"Checking reply level " + strconv.Itoa(i+1) + " for playable media",
		)
		if v, a := playableMedia(curr); v || a {
			gologging.Debug(
				"Found media in reply chain | isVideo: " + strconv.FormatBool(v),
			)
			return curr, v, nil
		}

		if !curr.IsReply() {
			break
		}

		next, err := curr.GetReplyMessage()
		if err != nil {
			gologging.Debug("Reply chain ended due to error: " + err.Error())
			break
		}
		curr = next
	}

	return nil, false, errors.New("⚠️ Reply with a valid media (audio/video)")
}

func downloadThumbnail(m *telegram.NewMessage, t *state.Track) {
	if err := os.MkdirAll("cache", os.ModePerm); err != nil {
		gologging.Error("Thumbnail cache creation failed: " + err.Error())
		return
	}

	dest := filepath.Join("cache", "thumb_"+t.ID+".jpg")
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		gologging.Debug("Downloading thumbnail to: " + dest)
		path, err := m.Download(&telegram.DownloadOptions{
			ThumbOnly: true,
			FileName:  dest,
		})
		if err == nil {
			t.Artwork = path
			gologging.Debug("Thumbnail successfully linked: " + path)
		} else {
			gologging.Error("Thumbnail download failed: " + err.Error())
		}
	} else {
		gologging.Debug("Using cached thumbnail for track: " + t.ID)
		t.Artwork = dest
	}
}

func hasPlayableReply(m *telegram.NewMessage) bool {
	if !m.IsReply() {
		return false
	}
	rmsg, err := m.GetReplyMessage()
	if err != nil {
		return false
	}
	v, a := playableMedia(rmsg)
	return v || a
}

func combineErrors(prefix string, errs []string) error {
	if len(errs) == 0 {
		return errors.New(prefix)
	}
	return errors.New(prefix + "\n• " + strings.Join(errs, "\n• "))
}

func Init() (func(), error) {
	return func() {
		rc.Close()
	}, nil
}
