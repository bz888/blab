package config

import "flag"

var (
	Dev     bool
	LogPath string
)

func Init() {
	flag.BoolVar(&Dev, "dev", false, "Development mode")
	flag.StringVar(&LogPath, "logPath", "", "Path to save the log file")
	flag.Parse()
}
