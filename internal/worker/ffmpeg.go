// internal/worker/ffmpeg.go
package worker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ibra172/go-ffmpeg-pipeline/internal/features/task"
)

var resolutionToScale = map[string]string{
	"360p":  "-2:360",
	"480p":  "-2:480",
	"720p":  "-2:720",
	"1080p": "-2:1080",
	"2k":    "-2:1440",
	"4k":    "-2:2160",
}

func buildFFmpegCommand(
	msg task.Message,
	dataDir,
	watermarkPath string,
) (outputPath string, args []string, err error) {
	resultsDir := filepath.Join(dataDir, "results")
	if err := os.MkdirAll(resultsDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create results dir: %w", err)
	}

	inputBaseName := filepath.Base(msg.InputPath)
	inputExt := filepath.Ext(inputBaseName)
	nameWithoutExt := strings.TrimSuffix(inputBaseName, inputExt)

	switch msg.Operation {
	case "thumbnail":
		outputPath = filepath.Join(resultsDir, nameWithoutExt+"_preview.jpg")
		args = []string{
			"-y", "-ss", "00:00:01", "-i", msg.InputPath,
			"-vf", "thumbnail,smartblur,unsharp",
			"-vframes", "1", "-q:v", "2",
			outputPath,
		}
	case "transcode":
		args = []string{"-y", "-i", msg.InputPath}
		if msg.Resolution != "" {
			args = append(args, "-vf", "scale="+resolutionToScale[msg.Resolution], "-c:a", "copy")
			nameWithoutExt += "_" + msg.Resolution
		}
		outputPath = filepath.Join(resultsDir, nameWithoutExt+"."+msg.TargetFormat)
		args = append(args, outputPath)
	case "watermark":
		outputPath = filepath.Join(resultsDir, inputBaseName+"_logo"+inputExt)
		args = []string{
			"-y", "-i", msg.InputPath, "-i", watermarkPath,
			"-filter_complex",
			"[1:v][0:v]scale2ref=w=ih*0.15*mdar:h=ih*0.15[wm][vid];" +
				"[vid][wm]overlay=W-w-20:H-h-20",
			"-c:v", "libx264", "-c:a", "copy",
			outputPath,
		}
	default:
		return "", nil, fmt.Errorf("unknown operation: %s", msg.Operation)
	}

	return outputPath, args, nil
}

func runFFmpeg(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}

	return nil
}
