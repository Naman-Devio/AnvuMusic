/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package database

import (
	"fmt"
	"image"
	"image/color"
	stdDraw "image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Laky-64/gologging"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/webp"
)

var (
	cacheDir   = "cache"
	httpClient = &http.Client{Timeout: 12 * time.Second}
)

func init() {
	_ = os.MkdirAll(cacheDir, 0o755)
}

const (
	W           = 1280
	H           = 720
	renderScale = 2
	internalW   = W * renderScale
	internalH   = H * renderScale
)

type TrackInfo struct {
	VideoID  string
	Title    string
	Artist   string
	Channel  string
	Album    string
	Duration string
	Views    string
	Quality  string
	Source   string
	Artwork  string
	Elapsed  string
	Progress float64

	Verified       bool
	Explicit       bool
	HQ             bool
	Lossless       bool
	DolbyAtmos     bool
	Lyrics         bool
	Premium        bool
	ShuffleEnabled bool
	QueueEnabled   bool
	IsPlaying      bool
	RepeatMode     string
	Volume         float64
}

type palette struct {
	Dominant       color.RGBA
	Accent         color.RGBA
	Glow           color.RGBA
	Shadow         color.RGBA
	TextPrimary    color.RGBA
	TextSecondary  color.RGBA
	TextMuted      color.RGBA
	CardFill       color.RGBA
	CardStroke     color.RGBA
	TrackFill      color.RGBA
	TrackRemainder color.RGBA
	BackgroundTop  color.RGBA
	BackgroundBot  color.RGBA
	UseDarkText    bool
}

type layout struct {
	SafePad       int
	Gap           int
	ArtSize       int
	ArtX          int
	ArtY          int
	CardX         int
	CardY         int
	CardW         int
	CardH         int
	WatermarkX    int
	WatermarkY    int
	WatermarkW    int
	WatermarkH    int
	CardRadius    int
	ArtRadius     int
	InnerPad      int
	TitleTop      int
	ProgressY     int
	ProgressH     int
	ControlsY     int
	VolumeY       int
	WaveformY     int
	WaveformH     int
	BadgeGap      int
	ButtonRadius  int
	MainButtonRad int
}

type renderer struct {
	track    TrackInfo
	artwork  image.Image
	albumArt *image.RGBA
	palette  palette
	layout   layout

	background *image.RGBA
	ambient    *image.RGBA
	content    *image.RGBA
	backdrop   *image.RGBA
}

type textBlock struct {
	Lines      []string
	Size       int
	LineHeight int
	Height     int
}

type fontKey struct {
	Size int
	Bold bool
}

var (
	fontOnce  sync.Once
	fontMu    sync.Mutex
	fontCache = map[fontKey]font.Face{}
	regularTT *opentype.Font
	boldTT    *opentype.Font
)

func parseFonts() {
	fontOnce.Do(func() {
		if f, err := opentype.Parse(goregular.TTF); err == nil {
			regularTT = f
		} else {
			gologging.WarnF("[thumbgen] regular font parse failed: %v", err)
		}
		if f, err := opentype.Parse(gobold.TTF); err == nil {
			boldTT = f
		} else {
			gologging.WarnF("[thumbgen] bold font parse failed: %v", err)
		}
	})
}

func getFace(size int, bold bool) font.Face {
	parseFonts()
	if size < 8 {
		size = 8
	}
	key := fontKey{Size: size, Bold: bold}
	fontMu.Lock()
	defer fontMu.Unlock()
	if f, ok := fontCache[key]; ok {
		return f
	}
	src := regularTT
	if bold {
		src = boldTT
	}
	if src == nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(src, &opentype.FaceOptions{Size: float64(size), DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		gologging.WarnF("[thumbgen] face creation failed size=%d bold=%v: %v", size, bold, err)
		return basicfont.Face7x13
	}
	fontCache[key] = face
	return face
}

func Generate(t TrackInfo) (string, error) {
	if t.VideoID == "" {
		t.VideoID = fmt.Sprintf("anon_%d", time.Now().UnixNano())
	}
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("%s_anvu.png", t.VideoID))
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	rawPath := filepath.Join(cacheDir, fmt.Sprintf("raw_%s.img", t.VideoID))
	if err := downloadFile(t.Artwork, rawPath); err != nil {
		return "", fmt.Errorf("download artwork: %w", err)
	}
	defer os.Remove(rawPath)

	src, err := loadImage(rawPath)
	if err != nil {
		return "", fmt.Errorf("decode artwork: %w", err)
	}

	out, err := render(src, t)
	if err != nil {
		return "", fmt.Errorf("render thumbnail: %w", err)
	}
	f, err := os.Create(cachePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		_ = os.Remove(cachePath)
		return "", err
	}
	return cachePath, nil
}

func ClearCache() {
	matches, _ := filepath.Glob(filepath.Join(cacheDir, "*_anvu.png"))
	for _, file := range matches {
		_ = os.Remove(file)
	}
}

func render(src image.Image, t TrackInfo) (image.Image, error) {
	r := &renderer{track: normalizeTrack(t), artwork: src}
	r.layout = makeLayout(renderScale)
	r.palette = extractPalette(src)
	r.albumArt = coverCropResize(src, r.layout.ArtSize, r.layout.ArtSize)
	r.background = r.buildBackground()
	r.ambient = newRGBA(internalW, internalH)
	r.content = newRGBA(internalW, internalH)
	r.backdrop = flattenLayers(r.background, r.ambient)

	r.drawAmbient()
	r.backdrop = flattenLayers(r.background, r.ambient)
	r.drawWatermarkCard()
	r.drawArtwork()
	r.drawInfoCard()

	final := flattenLayers(r.background, r.ambient, r.content)
	return resizeSmooth(final, W, H), nil
}

func normalizeTrack(t TrackInfo) TrackInfo {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		t.Title = "Unknown Track"
	}
	if strings.TrimSpace(t.Artist) == "" {
		t.Artist = strings.TrimSpace(t.Channel)
	}
	if strings.TrimSpace(t.Artist) == "" {
		t.Artist = "Unknown Artist"
	}
	if strings.TrimSpace(t.Source) == "" {
		t.Source = "YouTube"
	}
	if t.Volume <= 0 {
		t.Volume = 0.72
	}
	if t.Volume > 1 {
		t.Volume = 1
	}
	if t.RepeatMode == "" {
		t.RepeatMode = "all"
	}
	if !t.IsPlaying {
		t.IsPlaying = true
	}
	return t
}

func makeLayout(scale int) layout {
	s := scale
	safe := 92 * s
	gap := 54 * s
	art := 468 * s
	cardX := safe + art + gap
	cardW := internalW - safe - cardX
	cardY := 168 * s
	cardH := internalH - 2*cardY
	return layout{
		SafePad:       safe,
		Gap:           gap,
		ArtSize:       art,
		ArtX:          safe,
		ArtY:          (internalH - art) / 2,
		CardX:         cardX,
		CardY:         cardY,
		CardW:         cardW,
		CardH:         cardH,
		WatermarkX:    safe,
		WatermarkY:    76 * s,
		WatermarkW:    312,
		WatermarkH:    72,
		CardRadius:    48,
		ArtRadius:     58,
		InnerPad:      84,
		TitleTop:      198,
		ProgressY:     cardY + cardH - 302,
		ProgressH:     12,
		ControlsY:     cardY + cardH - 164,
		VolumeY:       cardY + cardH - 102,
		WaveformY:     cardY + cardH - 360,
		WaveformH:     48,
		BadgeGap:      16,
		ButtonRadius:  32,
		MainButtonRad: 50,
	}
}

