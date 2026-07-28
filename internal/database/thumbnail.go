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
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Laky-64/gologging"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/webp"
)

// ─── Cache ───────────────────────────────────────────────────────────────────

var (
	cacheDir   = "cache"
	httpClient = &http.Client{Timeout: 12 * time.Second}
)

func init() {
	_ = os.MkdirAll(cacheDir, 0o755)
}

// ─── Public Entry Point ───────────────────────────────────────────────────────

// TrackInfo holds the data needed to render a thumbnail.
type TrackInfo struct {
	VideoID  string // YouTube video ID (used for cache key)
	Title    string
	Artist   string
	Duration string // e.g. "3:45"
	Views    string // e.g. "1.2M views"
	Artwork  string // remote thumbnail URL

	// Elapsed and Progress drive the playback bar. Progress is 0..1; if it's
	// left at zero and Elapsed is empty, both are derived from Duration so
	// old call sites that don't set these still render a sane static bar.
	Elapsed  string  // e.g. "00:25" — pass the real playhead position
	Progress float64 // 0..1 — pass Elapsed/TotalDuration
}

// Generate returns a local path to the rendered thumbnail PNG.
// If a cached copy exists it is returned immediately.
func Generate(t TrackInfo) (string, error) {
	gologging.DebugF("[thumbgen] Generating thumbnail for %s (artwork: %.80s)", t.VideoID, t.Artwork)

	cachePath := filepath.Join(cacheDir, fmt.Sprintf("%s_anvu.png", t.VideoID))
	if _, err := os.Stat(cachePath); err == nil {
		gologging.DebugF("[thumbgen] Cache hit: %s", cachePath)
		return cachePath, nil
	}

	rawPath := filepath.Join(cacheDir, fmt.Sprintf("raw_%s.jpg", t.VideoID))
	if err := downloadFile(t.Artwork, rawPath); err != nil {
		gologging.WarnF("[thumbgen] download failed: %v", err)
		return "", fmt.Errorf("failed to download artwork: %w", err)
	}
	defer os.Remove(rawPath)

	src, err := loadImage(rawPath)
	if err != nil {
		gologging.WarnF("[thumbgen] loadImage failed: %v", err)
		return "", fmt.Errorf("loadImage failed: %w", err)
	}
	gologging.DebugF("[thumbgen] Image loaded (%dx%d)", src.Bounds().Dx(), src.Bounds().Dy())

	out, err := render(src, t)
	if err != nil {
		gologging.ErrorF("[thumbgen] render failed: %v", err)
		return "", fmt.Errorf("render failed: %w", err)
	}

	f, err := os.Create(cachePath)
	if err != nil {
		gologging.ErrorF("[thumbgen] os.Create failed: %v", err)
		return "", fmt.Errorf("os.Create failed: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, out); err != nil {
		os.Remove(cachePath)
		gologging.ErrorF("[thumbgen] png.Encode failed: %v", err)
		return "", fmt.Errorf("png.Encode failed: %w", err)
	}

	gologging.InfoF("[thumbgen] Thumbnail saved: %s", cachePath)
	return cachePath, nil
}

// ClearCache removes all generated thumbnails from the cache directory.
func ClearCache() {
	matches, _ := filepath.Glob(filepath.Join(cacheDir, "*_anvu.png"))
	for _, m := range matches {
		os.Remove(m)
	}
}

// ─── Canvas ──────────────────────────────────────────────────────────────────

const (
	W = 1280
	H = 720
)

// Accent palette — swap these two lines to switch the whole theme.
var (
	accentColor = color.RGBA{30, 215, 96, 255} // Spotify green
	// accentColor = color.RGBA{252, 61, 122, 255} // Apple Music pink — swap in if preferred
)

// ─── Fonts ───────────────────────────────────────────────────────────────────

var (
	fontMu       sync.Mutex
	regularCache = map[int]font.Face{}
	boldCache    = map[int]font.Face{}
	regularTTF   *opentype.Font
	boldTTF      *opentype.Font
)

func parseFonts() {
	if regularTTF == nil {
		f, err := opentype.Parse(goregular.TTF)
		if err != nil {
			gologging.ErrorF("[thumbgen] failed to parse regular TTF: %v", err)
		} else {
			regularTTF = f
		}
	}
	if boldTTF == nil {
		f, err := opentype.Parse(gobold.TTF)
		if err != nil {
			gologging.ErrorF("[thumbgen] failed to parse bold TTF: %v", err)
		} else {
			boldTTF = f
		}
	}
}

func getFace(size int, bold bool) font.Face {
	fontMu.Lock()
	defer fontMu.Unlock()
	parseFonts()

	cache := regularCache
	src := regularTTF
	if bold {
		cache = boldCache
		src = boldTTF
	}

	if f, ok := cache[size]; ok {
		return f
	}
	if src == nil {
		return basicfont.Face7x13
	}
	face, err := opentype.NewFace(src, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		gologging.ErrorF("[thumbgen] failed to create face size=%d bold=%v: %v", size, bold, err)
		return basicfont.Face7x13
	}
	cache[size] = face
	return face
}

// drawText renders a single line of text with its baseline at (x, y).
func drawText(img *image.RGBA, text string, x, y int, c color.RGBA, size int, bold bool) {
	face := getFace(size, bold)
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

// measureText returns the pixel width text would occupy at the given size.
func measureText(text string, size int, bold bool) int {
	face := getFace(size, bold)
	d := &font.Drawer{Face: face}
	return d.MeasureString(text).Ceil()
}

// ─── Layout / Render ─────────────────────────────────────────────────────────

func render(src image.Image, t TrackInfo) (image.Image, error) {
	bg := newRGBA(W, H)

	// 1. Blurred, darkened backdrop filling the whole canvas.
	backdrop := resizeImageSmooth(src, W, H)
	draw.Draw(bg, bg.Bounds(), backdrop, image.Point{}, draw.Src)
	gaussianBlur(bg, 32)
	darken(bg, 0.34)

	// 2. Album art frame — left side, rounded square with soft drop shadow.
	frameW, frameH := 460, 460
	frameX := 100
	frameY := (H - frameH) / 2

	album := resizeImageSmooth(src, frameW, frameH)

	drawGlow(bg, frameX-24, frameY-24, frameW+48, frameH+48, 44, color.RGBA{0, 0, 0, 170})
	pasteRoundedAA(bg, album, frameX, frameY, frameW, frameH, 36)
	drawRoundedRectBorderAA(bg, frameX, frameY, frameW, frameH, 36, color.RGBA{255, 255, 255, 90}, 3)

	// 3. Frosted glass card — right side.
	cardX1 := frameX + frameW + 60
	cardY1 := frameY
	cardX2 := W - 70
	cardY2 := frameY + frameH
	drawFrostedGlassCard(bg, cardX1, cardY1, cardX2, cardY2, 32)

	// 4. Text + progress bar inside the card.
	pad := 42
	textX := cardX1 + pad
	maxTextW := (cardX2 - pad) - textX

	titleColor := color.RGBA{255, 255, 255, 255}
	artistColor := color.RGBA{190, 190, 198, 235} // pulled down from the old 215/215/220 — reads
	// as a clearer step below the title now instead of a near-white blur into it
	metaColor := color.RGBA{150, 150, 158, 200}
	timeColor := color.RGBA{225, 225, 228, 220}

	titleSize, artistSize, metaSize, timeSize := 34, 21, 18, 16

	titleY := cardY1 + 64
	artistY := titleY + 44
	viewsY := artistY + 36
	barY := cardY2 - 92

	title := fitText(t.Title, titleSize, true, maxTextW)
	drawText(bg, title, textX, titleY, titleColor, titleSize, true)

	artistLine := fitText("By "+t.Artist, artistSize, false, maxTextW)
	drawText(bg, artistLine, textX, artistY, artistColor, artistSize, false)

	viewsLine := fitText("Views: "+t.Views, metaSize, false, maxTextW)
	drawText(bg, viewsLine, textX, viewsY, metaColor, metaSize, false)

	// Progress bar — now driven by t.Progress/t.Elapsed instead of a fixed
	// preview value, with a sane fallback so old callers don't break.
	progress := t.Progress
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	barW := maxTextW
	barH := 6
	drawRoundedRect(bg, textX, barY, barW, barH, 3, color.RGBA{255, 255, 255, 45})

	filledW := int(float64(barW) * progress)
	if filledW > 0 {
		drawRoundedRect(bg, textX, barY, filledW, barH, 3, accentColor)
	}
	drawCircleAA(bg, textX+filledW, barY+barH/2, 8, color.RGBA{255, 255, 255, 255})

	elapsed := t.Elapsed
	if elapsed == "" {
		elapsed = elapsedFromProgress(t.Duration, progress)
	}
	drawText(bg, elapsed, textX, barY+34, timeColor, timeSize, false)

	duration := t.Duration
	if duration == "" {
		duration = "--:--"
	}
	durW := measureText(duration, timeSize, false)
	drawText(bg, duration, textX+barW-durW, barY+34, timeColor, timeSize, false)

	return bg, nil
}

// elapsedFromProgress derives an "mm:ss" elapsed label from a duration string
// (format "m:ss" or "mm:ss", matching TrackInfo.Duration) and a 0..1
// progress fraction. Falls back to "00:00" if duration can't be parsed —
// this only fires when a caller sets Progress but not Elapsed directly.
func elapsedFromProgress(duration string, progress float64) string {
	var mins, secs int
	if _, err := fmt.Sscanf(duration, "%d:%d", &mins, &secs); err != nil {
		return "00:00"
	}
	totalSecs := mins*60 + secs
	elapsedSecs := int(float64(totalSecs) * progress)
	return fmt.Sprintf("%02d:%02d", elapsedSecs/60, elapsedSecs%60)
}

// fitText truncates text with an ellipsis until it fits within maxW pixels
// at the given font size/weight. This replaces naive rune-count truncation,
// which doesn't account for variable glyph widths.
func fitText(s string, size int, bold bool, maxW int) string {
	if measureText(s, size, bold) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "…"
		if measureText(candidate, size, bold) <= maxW {
			return candidate
		}
	}
	return "…"
}

// ─── Image Ops ───────────────────────────────────────────────────────────────

func newRGBA(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

// resizeImageSmooth performs bilinear resampling (much smoother than nearest-
// neighbour, avoids the blocky look on both the backdrop and the album art).
func resizeImageSmooth(src image.Image, newW, newH int) *image.RGBA {
	dst := newRGBA(newW, newH)
	sb := src.Bounds()
	srcW, srcH := float64(sb.Dx()), float64(sb.Dy())

	// Pre-fetch source into a plain RGBA buffer for fast random access.
	srcRGBA := toRGBA(src)

	for y := 0; y < newH; y++ {
		fy := (float64(y)+0.5)*srcH/float64(newH) - 0.5
		y0 := int(math.Floor(fy))
		wy := fy - float64(y0)
		for x := 0; x < newW; x++ {
			fx := (float64(x)+0.5)*srcW/float64(newW) - 0.5
			x0 := int(math.Floor(fx))
			wx := fx - float64(x0)

			c00 := sampleClamped(srcRGBA, x0, y0)
			c10 := sampleClamped(srcRGBA, x0+1, y0)
			c01 := sampleClamped(srcRGBA, x0, y0+1)
			c11 := sampleClamped(srcRGBA, x0+1, y0+1)

			r := lerp2D(c00.R, c10.R, c01.R, c11.R, wx, wy)
			g := lerp2D(c00.G, c10.G, c01.G, c11.G, wx, wy)
			bch := lerp2D(c00.B, c10.B, c01.B, c11.B, wx, wy)
			a := lerp2D(c00.A, c10.A, c01.A, c11.A, wx, wy)
			dst.SetRGBA(x, y, color.RGBA{r, g, bch, a})
		}
	}
	return dst
}

func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok {
		return r
	}
	b := src.Bounds()
	out := newRGBA(b.Dx(), b.Dy())
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	return out
}

func sampleClamped(src *image.RGBA, x, y int) color.RGBA {
	b := src.Bounds()
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= b.Dx() {
		x = b.Dx() - 1
	}
	if y >= b.Dy() {
		y = b.Dy() - 1
	}
	return src.RGBAAt(b.Min.X+x, b.Min.Y+y)
}

func lerp2D(v00, v10, v01, v11 uint8, wx, wy float64) uint8 {
	top := float64(v00)*(1-wx) + float64(v10)*wx
	bot := float64(v01)*(1-wx) + float64(v11)*wx
	res := top*(1-wy) + bot*wy
	if res < 0 {
		res = 0
	}
	if res > 255 {
		res = 255
	}
	return uint8(res)
}

// gaussianBlur applies a 3-pass box blur (a good Gaussian approximation).
func gaussianBlur(img *image.RGBA, radius int) {
	if radius <= 0 {
		return
	}
	for pass := 0; pass < 3; pass++ {
		boxBlurH(img, radius)
		boxBlurV(img, radius)
	}
}

func boxBlurH(img *image.RGBA, r int) {
	b := img.Bounds()
	buf := make([]color.RGBA, b.Dx())
	for y := b.Min.Y; y < b.Max.Y; y++ {
		var sumR, sumG, sumB, sumA int64
		count := int64(0)
		for dx := 0; dx <= r && dx < b.Dx(); dx++ {
			c := img.RGBAAt(b.Min.X+dx, y)
			sumR += int64(c.R)
			sumG += int64(c.G)
			sumB += int64(c.B)
			sumA += int64(c.A)
			count++
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			buf[x-b.Min.X] = color.RGBA{
				R: uint8(sumR / count), G: uint8(sumG / count),
				B: uint8(sumB / count), A: uint8(sumA / count),
			}
			addX := x + r + 1
			remX := x - r
			if addX < b.Max.X {
				c := img.RGBAAt(addX, y)
				sumR += int64(c.R)
				sumG += int64(c.G)
				sumB += int64(c.B)
				sumA += int64(c.A)
				count++
			}
			if remX >= b.Min.X {
				c := img.RGBAAt(remX, y)
				sumR -= int64(c.R)
				sumG -= int64(c.G)
				sumB -= int64(c.B)
				sumA -= int64(c.A)
				count--
			}
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, buf[x-b.Min.X])
		}
	}
}

