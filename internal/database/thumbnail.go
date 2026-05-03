/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package thumbgen

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
)

// ─── Cache ───────────────────────────────────────────────────────────────────

var (
	cacheDir  = "cache"
	cacheMu   sync.Mutex
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
}

// Generate returns a local path to the rendered thumbnail PNG.
// If a cached copy exists it is returned immediately.
// On any render failure it falls back to downloading the raw artwork.
func Generate(t TrackInfo) (string, error) {
	cachePath := filepath.Join(cacheDir, fmt.Sprintf("%s_anvu.png", t.VideoID))
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	// Download raw artwork
	rawPath := filepath.Join(cacheDir, fmt.Sprintf("raw_%s.jpg", t.VideoID))
	if err := downloadFile(t.Artwork, rawPath); err != nil {
		return "", fmt.Errorf("failed to download artwork: %w", err)
	}
	defer os.Remove(rawPath)

	src, err := loadImage(rawPath)
	if err != nil {
		return t.Artwork, nil // fallback to URL
	}

	out, err := render(src, t)
	if err != nil {
		gologging.ErrorF("[thumbgen] render failed: %v", err)
		return t.Artwork, nil // fallback to URL
	}

	f, err := os.Create(cachePath)
	if err != nil {
		return t.Artwork, nil
	}
	defer f.Close()

	if err := png.Encode(f, out); err != nil {
		os.Remove(cachePath)
		return t.Artwork, nil
	}

	return cachePath, nil
}

// ClearCache removes all generated thumbnails from the cache directory.
func ClearCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	matches, _ := filepath.Glob(filepath.Join(cacheDir, "*_anvu.png"))
	for _, m := range matches {
		os.Remove(m)
	}
}

// ─── Renderer ────────────────────────────────────────────────────────────────

const (
	W = 1280
	H = 720
)

func render(src image.Image, t TrackInfo) (image.Image, error) {
	// 1. Create blurred + darkened background
	bg := newRGBA(W, H)
	scaled := resizeImage(src, W, H)
	draw.Draw(bg, bg.Bounds(), scaled, image.Point{}, draw.Src)
	gaussianBlur(bg, 28)
	darken(bg, 0.38)

	// 2. Album art frame (left side)
	frameW, frameH := 450, 450
	frameX := 100
	frameY := (H - frameH) / 2

	album := resizeImage(src, frameW, frameH)

	// Glow shadow behind frame
	drawGlow(bg, frameX-18, frameY-18, frameW+36, frameH+36, 40, color.RGBA{0, 0, 0, 160})

	// Paste rounded album art
	pasteRounded(bg, album, frameX, frameY, frameW, frameH, 40)

	// White border overlay on frame
	drawRoundedRectBorder(bg, frameX, frameY, frameW, frameH, 40, color.RGBA{255, 255, 255, 80}, 5)

	// 3. Glass card (right side)
	textX := 620
	glassX1, glassY1 := textX-40, frameY
	glassX2, glassY2 := W-60, frameY+frameH
	drawGlassCard(bg, glassX1, glassY1, glassX2, glassY2, 28, 22)

	// 4. Text
	titleColor  := color.RGBA{255, 255, 255, 255}
	artistColor := color.RGBA{200, 200, 200, 230}
	metaColor   := color.RGBA{180, 180, 180, 200}
	timeColor   := color.RGBA{255, 255, 255, 200}
	accentColor := color.RGBA{0, 200, 255, 255}

	titleY  := frameY + 40
	artistY := titleY + 72
	viewsY  := artistY + 52
	barY    := frameY + 320

	// Title (big)
	title := truncate(t.Title, 28)
	drawTextSimple(bg, title, textX, titleY, titleColor, 2) // weight 2 = bold-ish

	// Artist
	artist := truncate("By "+t.Artist, 32)
	drawTextSimple(bg, artist, textX, artistY, artistColor, 1)

	// Views
	drawTextSimple(bg, "Views: "+t.Views, textX, viewsY, metaColor, 1)

	// Progress bar  ─────────────────────────────────────────────
	barW := 500
	barH := 8

	// Track rail (dim)
	drawRoundedRect(bg, textX, barY, barW, barH, 4, color.RGBA{255, 255, 255, 50})

	// Filled portion (40%)
	filledW := int(float64(barW) * 0.4)
	drawRoundedRect(bg, textX, barY, filledW, barH, 4, accentColor)

	// Thumb circle
	cx := textX + filledW
	cy := barY + barH/2
	drawCircle(bg, cx, cy, 10, color.RGBA{255, 255, 255, 255})

	// Time labels
	drawTextSimple(bg, "00:25", textX, barY+22, timeColor, 1)
	durLabel := truncate(t.Duration, 10)
	drawTextSimple(bg, durLabel, textX+barW-55, barY+22, timeColor, 1)

	return bg, nil
}