func (r *renderer) buildBackground() *image.RGBA {
	bg := coverCropResize(r.artwork, internalW, internalH)
	gaussianBlur(bg, 34)
	applyTint(bg, mixColor(color.RGBA{9, 11, 15, 255}, r.palette.Dominant, 0.08), 0.34)
	overlayVerticalGradient(bg, color.RGBA{255, 255, 255, 12}, r.palette.BackgroundTop, 0.00, 0.48)
	overlayVerticalGradient(bg, color.RGBA{0, 0, 0, 0}, r.palette.BackgroundBot, 0.46, 1.00)
	applyVignette(bg, 0.18)
	addFilmGrain(bg, 0.010)
	return bg
}

func (r *renderer) drawAmbient() {
	artRect := image.Rect(r.layout.ArtX, r.layout.ArtY, r.layout.ArtX+r.layout.ArtSize, r.layout.ArtY+r.layout.ArtSize)
	glowA := colorWithAlpha(mixColor(r.palette.Accent, color.RGBA{255, 255, 255, 255}, 0.28), 42)
	glowB := colorWithAlpha(mixColor(r.palette.Dominant, color.RGBA{255, 255, 255, 255}, 0.20), 28)
	drawRadialGlow(r.ambient, artRect.Min.X+artRect.Dx()/2-96, artRect.Min.Y+artRect.Dy()/2-72, artRect.Dx()+180, artRect.Dy()+180, glowA)
	drawRadialGlow(r.ambient, artRect.Min.X+artRect.Dx()/2+140, artRect.Min.Y+artRect.Dy()/2+96, artRect.Dx()-60, artRect.Dy()-60, glowB)
	drawRadialGlow(r.ambient, r.layout.CardX+r.layout.CardW/2, r.layout.CardY+180, r.layout.CardW+120, 240, colorWithAlpha(color.RGBA{255, 255, 255, 255}, 18))
	drawRadialGlow(r.ambient, r.layout.CardX+r.layout.CardW/2, r.layout.CardY+r.layout.CardH-110, r.layout.CardW+160, 180, colorWithAlpha(color.RGBA{8, 10, 16, 255}, 18))
}

func (r *renderer) drawWatermarkCard() {
	drawGlassPanel(r.content, r.backdrop, r.layout.WatermarkX, r.layout.WatermarkY, r.layout.WatermarkW, r.layout.WatermarkH, 28, r.palette, 0.22)
	labelColor := colorWithAlpha(r.palette.TextSecondary, 210)
	drawText(r.content, "ANVU MUSIC", r.layout.WatermarkX+32, r.layout.WatermarkY+47, labelColor, 24, true)
	sub := strings.ToUpper(strings.TrimSpace(r.track.Source))
	if sub != "" {
		drawText(r.content, sub, r.layout.WatermarkX+188, r.layout.WatermarkY+47, colorWithAlpha(r.palette.TextMuted, 175), 19, false)
	}
}

func (r *renderer) drawArtwork() {
	x, y, size := r.layout.ArtX, r.layout.ArtY, r.layout.ArtSize
	drawShadowRect(r.content, x-20, y+22, size+40, size+40, r.layout.ArtRadius+22, colorWithAlpha(r.palette.Shadow, 128), 30)
	drawShadowRect(r.content, x-10, y+10, size+20, size+20, r.layout.ArtRadius+14, colorWithAlpha(color.RGBA{255, 255, 255, 255}, 22), 16)
	pasteRoundedAA(r.content, r.albumArt, x, y, size, size, r.layout.ArtRadius)
	drawRoundedRectBorderAA(r.content, x, y, size, size, r.layout.ArtRadius, color.RGBA{255, 255, 255, 104}, 2)
	drawInnerHighlight(r.content, x, y, size, size, r.layout.ArtRadius, color.RGBA{255, 255, 255, 44})
	drawReflection(r.content, x, y, size, size, r.layout.ArtRadius)
}

func (r *renderer) drawInfoCard() {
	l := r.layout
	drawGlassPanel(r.content, r.backdrop, l.CardX, l.CardY, l.CardW, l.CardH, l.CardRadius, r.palette, 0.28)
	innerX := l.CardX + l.InnerPad
	innerW := l.CardW - 2*l.InnerPad
	top := l.CardY + 92

	drawText(r.content, "NOW PLAYING", innerX, top, colorWithAlpha(r.palette.TextMuted, 190), 24, true)
	if r.track.Premium {
		drawChip(r.content, innerX+208, top-32, "PREMIUM", 20, colorWithAlpha(r.palette.Accent, 56), colorWithAlpha(r.palette.TextPrimary, 214), 16, true)
	}

	title := layoutTitle(r.track.Title, innerW, 2, 86, 52, true)
	titleY := l.CardY + l.TitleTop
	for i, line := range title.Lines {
		drawText(r.content, line, innerX, titleY+i*title.LineHeight, r.palette.TextPrimary, title.Size, true)
	}
	artistY := titleY + title.Height + 38
	drawText(r.content, r.track.Artist, innerX, artistY, r.palette.TextSecondary, 38, false)
	cursorX := innerX + measureText(r.track.Artist, 38, false) + 18
	if r.track.Verified {
		drawVerifiedBadge(r.content, cursorX, artistY-23, 18)
		cursorX += 46
	}
	if album := strings.TrimSpace(r.track.Album); album != "" {
		drawText(r.content, album, innerX, artistY+46, colorWithAlpha(r.palette.TextMuted, 196), 28, false)
	}

	badgeY := artistY + 82
	metaRowH := r.drawBadges(innerX, badgeY, innerW)
	metaY := badgeY + metaRowH + 30
	r.drawMetadata(innerX, metaY, innerW)
	r.drawProgress(innerX, innerW)
	r.drawControls(innerX, innerW)
	r.drawVolume(innerX, innerW)
	drawText(r.content, "ANVU MUSIC", l.CardX+l.CardW-216, l.CardY+l.CardH-42, colorWithAlpha(r.palette.TextMuted, 72), 18, true)
}

func (r *renderer) drawBadges(x, y, maxW int) int {
	items := make([]string, 0, 8)
	if r.track.Explicit {
		items = append(items, "EXPLICIT")
	}
	if r.track.HQ {
		items = append(items, "HQ")
	}
	if r.track.Lossless {
		items = append(items, "LOSSLESS")
	}
	if r.track.DolbyAtmos {
		items = append(items, "DOLBY ATMOS")
	}
	if r.track.Lyrics {
		items = append(items, "LYRICS")
	}
	if r.track.Source != "" {
		items = append(items, strings.ToUpper(r.track.Source))
	}
	if len(items) == 0 {
		return 0
	}
	cx := x
	cy := y
	rowH := 38
	for _, item := range items {
		w := measureText(item, 16, true) + 34
		if cx+w > x+maxW {
			cx = x
			cy += rowH + 14
		}
		drawChip(r.content, cx, cy-24, item, w, colorWithAlpha(r.palette.TextPrimary, 24), colorWithAlpha(r.palette.TextPrimary, 224), 16, true)
		cx += w + r.layout.BadgeGap
	}
	return cy - y + rowH
}