func boxBlurV(img *image.RGBA, r int) {
	b := img.Bounds()
	buf := make([]color.RGBA, b.Dy())
	for x := b.Min.X; x < b.Max.X; x++ {
		var sumR, sumG, sumB, sumA int64
		count := int64(0)
		for dy := 0; dy <= r && dy < b.Dy(); dy++ {
			c := img.RGBAAt(x, b.Min.Y+dy)
			sumR += int64(c.R)
			sumG += int64(c.G)
			sumB += int64(c.B)
			sumA += int64(c.A)
			count++
		}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			buf[y-b.Min.Y] = color.RGBA{
				R: uint8(sumR / count), G: uint8(sumG / count),
				B: uint8(sumB / count), A: uint8(sumA / count),
			}
			addY := y + r + 1
			remY := y - r
			if addY < b.Max.Y {
				c := img.RGBAAt(x, addY)
				sumR += int64(c.R)
				sumG += int64(c.G)
				sumB += int64(c.B)
				sumA += int64(c.A)
				count++
			}
			if remY >= b.Min.Y {
				c := img.RGBAAt(x, remY)
				sumR -= int64(c.R)
				sumG -= int64(c.G)
				sumB -= int64(c.B)
				sumA -= int64(c.A)
				count--
			}
		}
		for y := b.Min.Y; y < b.Max.Y; y++ {
			img.SetRGBA(x, y, buf[y-b.Min.Y])
		}
	}
}