// ─── Drawing Primitives ───────────────────────────────────────────────────────

func newRGBA(w, h int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

// gaussianBlur applies a simple box-blur approximation (3 passes → Gaussian-like).
func gaussianBlur(img *image.RGBA, radius int) {
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
		// seed window
		for dx := 0; dx <= r && dx < b.Dx(); dx++ {
			c := img.RGBAAt(b.Min.X+dx, y)
			sumR += int64(c.R); sumG += int64(c.G)
			sumB += int64(c.B); sumA += int64(c.A)
			count++
		}
		for x := b.Min.X; x < b.Max.X; x++ {
			buf[x-b.Min.X] = color.RGBA{
				R: uint8(sumR / count), G: uint8(sumG / count),
				B: uint8(sumB / count), A: uint8(sumA / count),
			}
			// slide window
			addX := x + r + 1
			remX := x - r
			if addX < b.Max.X {
				c := img.RGBAAt(addX, y)
				sumR += int64(c.R); sumG += int64(c.G)
				sumB += int64(c.B); sumA += int64(c.A)
				count++
			}
			if remX >= b.Min.X {
				c := img.RGBAAt(remX, y)
				sumR -= int64(c.R); sumG -= int64(c.G)
				sumB -= int64(c.B); sumA -= int64(c.A)
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
			sumR += int64(c.R); sumG += int64(c.G)
			sumB += int64(c.B); sumA += int64(c.A)
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
				sumR += int64(c.R); sumG += int64(c.G)
				sumB += int64(c.B); sumA += int64(c.A)
				count++
			}
			if remY >= b.Min.Y {
				c := img.RGBAAt(x, remY)
				sumR -= int64(c.R); sumG -= int64(c.G)
				sumB -= int64(c.B); sumA -= int64(c.A)
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

// resizeImage does a nearest-neighbour resize (fast, good enough for blurred bg).
func resizeImage(src image.Image, newW, newH int) *image.RGBA {
	dst := newRGBA(newW, newH)
	sb := src.Bounds()
	srcW := float64(sb.Dx())
	srcH := float64(sb.Dy())
	for y := 0; y < newH; y++ {
		sy := int(float64(y) * srcH / float64(newH))
		for x := 0; x < newW; x++ {
			sx := int(float64(x) * srcW / float64(newW))
			r, g, b, a := src.At(sb.Min.X+sx, sb.Min.Y+sy).RGBA()
			dst.SetRGBA(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}
	return dst
}

// pasteRounded composites src onto dst at (ox, oy) with a rounded-rectangle mask.
func pasteRounded(dst *image.RGBA, src image.Image, ox, oy, w, h, radius int) {
	mask := buildRoundedMask(w, h, radius)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			alpha := mask[y*w+x]
			if alpha == 0 {
				continue
			}
			sr, sg, sb, _ := src.At(x, y).RGBA()
			sc := color.RGBA{uint8(sr >> 8), uint8(sg >> 8), uint8(sb >> 8), alpha}
			dc := dst.RGBAAt(ox+x, oy+y)
			dst.SetRGBA(ox+x, oy+y, blendOver(sc, dc))
		}
	}
}

// buildRoundedMask returns a flat alpha slice for a w×h rounded rect.
func buildRoundedMask(w, h, r int) []uint8 {
	mask := make([]uint8, w*h)
	rf := float64(r)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isInsideRoundedRect(x, y, w, h, rf) {
				mask[y*w+x] = 255
			}
		}
	}
	return mask
}

func isInsideRoundedRect(x, y, w, h int, r float64) bool {
	xf, yf := float64(x), float64(y)
	// four corner circles
	corners := [][2]float64{
		{r, r},
		{float64(w) - r, r},
		{r, float64(h) - r},
		{float64(w) - r, float64(h) - r},
	}
	for _, c := range corners {
		dx := xf - c[0]
		dy := yf - c[1]
		if dx*dx+dy*dy < r*r {
			// could be outside if in diagonal corner area outside the arc
			inCornerRegion := (x < int(r) || x >= w-int(r)) && (y < int(r) || y >= h-int(r))
			if inCornerRegion {
				return dx*dx+dy*dy <= r*r
			}
		}
	}
	// outside the actual arc quadrant region → just check rect
	return xf >= 0 && xf < float64(w) && yf >= 0 && yf < float64(h)
}

func drawRoundedRect(img *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if isInsideRoundedRect(px-x, py-y, w, h, float64(radius)) {
				blendPixel(img, px, py, c)
			}
		}
	}
}