func (r *renderer) drawMetadata(x, y, maxW int) {
	values := make([]string, 0, 4)
	if strings.TrimSpace(r.track.Views) != "" {
		values = append(values, r.track.Views)
	}
	if strings.TrimSpace(r.track.Duration) != "" {
		values = append(values, r.track.Duration)
	}
	if strings.TrimSpace(r.track.Quality) != "" {
		values = append(values, r.track.Quality)
	}
	if strings.TrimSpace(r.track.Source) != "" {
		values = append(values, r.track.Source)
	}
	if len(values) == 0 {
		return
	}
	line := strings.Join(values, "   •   ")
	block := layoutTitle(line, maxW, 2, 24, 18, false)
	for i, l := range block.Lines {
		drawText(r.content, l, x, y+i*block.LineHeight, colorWithAlpha(r.palette.TextMuted, 196), block.Size, false)
	}
}

func (r *renderer) drawProgress(x, width int) {
	l := r.layout
	waveY := l.WaveformY
	drawWaveform(r.content, x, waveY, width, l.WaveformH, colorWithAlpha(r.palette.TextPrimary, 38), r.palette.Accent)

	progress := resolveProgress(r.track)
	elapsed, total, remaining := resolveTimes(r.track, progress)
	trackY := l.ProgressY
	drawRoundedRect(r.content, x, trackY, width, l.ProgressH, l.ProgressH/2, colorWithAlpha(r.palette.TrackRemainder, 148))
	fillW := int(float64(width) * progress)
	if fillW < l.ProgressH {
		fillW = l.ProgressH
		if progress == 0 {
			fillW = 0
		}
	}
	if fillW > 0 {
		drawHorizontalGradientRoundedRect(r.content, x, trackY, fillW, l.ProgressH, l.ProgressH/2, colorWithAlpha(lightenColor(r.palette.Accent, 18), 232), colorWithAlpha(r.palette.Accent, 240))
		knobX := clampInt(x+l.ProgressH/2, x+width-l.ProgressH/2, x+fillW)
		drawCircleAA(r.content, knobX, trackY+l.ProgressH/2, 9, color.RGBA{255, 255, 255, 248})
		drawCircleBorder(r.content, knobX, trackY+l.ProgressH/2, 9, colorWithAlpha(r.palette.Accent, 86), 2)
	}
	labelY := trackY + 48
	drawText(r.content, elapsed, x, labelY, colorWithAlpha(r.palette.TextSecondary, 215), 22, false)
	totalW := measureText(total, 22, false)
	drawText(r.content, total, x+width/2-totalW/2, labelY, colorWithAlpha(r.palette.TextMuted, 170), 22, false)
	remainingW := measureText(remaining, 22, false)
	drawText(r.content, remaining, x+width-remainingW, labelY, colorWithAlpha(r.palette.TextSecondary, 215), 22, false)
}

func (r *renderer) drawControls(x, width int) {
	cy := r.layout.ControlsY
	center := x + width/2
	gap := 94
	iconColor := colorWithAlpha(r.palette.TextPrimary, 232)
	mutedColor := colorWithAlpha(r.palette.TextPrimary, 176)

	if r.track.ShuffleEnabled {
		drawIconShuffle(r.content, center-gap*2-48, cy, 18, mutedColor)
	}
	drawIconPrevious(r.content, center-gap, cy, 20, iconColor)
	drawControlButton(r.content, center, cy, r.layout.MainButtonRad, colorWithAlpha(r.palette.Accent, 238), colorWithAlpha(r.palette.Glow, 90))
	if r.track.IsPlaying {
		drawIconPause(r.content, center, cy, 22, color.RGBA{255, 255, 255, 255})
	} else {
		drawIconPlay(r.content, center+2, cy, 24, color.RGBA{255, 255, 255, 255})
	}
	drawIconNext(r.content, center+gap, cy, 20, iconColor)
	drawIconRepeat(r.content, center+gap*2+48, cy, 18, mutedColor)

	utilityX := x + width - 150
	if r.track.Lyrics {
		drawIconLyrics(r.content, utilityX, cy, 18, mutedColor)
		utilityX += 54
	}
	if r.track.QueueEnabled {
		drawIconQueue(r.content, utilityX, cy, 18, mutedColor)
	}
}

func (r *renderer) drawVolume(x, width int) {
	cy := r.layout.VolumeY
	sliderW := 190
	sliderX := x + width - sliderW
	drawIconVolume(r.content, sliderX-34, cy, 16, colorWithAlpha(r.palette.TextSecondary, 220))
	drawRoundedRect(r.content, sliderX, cy-7, sliderW, 10, 5, colorWithAlpha(r.palette.TrackRemainder, 156))
	fill := int(float64(sliderW) * clamp01(r.track.Volume))
	if fill > 0 {
		drawHorizontalGradientRoundedRect(r.content, sliderX, cy-7, fill, 10, 5, colorWithAlpha(lightenColor(r.palette.Accent, 20), 220), colorWithAlpha(r.palette.Accent, 226))
		knobX := clampInt(sliderX+5, sliderX+sliderW-5, sliderX+fill)
		drawCircleAA(r.content, knobX, cy-2, 7, color.RGBA{255, 255, 255, 244})
		drawCircleBorder(r.content, knobX, cy-2, 7, colorWithAlpha(r.palette.Accent, 76), 2)
	}
}

func layoutTitle(text string, maxWidth, maxLines, maxSize, minSize int, bold bool) textBlock {
	text = normalizeWhitespace(text)
	if text == "" {
		return textBlock{Lines: []string{""}, Size: minSize, LineHeight: lineHeight(minSize, bold), Height: lineHeight(minSize, bold)}
	}
	for size := maxSize; size >= minSize; size -= 2 {
		lines := wrapText(text, maxWidth, size, bold)
		if len(lines) <= maxLines {
			lh := lineHeight(size, bold)
			return textBlock{Lines: lines, Size: size, LineHeight: lh, Height: len(lines) * lh}
		}
	}
	lines := wrapText(text, maxWidth, minSize, bold)
	if len(lines) > maxLines {
		last := strings.Join(lines[maxLines-1:], " ")
		lines = append(lines[:maxLines-1], ellipsizeToWidth(last, maxWidth, minSize, bold))
	}
	lh := lineHeight(minSize, bold)
	clamped := lines[:minInt(len(lines), maxLines)]
	return textBlock{Lines: clamped, Size: minSize, LineHeight: lh, Height: len(clamped) * lh}
}