func darken(img *image.RGBA, factor float64) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R) * factor),
				G: uint8(float64(c.G) * factor),
				B: uint8(float64(c.B) * factor),
				A: c.A,
			})
		}
	}
}

func lighten(v uint8, amt int) uint8 {
	n := int(v) + amt
	if n > 255 {
		n = 255
	}
	if n < 0 {
		n = 0
	}
	return uint8(n)
}

// ─── Rounded Rects / Anti-aliasing ────────────────────────────────────────────

// roundedContains reports whether point (x,y) lies inside a w×h rect with
// corner radius r, treating (0,0) as the rect's top-left.
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

func dist2(x1, y1, x2, y2 float64) float64 {
	dx, dy := x1-x2, y1-y2
	return dx*dx + dy*dy
}

// roundedCoverage returns 0..1 anti-aliased edge coverage via 4×4 supersampling.
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

// pasteRoundedAA composites src onto dst at (ox, oy) with anti-aliased rounded corners.
func pasteRoundedAA(dst *image.RGBA, src image.Image, ox, oy, w, h, radius int) {
	rf := float64(radius)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cov := roundedCoverage(x, y, w, h, rf)
			if cov <= 0 {
				continue
			}
			sr, sg, sb, sa := src.At(x, y).RGBA()
			a := uint8((float64(sa>>8) / 255) * 255 * cov)
			sc := color.RGBA{uint8(sr >> 8), uint8(sg >> 8), uint8(sb >> 8), a}
			blendPixel(dst, ox+x, oy+y, sc)
		}
	}
}

