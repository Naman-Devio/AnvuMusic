/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package main

import (
	"os/exec"

	"github.com/Laky-64/gologging"
)

func checkFFmpegAndFFprobe() {
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			gologging.FatalF(
				"❌ %s not found in PATH. Please install %s.",
				bin,
				bin,
			)
		}
	}
}