func wrapText(text string, maxWidth, size int, bold bool) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	tokens := make([]string, 0, len(words))
	for _, word := range words {
		if measureText(word, size, bold) <= maxWidth {
			tokens = append(tokens, word)
			continue
		}
		tokens = append(tokens, breakLongWord(word, maxWidth, size, bold)...)
	}
	lines := make([]string, 0, 2)
	current := ""
	for _, token := range tokens {
		candidate := token
		if current != "" {
			candidate = current + " " + token
		}
		if measureText(candidate, size, bold) <= maxWidth {
			current = candidate
			continue
		}
		if current != "" {
			lines = append(lines, current)
		}
		current = token
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func breakLongWord(word string, maxWidth, size int, bold bool) []string {
	runes := []rune(word)
	parts := make([]string, 0, 2)
	start := 0
	for start < len(runes) {
		end := start + 1
		for end <= len(runes) && measureText(string(runes[start:end]), size, bold) <= maxWidth {
			end++
		}
		if end == start+1 {
			parts = append(parts, string(runes[start:end]))
			start = end
			continue
		}
		parts = append(parts, string(runes[start:end-1]))
		start = end - 1
	}
	return parts
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}

func ellipsizeToWidth(text string, maxWidth, size int, bold bool) string {
	text = normalizeWhitespace(text)
	if text == "" || measureText(text, size, bold) <= maxWidth {
		return text
	}
	runes := []rune(text)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := strings.TrimSpace(string(runes)) + "…"
		if measureText(candidate, size, bold) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func lineHeight(size int, bold bool) int {
	return getFace(size, bold).Metrics().Height.Ceil() + int(float64(size)*0.10)
}

func drawText(dst *image.RGBA, text string, x, y int, c color.RGBA, size int, bold bool) {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: getFace(size, bold), Dot: fixed.P(x, y)}
	d.DrawString(text)
}

func measureText(text string, size int, bold bool) int {
	d := &font.Drawer{Face: getFace(size, bold)}
	return d.MeasureString(text).Ceil()
}

func resolveProgress(t TrackInfo) float64 {
	if t.Progress > 0 {
		return clamp01(t.Progress)
	}
	if elapsedSec, totalSec, ok := parseElapsedAndDuration(t.Elapsed, t.Duration); ok && totalSec > 0 {
		return clamp01(float64(elapsedSec) / float64(totalSec))
	}
	if strings.TrimSpace(t.Duration) != "" {
		return 0.38
	}
	return 0.28
}

func resolveTimes(t TrackInfo, progress float64) (string, string, string) {
	totalSec, ok := parseDuration(t.Duration)
	if !ok || totalSec <= 0 {
		elapsed := strings.TrimSpace(t.Elapsed)
		if elapsed == "" {
			elapsed = "00:00"
		}
		return elapsed, "--:--", "--:--"
	}
	elapsedSec := int(math.Round(float64(totalSec) * progress))
	if parsedElapsed, _, ok := parseElapsedAndDuration(t.Elapsed, t.Duration); ok {
		elapsedSec = parsedElapsed
	}
	if elapsedSec < 0 {
		elapsedSec = 0
	}
	if elapsedSec > totalSec {
		elapsedSec = totalSec
	}
	remaining := totalSec - elapsedSec
	return formatClock(elapsedSec), formatClock(totalSec), "-" + formatClock(remaining)
}

func parseElapsedAndDuration(elapsed, duration string) (int, int, bool) {
	e, ok1 := parseDuration(elapsed)
	t, ok2 := parseDuration(duration)
	return e, t, ok1 && ok2
}

func parseDuration(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, false
	}
	vals := make([]int, len(parts))
	for i, p := range parts {
		var v int
		if _, err := fmt.Sscanf(p, "%d", &v); err != nil {
			return 0, false
		}
		vals[i] = v
	}
	if len(vals) == 2 {
		return vals[0]*60 + vals[1], true
	}
	return vals[0]*3600 + vals[1]*60 + vals[2], true
}

func formatClock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, (sec/60)%60, sec%60)
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func extractPalette(src image.Image) palette {
	sample := coverCropResize(src, 72, 72)
	type bucket struct{ count, r, g, b int }
	buckets := map[int]*bucket{}
	var totalR, totalG, totalB, total int
	for y := 0; y < 72; y++ {
		for x := 0; x < 72; x++ {
			c := sample.RGBAAt(x, y)
			if c.A < 32 {
				continue
			}
			lum := luminance(c)
			sat := saturation(c)
			weight := 1
			if sat > 0.20 {
				weight += 2
			}
			if lum > 0.08 && lum < 0.92 {
				weight++
			}
			key := int(c.R>>4)<<8 | int(c.G>>4)<<4 | int(c.B>>4)
			bk := buckets[key]
			if bk == nil {
				bk = &bucket{}
				buckets[key] = bk
			}
			bk.count += weight
			bk.r += int(c.R) * weight
			bk.g += int(c.G) * weight
			bk.b += int(c.B) * weight
			totalR += int(c.R)
			totalG += int(c.G)
			totalB += int(c.B)
			total++
		}
	}
	dominant := color.RGBA{90, 120, 180, 255}
	bestScore := -1.0
	for _, bk := range buckets {
		if bk.count == 0 {
			continue
		}
		c := color.RGBA{uint8(bk.r / bk.count), uint8(bk.g / bk.count), uint8(bk.b / bk.count), 255}
		score := float64(bk.count) * (0.72 + saturation(c))
		lum := luminance(c)
		if lum < 0.08 || lum > 0.94 {
			score *= 0.65
		}
		if score > bestScore {
			bestScore = score
			dominant = c
		}
	}
	avg := dominant
	if total > 0 {
		avg = color.RGBA{uint8(totalR / total), uint8(totalG / total), uint8(totalB / total), 255}
	}
	accent := boostColor(mixColor(dominant, avg, 0.34), 1.06, 1.02)
	if saturation(accent) > 0.78 {
		accent = mixColor(accent, avg, 0.18)
	}
	if luminance(accent) < 0.18 {
		accent = lightenColor(accent, 22)
	}
	if luminance(accent) > 0.82 {
		accent = darkenColor(accent, 0.78)
	}
	glow := colorWithAlpha(mixColor(accent, color.RGBA{255, 255, 255, 255}, 0.16), 255)
	shadow := darkenColor(mixColor(accent, color.RGBA{14, 16, 20, 255}, 0.62), 0.72)
	useDarkText := luminance(avg) > 0.64
	textPrimary := color.RGBA{246, 247, 250, 255}
	textSecondary := color.RGBA{220, 223, 229, 255}
	textMuted := color.RGBA{177, 182, 191, 255}
	cardFill := color.RGBA{255, 255, 255, 42}
	cardStroke := color.RGBA{255, 255, 255, 72}
	trackRemainder := color.RGBA{255, 255, 255, 46}
	if useDarkText {
		textPrimary = color.RGBA{28, 32, 40, 255}
		textSecondary = color.RGBA{52, 58, 69, 255}
		textMuted = color.RGBA{81, 89, 102, 255}
		cardFill = color.RGBA{255, 255, 255, 92}
		cardStroke = color.RGBA{255, 255, 255, 132}
		trackRemainder = color.RGBA{30, 36, 48, 42}
	}
	return palette{
		Dominant:       dominant,
		Accent:         accent,
		Glow:           glow,
		Shadow:         shadow,
		TextPrimary:    textPrimary,
		TextSecondary:  textSecondary,
		TextMuted:      textMuted,
		CardFill:       cardFill,
		CardStroke:     cardStroke,
		TrackFill:      accent,
		TrackRemainder: trackRemainder,
		BackgroundTop:  color.RGBA{10, 12, 16, 96},
		BackgroundBot:  color.RGBA{4, 5, 8, 214},
		UseDarkText:    useDarkText,
	}
}