func drawRoundedRect(img *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	rf := float64(radius)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			cov := roundedCoverage(px, py, w, h, rf)
			if cov <= 0 {
				continue
			}
			cc := c
			cc.A = uint8(float64(c.A) * cov)
			blendPixel(img, x+px, y+py, cc)
		}
	}
}

// drawRoundedRectBorderAA draws an anti-aliased ring by subtracting an inset
// rounded-rect's coverage from the outer one, so only the border band gets painted.
func drawRoundedRectBorderAA(img *image.RGBA, x, y, w, h, radius int, c color.RGBA, thickness int) {
	rf := float64(radius)
	inner := rf - float64(thickness)
	for py := -thickness; py < h+thickness; py++ {
		for px := -thickness; px < w+thickness; px++ {
			outerCov := roundedCoverage(px, py, w, h, rf)
			innerCov := roundedCoverage(px, py, w, h, math.Max(inner, 0))
			cov := outerCov - innerCov
			if cov <= 0 {
				continue
			}
			cc := c
			cc.A = uint8(float64(c.A) * cov)
			blendPixel(img, x+px, y+py, cc)
		}
	}
}

func drawGlow(img *image.RGBA, x, y, w, h, blurR int, c color.RGBA) {
	tmp := newRGBA(img.Bounds().Dx(), img.Bounds().Dy())
	drawRoundedRect(tmp, x, y, w, h, 40, c)
	gaussianBlur(tmp, blurR)
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			gc := tmp.RGBAAt(px, py)
			if gc.A == 0 {
				continue
			}
			blendPixel(img, px, py, gc)
		}
	}
}