func drawRoundedRectBorder(img *image.RGBA, x, y, w, h, radius int, c color.RGBA, thickness int) {
	for t := 0; t < thickness; t++ {
		drawRoundedRectOutline(img, x+t, y+t, w-2*t, h-2*t, radius-t, c)
	}
}

func drawRoundedRectOutline(img *image.RGBA, x, y, w, h, radius int, c color.RGBA) {
	rf := float64(radius)
	for px := x; px < x+w; px++ {
		for py := y; py < y+h; py++ {
			inside := isInsideRoundedRect(px-x, py-y, w, h, rf)
			insideInner := isInsideRoundedRect(px-x-1, py-y-1, w-2, h-2, rf-1)
			if inside && !insideInner {
				blendPixel(img, px, py, c)
			}
		}
	}
}

func drawGlassCard(img *image.RGBA, x1, y1, x2, y2, radius, alpha int) {
	glassColor := color.RGBA{255, 255, 255, uint8(alpha)}
	w := x2 - x1
	h := y2 - y1
	drawRoundedRect(img, x1, y1, w, h, radius, glassColor)
}

func drawGlow(img *image.RGBA, x, y, w, h, blurR int, c color.RGBA) {
	tmp := newRGBA(img.Bounds().Dx(), img.Bounds().Dy())
	drawRoundedRect(tmp, x, y, w, h, 40, c)
	gaussianBlur(tmp, blurR)
	// composite glow onto img
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			gc := tmp.RGBAAt(px, py)
			if gc.A == 0 {
				continue
			}
			dc := img.RGBAAt(px, py)
			img.SetRGBA(px, py, blendOver(gc, dc))
		}
	}
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	rf := float64(r)
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			if math.Sqrt(dx*dx+dy*dy) <= rf {
				blendPixel(img, x, y, c)
			}
		}
	}
}

// drawTextSimple renders ASCII text using a 7×9 bitmap font baked inline.
// weight=2 doubles the pixel width for a bolder look.
func drawTextSimple(img *image.RGBA, text string, x, y int, c color.RGBA, weight int) {
	cx := x
	for _, ch := range text {
		bm, ok := bitmapFont[ch]
		if !ok {
			bm = bitmapFont[' ']
		}
		for row := 0; row < fontH; row++ {
			for col := 0; col < fontW; col++ {
				if bm[row]&(1<<uint(fontW-1-col)) != 0 {
					for w := 0; w < weight; w++ {
						blendPixel(img, cx+col*weight+w, y+row*weight, c)
					}
				}
			}
		}
		cx += (fontW + 1) * weight
	}
}

// blendPixel alpha-composites c over the existing pixel at (x, y).
func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
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

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

// ─── I/O helpers ─────────────────────────────────────────────────────────────

func downloadFile(url, dest string) error {
	resp, err := httpClient.Get(url)
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

	// Try JPEG first, then PNG
	img, err := jpeg.Decode(f)
	if err != nil {
		f.Seek(0, 0)
		img, err = png.Decode(f)
	}
	return img, err
}