func newRGBA(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

func flattenLayers(layers ...*image.RGBA) *image.RGBA {
	if len(layers) == 0 {
		return newRGBA(internalW, internalH)
	}
	out := newRGBA(layers[0].Bounds().Dx(), layers[0].Bounds().Dy())
	for _, layer := range layers {
		if layer == nil {
			continue
		}
		stdDraw.Draw(out, out.Bounds(), layer, image.Point{}, stdDraw.Over)
	}
	return out
}

func copyRGBA(src *image.RGBA) *image.RGBA {
	out := newRGBA(src.Bounds().Dx(), src.Bounds().Dy())
	copy(out.Pix, src.Pix)
	return out
}

func resizeSmooth(src image.Image, w, h int) *image.RGBA {
	dst := newRGBA(w, h)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), stdDraw.Over, nil)
	return dst
}

func coverCropResize(src image.Image, w, h int) *image.RGBA {
	crop := focusCropRect(src, w, h)
	tmp := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	stdDraw.Draw(tmp, tmp.Bounds(), src, crop.Min, stdDraw.Src)
	return resizeSmooth(tmp, w, h)
}

func focusCropRect(src image.Image, w, h int) image.Rectangle {
	sb := src.Bounds()
	sw, sh := float64(sb.Dx()), float64(sb.Dy())
	targetRatio := float64(w) / float64(h)
	srcRatio := sw / sh
	crop := sb
	candidates := 11
	if srcRatio > targetRatio {
		cropW := int(sh * targetRatio)
		maxOffset := sb.Dx() - cropW
		bestScore := -1.0
		bestX := sb.Min.X + maxOffset/2
		for i := 0; i < candidates; i++ {
			t := float64(i) / float64(candidates-1)
			ox := sb.Min.X + int(float64(maxOffset)*t)
			rect := image.Rect(ox, sb.Min.Y, ox+cropW, sb.Max.Y)
			score := regionEnergy(src, rect) * (1 - math.Abs(t-0.5)*0.14)
			if score > bestScore {
				bestScore = score
				bestX = ox
			}
		}
		crop = image.Rect(bestX, sb.Min.Y, bestX+cropW, sb.Max.Y)
	} else if srcRatio < targetRatio {
		cropH := int(sw / targetRatio)
		maxOffset := sb.Dy() - cropH
		bestScore := -1.0
		bestY := sb.Min.Y + maxOffset/2
		for i := 0; i < candidates; i++ {
			t := float64(i) / float64(candidates-1)
			oy := sb.Min.Y + int(float64(maxOffset)*t)
			rect := image.Rect(sb.Min.X, oy, sb.Max.X, oy+cropH)
			score := regionEnergy(src, rect) * (1 - math.Abs(t-0.5)*0.14)
			if score > bestScore {
				bestScore = score
				bestY = oy
			}
		}
		crop = image.Rect(sb.Min.X, bestY, sb.Max.X, bestY+cropH)
	}
	return crop
}

func regionEnergy(src image.Image, rect image.Rectangle) float64 {
	stepX := maxInt(1, rect.Dx()/28)
	stepY := maxInt(1, rect.Dy()/28)
	var score float64
	for y := rect.Min.Y; y < rect.Max.Y; y += stepY {
		for x := rect.Min.X; x < rect.Max.X; x += stepX {
			base := color.RGBAModel.Convert(src.At(x, y)).(color.RGBA)
			lum := luminance(base)
			sat := saturation(base)
			score += sat*1.7 + (1 - math.Abs(lum-0.55))
			if x+stepX < rect.Max.X {
				right := color.RGBAModel.Convert(src.At(x+stepX, y)).(color.RGBA)
				score += math.Abs(lum-luminance(right)) * 1.4
			}
			if y+stepY < rect.Max.Y {
				down := color.RGBAModel.Convert(src.At(x, y+stepY)).(color.RGBA)
				score += math.Abs(lum-luminance(down)) * 1.2
			}
		}
	}
	return score
}

func gaussianBlur(img *image.RGBA, radius int) {
	if radius <= 0 {
		return
	}
	for i := 0; i < 3; i++ {
		boxBlurHorizontal(img, radius)
		boxBlurVertical(img, radius)
	}
}

func boxBlurHorizontal(img *image.RGBA, r int) {
	b := img.Bounds()
	buf := make([]color.RGBA, b.Dx())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		var sumR, sumG, sumB, sumA int
		count := 0
		for dx := -r; dx <= r; dx++ {
			x := clampInt(b.Min.X, b.Max.X-1, b.Min.X+dx)
			c := img.RGBAAt(x, y)
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
			sumA += int(c.A)
			count++
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			buf[x-b.Min.X] = color.RGBA{uint8(sumR / count), uint8(sumG / count), uint8(sumB / count), uint8(sumA / count)}
			addX := clampInt(b.Min.X, b.Max.X-1, x+r+1)
			remX := clampInt(b.Min.X, b.Max.X-1, x-r)
			add := img.RGBAAt(addX, y)
			rem := img.RGBAAt(remX, y)
			sumR += int(add.R) - int(rem.R)
			sumG += int(add.G) - int(rem.G)
			sumB += int(add.B) - int(rem.B)
			sumA += int(add.A) - int(rem.A)
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, buf[x-b.Min.X])
		}
	}
}

func boxBlurVertical(img *image.RGBA, r int) {
	b := img.Bounds()
	buf := make([]color.RGBA, b.Dy())
	for x := b.Min.X; x < b.Max.X; x++ {
		var sumR, sumG, sumB, sumA int
		count := 0
		for dy := -r; dy <= r; dy++ {
			y := clampInt(b.Min.Y, b.Max.Y-1, b.Min.Y+dy)
			c := img.RGBAAt(x, y)
			sumR += int(c.R)
			sumG += int(c.G)
			sumB += int(c.B)
			sumA += int(c.A)
			count++
		}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			buf[y-b.Min.Y] = color.RGBA{uint8(sumR / count), uint8(sumG / count), uint8(sumB / count), uint8(sumA / count)}
			addY := clampInt(b.Min.Y, b.Max.Y-1, y+r+1)
			remY := clampInt(b.Min.Y, b.Max.Y-1, y-r)
			add := img.RGBAAt(x, addY)
			rem := img.RGBAAt(x, remY)
			sumR += int(add.R) - int(rem.R)
			sumG += int(add.G) - int(rem.G)
			sumB += int(add.B) - int(rem.B)
			sumA += int(add.A) - int(rem.A)
		}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			img.SetRGBA(x, y, buf[y-b.Min.Y])
		}
	}
}

func applyTint(img *image.RGBA, tint color.RGBA, opacity float64) {
	alpha := clamp01(opacity)
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, mixColor(c, tint, alpha))
		}
	}
}

func overlayVerticalGradient(img *image.RGBA, top, bottom color.RGBA, from, to float64) {
	b := img.Bounds()
	startY := int(float64(b.Dy()) * from)
	endY := int(float64(b.Dy()) * to)
	if endY <= startY {
		return
	}
	for y := startY; y < endY; y++ {
		t := float64(y-startY) / float64(endY-startY)
		g := mixColor(top, bottom, t)
		for x := b.Min.X; x < b.Max.X; x++ {
			blendPixel(img, x, y, g)
		}
	}
}

func applyVignette(img *image.RGBA, strength float64) {
	b := img.Bounds()
	cx := float64(b.Dx()) / 2
	cy := float64(b.Dy()) / 2
	maxD := math.Hypot(cx, cy)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy) / maxD
			fade := clamp01((d - 0.35) / 0.65)
			alpha := uint8(255 * fade * strength)
			blendPixel(img, x, y, color.RGBA{0, 0, 0, alpha})
		}
	}
}

