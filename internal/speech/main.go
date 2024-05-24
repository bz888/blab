package speech

import (
	speechCmd "github.com/bz888/blab/internal/speech/cmd"
	"github.com/bz888/blab/internal/speech/config"
	"github.com/bz888/blab/internal/speech/output_api"
)

func Init() {
	config.Init()
	output_api.Init()
}

func Run() (string, error) {
	outputText, err := speechCmd.Run()
	if err != nil {
		return "", err
	}
	return outputText, nil
}
