package server

import (
	"encoding/json"
	"github.com/bz888/blab/internal/logger"
	"log"
	"net/http"
	"strconv"
)

var (
	ollamaHost  = "localhost:11434"
	localLogger *logger.Logger
	port        = 8080
)

func Init() {
	localLogger = logger.NewLogger("Server")
}

func Run() {
	registerRoutes()

	address := ":" + strconv.Itoa(port)
	localLogger.Info("Debug mode is enabled")

	// Start the server
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("Error starting server: ", err)
	}
	localLogger.Info("Server started on http://localhost" + address + "/")
}

// UnmarshalJSON handles the custom unmarshalling for Families.
func (f *Families) UnmarshalJSON(data []byte) error {
	// If the JSON data is "null", return an empty Families slice.
	if string(data) == "null" {
		*f = Families{}
		return nil
	}

	// Otherwise, unmarshal the data as a regular slice of strings.
	var families []string
	if err := json.Unmarshal(data, &families); err != nil {
		return err
	}
	*f = Families(families)
	return nil
}