func addFilmGrain(img *image.RGBA, intensity float64) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			n := (hashNoise(x, y) - 0.5) * 2 * intensity * 255
			img.SetRGBA(x, y, color.RGBA{
				R: clampByte(float64(c.R) + n),
				G: clampByte(float64(c.G) + n),
				B: clampByte(float64(c.B) + n),
				A: c.A,
			})
		}
	}
}

func drawGlassPanel(dst, backdrop *image.RGBA, x, y, w, h, radius int, pal palette, fillStrength float64) {
	drawShadowRect(dst, x, y+18, w, h, radius, colorWithAlpha(color.RGBA{8, 10, 16, 255}, 54), 24)
	region := sampleRegion(backdrop, x, y, w, h)
	gaussianBlur(region, 22)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			cov := roundedCoverage(px, py, w, h, float64(radius))
			if cov <= 0 {
				continue
			}
			base := region.RGBAAt(px, py)
			lift := 0.10 + fillStrength*0.16 + math.Pow(1-clamp01(float64(py)/float64(h)*1.35), 2)*0.08
			glass := mixColor(base, color.RGBA{255, 255, 255, 255}, lift)
			glass = mixColor(glass, pal.Accent, 0.04)
			alpha := uint8((88 + fillStrength*84) * cov)
			blendPixel(dst, x+px, y+py, color.RGBA{glass.R, glass.G, glass.B, alpha})
		}
	}
	drawInnerHighlight(dst, x, y, w, h, radius, colorWithAlpha(color.RGBA{255, 255, 255, 255}, 34))
	drawRoundedRectBorderAA(dst, x, y, w, h, radius, pal.CardStroke, 2)
}

func sampleRegion(src *image.RGBA, x, y, w, h int) *image.RGBA {
	out := newRGBA(w, h)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			out.SetRGBA(px, py, src.RGBAAt(clampInt(0, src.Bounds().Dx()-1, x+px), clampInt(0, src.Bounds().Dy()-1, y+py)))
		}
	}
	return out
}

func drawChip(dst *image.RGBA, x, y int, label string, width int, fill, fg color.RGBA, size int, bold bool) {
	height := 34
	drawRoundedRect(dst, x, y, width, height, 17, fill)
	drawRoundedRectBorderAA(dst, x, y, width, height, 17, colorWithAlpha(color.RGBA{255, 255, 255, 255}, 28), 1)
	tx := x + (width-measureText(label, size, bold))/2
	drawText(dst, label, tx, y+23, fg, size, bold)
}

func drawWaveform(dst *image.RGBA, x, y, w, h int, base, accent color.RGBA) {
	bars := 58
	gap := 6
	barW := (w - gap*(bars-1)) / bars
	if barW < 2 {
		barW = 2
	}
	mid := y + h/2
	for i := 0; i < bars; i++ {
		phase := float64(i) / float64(maxInt(1, bars-1))
		amp := 0.20 + 0.80*(0.50*math.Abs(math.Sin(phase*math.Pi*3.0+0.6))+0.28*math.Abs(math.Sin(phase*math.Pi*7.2+0.1))+0.22*hashNoise(i*17, h))
		barH := int(float64(h) * clamp01(amp))
		if barH < 8 {
			barH = 8
		}
		bx := x + i*(barW+gap)
		by := mid - barH/2
		col := mixColor(base, accent, 0.14+0.34*phase)
		drawRoundedRect(dst, bx, by, barW, barH, barW/2, colorWithAlpha(col, 58))
	}
}

func drawVerifiedBadge(dst *image.RGBA, x, y, r int) {
	blue := color.RGBA{64, 153, 255, 255}
	drawCircleAA(dst, x, y, r, blue)
	drawLineAA(dst, float64(x-r/3), float64(y), float64(x-r/10), float64(y+r/4), 4, color.RGBA{255, 255, 255, 255})
	drawLineAA(dst, float64(x-r/12), float64(y+r/4), float64(x+r/2), float64(y-r/3), 4, color.RGBA{255, 255, 255, 255})
}

func drawControlButton(dst *image.RGBA, x, y, r int, fill, glow color.RGBA) {
	drawShadowCircle(dst, x, y+4, r+10, glow, 14)
	drawCircleAA(dst, x, y, r, fill)
	drawCircleBorder(dst, x, y, r, color.RGBA{255, 255, 255, 82}, 2)
}

func drawIconPlay(dst *image.RGBA, x, y, size int, c color.RGBA) {
	fillTriangle(dst, image.Point{x - size/3, y - size/2}, image.Point{x - size/3, y + size/2}, image.Point{x + size/2, y}, c)
}

func drawIconPause(dst *image.RGBA, x, y, size int, c color.RGBA) {
	w := size / 3
	h := size
	drawRoundedRect(dst, x-w-4, y-h/2, w, h, 4, c)
	drawRoundedRect(dst, x+4, y-h/2, w, h, 4, c)
}

func drawIconPrevious(dst *image.RGBA, x, y, size int, c color.RGBA) {
	drawRoundedRect(dst, x-size/2-6, y-size/2, 5, size, 2, c)
	fillTriangle(dst, image.Point{x - 4, y}, image.Point{x + size/2, y - size/2}, image.Point{x + size/2, y + size/2}, c)
	fillTriangle(dst, image.Point{x - size/2 + 2, y}, image.Point{x + 2, y - size/2}, image.Point{x + 2, y + size/2}, c)
}

func drawIconNext(dst *image.RGBA, x, y, size int, c color.RGBA) {
	drawRoundedRect(dst, x+size/2+1, y-size/2, 5, size, 2, c)
	fillTriangle(dst, image.Point{x + 4, y}, image.Point{x - size/2, y - size/2}, image.Point{x - size/2, y + size/2}, c)
	fillTriangle(dst, image.Point{x + size/2 - 2, y}, image.Point{x - 2, y - size/2}, image.Point{x - 2, y + size/2}, c)
}

func drawIconShuffle(dst *image.RGBA, x, y, size int, c color.RGBA) {
	drawLineAA(dst, float64(x-size), float64(y-size/2), float64(x+size), float64(y+size/2), 3, c)
	drawLineAA(dst, float64(x-size), float64(y+size/2), float64(x-size/5), float64(y+size/2), 3, c)
	drawLineAA(dst, float64(x+size/6), float64(y-size/2), float64(x+size), float64(y-size/2), 3, c)
	fillTriangle(dst, image.Point{x + size, y + size/2}, image.Point{x + size - 9, y + size/2 - 6}, image.Point{x + size - 9, y + size/2 + 6}, c)
	fillTriangle(dst, image.Point{x + size, y - size/2}, image.Point{x + size - 9, y - size/2 - 6}, image.Point{x + size - 9, y - size/2 + 6}, c)
	drawLineAA(dst, float64(x-size), float64(y+size/2), float64(x-size/5), float64(y+size/2), 3, c)
	drawLineAA(dst, float64(x-size/5), float64(y+size/2), float64(x+size), float64(y-size/2), 3, c)
}

