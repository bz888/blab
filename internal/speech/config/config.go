package config

import (
	"github.com/bz888/blab/internal/logger"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
)

var (
	SileroFilePath string
	Disable        = false
	LocalLogger    *logger.Logger
)

func Init() {
	LocalLogger = logger.NewLogger("speech")
	godotenv.Load()
	key := os.Getenv("API_KEY")
	if key == "" {
		Disable = true
	}

	basePath, err := filepath.Abs(filepath.Join("./internal", "files"))
	if err != nil {
		LocalLogger.Fatal("Failed to determine working directory: %v", err)
	}

	SileroFilePath = filepath.Join(basePath, "silero_vad.onnx")
}
