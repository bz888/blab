package cmd

import (
	"github.com/bz888/blab/internal/api"
	"github.com/bz888/blab/internal/api/server"
	"github.com/bz888/blab/internal/config"
	"github.com/bz888/blab/internal/logger"
	"github.com/bz888/blab/internal/speech"
	"github.com/bz888/blab/internal/ui"
	"log"
)

func init() {
	config.Init()
}

func Execute() {
	ui.Init()
	debugConsole, err := ui.GetDebugConsole()

	if err != nil {
		log.Fatal(err)
	}

	logger.InitLogger(config.Dev, config.LogPath, debugConsole)

	api.Init()
	server.Init()
	speech.Init()

	go server.Run()
	ui.Run()
}