func drawIconRepeat(dst *image.RGBA, x, y, size int, c color.RGBA) {
	drawLineAA(dst, float64(x-size), float64(y-size/3), float64(x+size/2), float64(y-size/3), 3, c)
	drawLineAA(dst, float64(x+size/2), float64(y-size/3), float64(x+size/2), float64(y+size/3), 3, c)
	drawLineAA(dst, float64(x+size), float64(y+size/3), float64(x-size/2), float64(y+size/3), 3, c)
	drawLineAA(dst, float64(x-size/2), float64(y+size/3), float64(x-size/2), float64(y-size/3), 3, c)
	fillTriangle(dst, image.Point{x + size/2, y - size/3}, image.Point{x + size/2 - 6, y - size/3 - 6}, image.Point{x + size/2 - 6, y - size/3 + 6}, c)
	fillTriangle(dst, image.Point{x - size/2, y + size/3}, image.Point{x - size/2 + 6, y + size/3 - 6}, image.Point{x - size/2 + 6, y + size/3 + 6}, c)
}

func drawIconLyrics(dst *image.RGBA, x, y, size int, c color.RGBA) {
	drawRoundedRectBorderAA(dst, x-size, y-size+2, size*2, size*2-4, 8, c, 2)
	drawLineAA(dst, float64(x-size/2), float64(y-size/3), float64(x+size/2), float64(y-size/3), 2, c)
	drawLineAA(dst, float64(x-size/2), float64(y), float64(x+size/2), float64(y), 2, c)
	drawLineAA(dst, float64(x-size/2), float64(y+size/3), float64(x+size/4), float64(y+size/3), 2, c)
}

func drawIconQueue(dst *image.RGBA, x, y, size int, c color.RGBA) {
	for i := -1; i <= 1; i++ {
		drawLineAA(dst, float64(x-size), float64(y+i*7), float64(x+size/2), float64(y+i*7), 2, c)
	}
	fillTriangle(dst, image.Point{x + size, y}, image.Point{x + size/2 + 3, y - 7}, image.Point{x + size/2 + 3, y + 7}, c)
}

func drawIconVolume(dst *image.RGBA, x, y, size int, c color.RGBA) {
	fillTriangle(dst, image.Point{x - size, y}, image.Point{x - size/3, y - size/2}, image.Point{x - size/3, y + size/2}, c)
	drawRoundedRect(dst, x-size/3, y-size/2, 7, size, 3, c)
	drawArc(dst, x+3, y, size/2, -0.65, 0.65, 2, c)
	drawArc(dst, x+4, y, size, -0.65, 0.65, 2, colorWithAlpha(c, 160))
}

func drawArc(dst *image.RGBA, cx, cy, r int, start, end float64, thickness float64, c color.RGBA) {
	steps := int(math.Max(12, float64(r)*2))
	prevX := float64(cx) + math.Cos(start)*float64(r)
	prevY := float64(cy) + math.Sin(start)*float64(r)
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		a := start + (end-start)*t
		x := float64(cx) + math.Cos(a)*float64(r)
		y := float64(cy) + math.Sin(a)*float64(r)
		drawLineAA(dst, prevX, prevY, x, y, thickness, c)
		prevX, prevY = x, y
	}
}

func drawReflection(dst *image.RGBA, x, y, w, h, radius int) {
	for py := 0; py < h/3; py++ {
		alpha := uint8(22 * math.Pow(1-float64(py)/float64(h/3), 2))
		for px := 0; px < w; px++ {
			cov := roundedCoverage(px, py, w, h, float64(radius))
			if cov > 0 {
				blendPixel(dst, x+px, y+py, color.RGBA{255, 255, 255, uint8(float64(alpha) * cov)})
			}
		}
	}
}

func drawInnerHighlight(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	band := 14
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			outer := roundedCoverage(px, py, w, h, float64(radius))
			inner := roundedCoverageInset(px, py, w, h, band, float64(radius))
			cov := outer - inner
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x+px, y+py, colorWithAlpha(c, uint8(float64(c.A)*cov)))
		}
	}
}

func drawShadowRect(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA, blur int) {
	tmp := newRGBA(dst.Bounds().Dx(), dst.Bounds().Dy())
	drawRoundedRect(tmp, x, y, w, h, radius, c)
	gaussianBlur(tmp, blur)
	stdDraw.Draw(dst, dst.Bounds(), tmp, image.Point{}, stdDraw.Over)
}

func drawRadialGlow(dst *image.RGBA, cx, cy, w, h int, c color.RGBA) {
	rx := float64(w) / 2
	ry := float64(h) / 2
	minX := cx - w/2
	maxX := cx + w/2
	minY := cy - h/2
	maxY := cy + h/2
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := (float64(x) - float64(cx)) / rx
			dy := (float64(y) - float64(cy)) / ry
			d := dx*dx + dy*dy
			if d >= 1 {
				continue
			}
			alpha := math.Pow(1-d, 2) * float64(c.A)
			blendPixel(dst, x, y, color.RGBA{c.R, c.G, c.B, uint8(alpha)})
		}
	}
}

func drawShadowCircle(dst *image.RGBA, cx, cy, r int, c color.RGBA, blur int) {
	tmp := newRGBA(dst.Bounds().Dx(), dst.Bounds().Dy())
	drawCircleAA(tmp, cx, cy, r, c)
	gaussianBlur(tmp, blur)
	stdDraw.Draw(dst, dst.Bounds(), tmp, image.Point{}, stdDraw.Over)
}