func drawCircleAA(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	for y := cy - r - 1; y <= cy+r+1; y++ {
		for x := cx - r - 1; x <= cx+r+1; x++ {
			d := math.Sqrt(float64((x-cx)*(x-cx) + (y-cy)*(y-cy)))
			cov := clamp01(float64(r) - d + 0.5)
			if cov <= 0 {
				continue
			}
			cc := c
			cc.A = uint8(float64(c.A) * cov)
			blendPixel(img, x, y, cc)
		}
	}
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

// ─── Frosted Glass ───────────────────────────────────────────────────────────
//
// A believable glass card needs four things beyond a blurred+lightened
// backdrop sample: (1) grain, because real frosted glass diffuses light
// unevenly rather than producing a perfectly flat gradient; (2) a soft
// highlight biased toward the top edge, mimicking a single overhead light
// source; (3) a slightly darker inset ring just inside the border, so the
// glass reads as having *thickness* rather than being a flat cutout; and
// (4) the light outer rim your original already had. Each is cheap and
// layers on top of the existing blur-and-composite approach — nothing here
// replaces your box-blur pipeline, it just adds three passes after it.

// noiseSeed keeps grain deterministic per-pixel without a PRNG dependency;
// a plain hash of (x,y) is enough since we only need visual dither, not
// statistical randomness.
func noiseSeed(x, y int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	h = h ^ (h >> 16)
	return float64(h%1000) / 1000.0 // 0..1
}

// drawFrostedGlassCard renders a genuine "frosted glass" panel: it samples the
// already-blurred backdrop under the card region, blurs that region further,
// lightens it, adds fine grain and a top-biased highlight, composites it back
// with a translucent alpha, then finishes with an inset shadow ring plus a
// 1px light outer border — this is what actually produces the Apple/Spotify
// glass look, rather than a single flat semi-transparent rectangle.
func drawFrostedGlassCard(img *image.RGBA, x1, y1, x2, y2, radius int) {
	w, h := x2-x1, y2-y1
	if w <= 0 || h <= 0 {
		return
	}

	region := newRGBA(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			region.SetRGBA(x, y, img.RGBAAt(x1+x, y1+y))
		}
	}
	gaussianBlur(region, 18)

	rf := float64(radius)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			cov := roundedCoverage(x, y, w, h, rf)
			if cov <= 0 {
				continue
			}
			rc := region.RGBAAt(x, y)

			// Grain: +/-4 levels per channel, cheap and deterministic.
			grain := int(noiseSeed(x1+x, y1+y)*8) - 4

			// Top-biased highlight: strongest near y=0, fading to ~40% of
			// its peak by mid-card, gone by the lower third. This is what
			// makes the panel look lit from above instead of uniformly
			// milky, which was the single biggest tell that the old
			// version was "blur + lighten" rather than "glass."
			highlightT := 1.0 - clamp01(float64(y)/(float64(h)*0.65))
			highlight := int(26 * highlightT * highlightT)

			glass := color.RGBA{
				R: lighten(rc.R, 22+highlight+grain),
				G: lighten(rc.G, 22+highlight+grain),
				B: lighten(rc.B, 26+highlight+grain),
				A: uint8(185 * cov),
			}
			blendPixel(img, x1+x, y1+y, glass)
		}
	}

	// Inset shadow ring: a soft dark band just inside the edge gives the
	// glass apparent thickness. Same subtract-two-coverages trick as
	// drawRoundedRectBorderAA, just pulled in from the edge and blurred
	// mentally by using a wide, low-alpha band rather than a hard 1px line.
	insetBand := math.Max(float64(radius)*0.4, 6)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			outerCov := roundedCoverage(px, py, w, h, rf)
			innerCov := roundedCoverage(px, py, w, h, math.Max(rf-insetBand, 0))
			cov := outerCov - innerCov
			if cov <= 0 {
				continue
			}
			cc := color.RGBA{0, 0, 0, uint8(28 * cov)}
			blendPixel(img, x1+px, y1+py, cc)
		}
	}

	drawRoundedRectBorderAA(img, x1, y1, w, h, radius, color.RGBA{255, 255, 255, 60}, 1)
}

// ─── Compositing ─────────────────────────────────────────────────────────────

func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	if c.A == 0 {
		return
	}
	dst := img.RGBAAt(x, y)
	img.SetRGBA(x, y, blendOver(c, dst))
}

func blendOver(src, dst color.RGBA) color.RGBA {
	sa := float64(src.A) / 255.0
	da := float64(dst.A) / 255.0
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

// ─── I/O helpers ─────────────────────────────────────────────────────────────

func downloadFile(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.youtube.com/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
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

	img, err := jpeg.Decode(f)
	if err == nil {
		return img, nil
	}
	if _, seekErr := f.Seek(0, 0); seekErr != nil {
		return nil, seekErr
	}
	img, err = webp.Decode(f)
	if err == nil {
		return img, nil
	}
	if _, seekErr := f.Seek(0, 0); seekErr != nil {
		return nil, seekErr
	}
	return png.Decode(f)
}
