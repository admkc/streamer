package streamer

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/sirupsen/logrus"
)

// IProcess is an interface around the FFMPEG process
type IProcess interface {
	Spawn(path, URI string) *exec.Cmd
}

// ProcessLoggingOpts describes options for process logging
type ProcessLoggingOpts struct {
	Enabled    bool   // Option to set logging for transcoding processes
	Directory  string // Directory for the logs
	MaxSize    int    // Maximum size of kept logging files in megabytes
	MaxBackups int    // Maximum number of old log files to retain
	MaxAge     int    // Maximum number of days to retain an old log file.
	Compress   bool   // Indicates if the log rotation should compress the log files
}

// Process is the main type for creating new processes
type Process struct {
	keepFiles      bool
	audio          bool
	live           bool
	streamDuration int
	loggingOpts    ProcessLoggingOpts
}

// Type check
var _ IProcess = (*Process)(nil)

// NewProcess creates a new process able to spawn transcoding FFMPEG processes
func NewProcess(
	keepFiles bool,
	audio bool,
	live bool,
	streamDuration int,
	loggingOpts ProcessLoggingOpts,
) *Process {
	return &Process{keepFiles, audio, live, streamDuration, loggingOpts}
}

// getHLSFlags are for getting the flags based on the config context
func (p Process) getHLSFlags() string {
	if p.keepFiles {
		return "append_list"
	}
	return "delete_segments+append_list"
}

// Spawn creates a new FFMPEG cmd
func (p Process) Spawn(path, URI string) *exec.Cmd {
	os.MkdirAll(path, os.ModePerm)
	processCommands := []string{
		"-y",
		"-rtsp_transport",
		"tcp",
		"-i",
		URI,
		"-c:v",
		"libx264",
		"-x264opts",
		"keyint=30:no-scenecut",
		"-preset",
		"veryfast",
	}
	if p.audio {
		processCommands = append(processCommands, "-c:a", "copy")
	} else {
		processCommands = append(processCommands, "-an")
	}
	if p.streamDuration > 0 {
		processCommands = append(processCommands, "-t", strconv.Itoa(p.streamDuration))
	}
	processCommands = append(processCommands,
		"-f",
		"hls",
	)
	if p.live {
		processCommands = append(processCommands,
			"-hls_flags",
			p.getHLSFlags(),
			"-segment_list_flags",
			"live",
			"-hls_time",
			"1",
			"-hls_list_size",
			"3",
		)
	} else {
		processCommands = append(processCommands,
			"-hls_list_size",
			"0",
			"-hls_time",
			"5",
		)
	}
	processCommands = append(processCommands,
		"-hls_segment_filename",
		fmt.Sprintf("%s/%%d.ts", path),
		fmt.Sprintf("%s/index.m3u8", path),
	)
	logrus.Debugf("ffmpeg params: %s", processCommands)
	cmd := exec.Command("ffmpeg", processCommands...)
	return cmd
}