func drawCircleAA(dst *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r - 1; y <= cy+r+1; y++ {
		for x := cx - r - 1; x <= cx+r+1; x++ {
			d := math.Hypot(float64(x-cx), float64(y-cy))
			cov := clamp01(float64(r) - d + 0.75)
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x, y, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func drawCircleBorder(dst *image.RGBA, cx, cy, r int, c color.RGBA, thickness int) {
	for y := cy - r - thickness; y <= cy+r+thickness; y++ {
		for x := cx - r - thickness; x <= cx+r+thickness; x++ {
			d := math.Hypot(float64(x-cx), float64(y-cy))
			cov := clamp01(float64(thickness) - math.Abs(d-float64(r)) + 0.6)
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x, y, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func fillTriangle(dst *image.RGBA, p1, p2, p3 image.Point, c color.RGBA) {
	minX := minInt(p1.X, minInt(p2.X, p3.X))
	maxX := maxInt(p1.X, maxInt(p2.X, p3.X))
	minY := minInt(p1.Y, minInt(p2.Y, p3.Y))
	maxY := maxInt(p1.Y, maxInt(p2.Y, p3.Y))
	area := edge(p1, p2, p3)
	if area == 0 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			p := image.Point{x, y}
			w0 := edge(p2, p3, p)
			w1 := edge(p3, p1, p)
			w2 := edge(p1, p2, p)
			if (w0 >= 0 && w1 >= 0 && w2 >= 0) || (w0 <= 0 && w1 <= 0 && w2 <= 0) {
				blendPixel(dst, x, y, c)
			}
		}
	}
}

func edge(a, b, c image.Point) int {
	return (c.X-a.X)*(b.Y-a.Y) - (c.Y-a.Y)*(b.X-a.X)
}

func drawLineAA(dst *image.RGBA, x1, y1, x2, y2, thickness float64, c color.RGBA) {
	minX := int(math.Floor(math.Min(x1, x2) - thickness - 1))
	maxX := int(math.Ceil(math.Max(x1, x2) + thickness + 1))
	minY := int(math.Floor(math.Min(y1, y2) - thickness - 1))
	maxY := int(math.Ceil(math.Max(y1, y2) + thickness + 1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			d := distanceToSegment(float64(x)+0.5, float64(y)+0.5, x1, y1, x2, y2)
			cov := clamp01(thickness/2 - d + 1)
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x, y, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func distanceToSegment(px, py, x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	if dx == 0 && dy == 0 {
		return math.Hypot(px-x1, py-y1)
	}
	t := ((px-x1)*dx + (py-y1)*dy) / (dx*dx + dy*dy)
	t = clamp01(t)
	cx := x1 + t*dx
	cy := y1 + t*dy
	return math.Hypot(px-cx, py-cy)
}

func roundedContains(x, y, w, h, r float64) bool {
	if x < 0 || y < 0 || x > w || y > h {
		return false
	}
	if x < r && y < r {
		return dist2(x, y, r, r) <= r*r
	}
	if x > w-r && y < r {
		return dist2(x, y, w-r, r) <= r*r
	}
	if x < r && y > h-r {
		return dist2(x, y, r, h-r) <= r*r
	}
	if x > w-r && y > h-r {
		return dist2(x, y, w-r, h-r) <= r*r
	}
	return true
}

func roundedCoverage(x, y, w, h int, r float64) float64 {
	const n = 4
	hit := 0
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			px := float64(x) + (float64(sx)+0.5)/n
			py := float64(y) + (float64(sy)+0.5)/n
			if roundedContains(px, py, float64(w), float64(h), r) {
				hit++
			}
		}
	}
	return float64(hit) / float64(n*n)
}

func pasteRoundedAA(dst *image.RGBA, src image.Image, ox, oy, w, h, radius int) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cov := roundedCoverage(x, y, w, h, float64(radius))
			if cov <= 0 {
				continue
			}
			r, g, b, a := src.At(x, y).RGBA()
			blendPixel(dst, ox+x, oy+y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(float64(a>>8) * cov)})
		}
	}
}

func drawRoundedRect(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			cov := roundedCoverage(px, py, w, h, float64(radius))
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x+px, y+py, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func drawRoundedRectBorderAA(dst *image.RGBA, x, y, w, h, radius int, c color.RGBA, thickness int) {
	for py := -thickness; py < h+thickness; py++ {
		for px := -thickness; px < w+thickness; px++ {
			outer := roundedCoverage(px, py, w, h, float64(radius))
			inner := roundedCoverageInset(px, py, w, h, thickness, float64(radius))
			cov := outer - inner
			if cov <= 0 {
				continue
			}
			blendPixel(dst, x+px, y+py, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func roundedCoverageInset(x, y, w, h, inset int, radius float64) float64 {
	iw := w - inset*2
	ih := h - inset*2
	if iw <= 0 || ih <= 0 {
		return 0
	}
	return roundedCoverage(x-inset, y-inset, iw, ih, math.Max(0, radius-float64(inset)))
}

func drawHorizontalGradientRoundedRect(dst *image.RGBA, x, y, w, h, radius int, left, right color.RGBA) {
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			cov := roundedCoverage(px, py, w, h, float64(radius))
			if cov <= 0 {
				continue
			}
			t := float64(px) / math.Max(1, float64(w-1))
			c := mixColor(left, right, t)
			blendPixel(dst, x+px, y+py, color.RGBA{c.R, c.G, c.B, uint8(float64(c.A) * cov)})
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, src color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y || src.A == 0 {
		return
	}
	dst := img.RGBAAt(x, y)
	img.SetRGBA(x, y, blendOver(src, dst))
}

func blendOver(src, dst color.RGBA) color.RGBA {
	sa := float64(src.A) / 255
	da := float64(dst.A) / 255
	oa := sa + da*(1-sa)
	if oa == 0 {
		return color.RGBA{}
	}
	return color.RGBA{
		R: uint8((float64(src.R)*sa + float64(dst.R)*da*(1-sa)) / oa),
		G: uint8((float64(src.G)*sa + float64(dst.G)*da*(1-sa)) / oa),
		B: uint8((float64(src.B)*sa + float64(dst.B)*da*(1-sa)) / oa),
		A: uint8(oa * 255),
	}
}

func dist2(x1, y1, x2, y2 float64) float64 {
	dx, dy := x1-x2, y1-y2
	return dx*dx + dy*dy
}

func luminance(c color.RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

func saturation(c color.RGBA) float64 {
	r := float64(c.R) / 255
	g := float64(c.G) / 255
	b := float64(c.B) / 255
	maxV := math.Max(r, math.Max(g, b))
	minV := math.Min(r, math.Min(g, b))
	if maxV == 0 {
		return 0
	}
	return (maxV - minV) / maxV
}

func boostColor(c color.RGBA, satBoost, lumBoost float64) color.RGBA {
	r := float64(c.R)
	g := float64(c.G)
	b := float64(c.B)
	gray := (r + g + b) / 3
	r = gray + (r-gray)*satBoost
	g = gray + (g-gray)*satBoost
	b = gray + (b-gray)*satBoost
	return color.RGBA{clampByte(r * lumBoost), clampByte(g * lumBoost), clampByte(b * lumBoost), 255}
}

func mixColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp01(t)
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
		A: uint8(float64(a.A)*(1-t) + float64(b.A)*t),
	}
}

func darkenColor(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{clampByte(float64(c.R) * factor), clampByte(float64(c.G) * factor), clampByte(float64(c.B) * factor), c.A}
}

func lightenColor(c color.RGBA, by int) color.RGBA {
	return color.RGBA{lighten(c.R, by), lighten(c.G, by), lighten(c.B, by), c.A}
}

func colorWithAlpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func lighten(v uint8, amt int) uint8 {
	return uint8(clampInt(0, 255, int(v)+amt))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampInt(minV, maxV, v int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func hashNoise(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h%1000) / 1000
}

func downloadFile(url, dest string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("empty artwork URL")
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://music.youtube.com/")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoders := []func(io.Reader) (image.Image, error){jpeg.Decode, webp.Decode, png.Decode}
	var firstErr error
	for i, dec := range decoders {
		if _, err := f.Seek(0, 0); err != nil {
			return nil, err
		}
		img, err := dec(f)
		if err == nil {
			return img, nil
		}
		if i == 0 {
			firstErr = err
		}
	}
	return nil, firstErr
}

func dominantStops(src image.Image, count int) []color.RGBA {
	sample := coverCropResize(src, 36, 36)
	type stop struct {
		color color.RGBA
		score float64
	}
	stops := make([]stop, 0, 36*36)
	for y := 0; y < 36; y++ {
		for x := 0; x < 36; x++ {
			c := sample.RGBAAt(x, y)
			stops = append(stops, stop{color: c, score: saturation(c) + (1 - math.Abs(luminance(c)-0.5))})
		}
	}
	sort.Slice(stops, func(i, j int) bool { return stops[i].score > stops[j].score })
	out := make([]color.RGBA, 0, count)
	for _, s := range stops {
		good := true
		for _, existing := range out {
			if colorDistance(existing, s.color) < 46 {
				good = false
				break
			}
		}
		if good {
			out = append(out, s.color)
			if len(out) == count {
				break
			}
		}
	}
	return out
}

func colorDistance(a, b color.RGBA) float64 {
	dr := float64(int(a.R) - int(b.R))
	dg := float64(int(a.G) - int(b.G))
	db := float64(int(a.B) - int(b.B))
	return math.Sqrt(dr*dr + dg*dg + db*db)
}
