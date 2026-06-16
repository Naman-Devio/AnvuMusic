/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package utils

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Laky-64/gologging"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func ShortTitle(title string, max ...int) string {
	limit := 25
	if len(max) > 0 {
		limit = max[0]
	}
	runes := []rune(title)
	if len(runes) <= limit {
		return title
	}
	return string(runes[:limit]) + "..."
}

func CleanURL(raw string) string {
	before, _, _ := strings.Cut(raw, "?")
	return before
}

// ResolveArtwork downloads a URL-based artwork to a local cache file
// so it can be used as media in SendOptions. If the input is already
// a local path or empty, it's returned as-is.
func ResolveArtwork(artwork string, trackID string) string {
	if artwork == "" {
		return ""
	}
	// Already a local file - return as-is
	if !strings.HasPrefix(artwork, "http://") && !strings.HasPrefix(artwork, "https://") {
		return artwork
	}
	
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		gologging.Error("Artwork cache dir creation failed: " + err.Error())
		return artwork
	}
	
	ext := ".jpg"
	if strings.Contains(artwork, ".png") {
		ext = ".png"
	} else if strings.Contains(artwork, ".webp") {
		ext = ".webp"
	}
	
	dest := filepath.Join(cacheDir, "art_"+trackID+ext)
	
	// Check cache
	if _, err := os.Stat(dest); err == nil {
		gologging.Debug("Using cached artwork: " + dest)
		return dest
	}
	
	// Download with timeout
	client := &http.Client{Timeout: 10 * time.Second}
	gologging.Debug("Downloading artwork from URL: " + artwork[:min(len(artwork), 80)])
	resp, err := client.Get(artwork)
	if err != nil {
		gologging.Error("Artwork download failed: " + err.Error())
		return artwork
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		gologging.Error("Artwork download HTTP " + strconv.Itoa(resp.StatusCode))
		return artwork
	}
	
	f, err := os.Create(dest)
	if err != nil {
		gologging.Error("Artwork file create failed: " + err.Error())
		return artwork
	}
	defer f.Close()
	
	if _, err := io.Copy(f, resp.Body); err != nil {
		gologging.Error("Artwork file write failed: " + err.Error())
		os.Remove(dest)
		return artwork
	}
	
	gologging.Debug("Artwork cached: " + dest)
	return dest
}

func MentionHTML(u *tg.UserObj) string {
	if u == nil {
		return "Unknown"
	}

	fullName := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if fullName == "" {
		fullName = "User"
	}
	fullName = html.EscapeString(ShortTitle(fullName, 15))

	return fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", u.ID, fullName)
}

// IfElse returns `a` if condition is true, else returns `b`.
func IfElse[T any](condition bool, a, b T) T {
	if condition {
		return a
	}
	return b
}

// ParseBool converts strings like "on", "off", "enable", "disable", "true", "false"
// into a boolean value. Returns an error if input is invalid.
func ParseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "enable", "enabled", "true", "1", "yes", "y":
		return true, nil
	case "off", "disable", "disabled", "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean string: %q", s)
	}
}

// IntToStr converts any signed integer type to string.
func IntToStr[T ~int | ~int8 | ~int16 | ~int32 | ~int64](v T) string {
	return strconv.FormatInt(int64(v), 10)
}
