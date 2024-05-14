package output_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Alternative struct {
	Transcript string  `json:"transcript"`
	Confidence float64 `json:"confidence"`
}

type Result struct {
	Alternative []Alternative `json:"alternative"`
	Final       bool          `json:"final"`
}

type Response struct {
	Result []Result `json:"result"`
}

type OutputParser struct {
	ShowAll        bool
	WithConfidence bool
}

func buildRecogniserRequestGoogle(audioData []byte) *http.Request {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loadin .env file: %v", err)
	}

	key := os.Getenv("API_KEY")
	apiURL := "http://www.google.com/speech-api/v2/recognize"
	data := url.Values{}
	data.Set("client", "chromium")
	data.Set("lang", "en-US")
	data.Set("key", key)
	data.Set("pFilter", "0")

	req, err := http.NewRequest("POST", apiURL+"?"+data.Encode(), bytes.NewReader(audioData))
	if err != nil {
		fmt.Println(err)
		return nil
	}

	req.Header.Add("Content-Type", "audio/x-flac; rate=16000")
	return req
}

func convertToResult(responseText string) (Result, error) {
	for _, line := range strings.Split(responseText, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response Response
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			return Result{}, err
		}

		log.Println(responseText)
		if len(response.Result) != 0 {
			if len(response.Result[0].Alternative) == 0 {
				log.Println("No alternatives found in the result.")
				return Result{}, errors.New("no alternatives found")
			}
			return response.Result[0], nil
		}
	}
	return Result{}, errors.New("no valid results found")
}

func findBestHypothesis(alternatives []Alternative) (Alternative, error) {
	if len(alternatives) == 0 {
		return Alternative{}, errors.New("no alternatives provided")
	}

	var bestHypothesis Alternative
	highestConfidence := -1.0

	for _, alternative := range alternatives {
		if alternative.Confidence > highestConfidence {
			highestConfidence = alternative.Confidence
			bestHypothesis = alternative
		}
	}

	if bestHypothesis.Transcript == "" {
		log.Println("Best hypothesis does not have a transcript.")
		return Alternative{}, errors.New("best hypothesis does not have a transcript")
	}

	return bestHypothesis, nil
}

func (op *OutputParser) parse(responseText string) (string, float64, error) {
	actualResult, err := convertToResult(responseText)
	if err != nil {
		return "", 0, err
	}

	if op.ShowAll {
		return fmt.Sprintf("%+v", actualResult), 0, nil
	}

	bestHypothesis, err := findBestHypothesis(actualResult.Alternative)
	if err != nil {
		return "", 0, err
	}

	confidence := bestHypothesis.Confidence
	if confidence == 0 {
		confidence = 0.5
	}

	if op.WithConfidence {
		return bestHypothesis.Transcript, confidence, nil
	}

	return bestHypothesis.Transcript, 0, nil
}

func sendRecogniserRequestGoogle(req *http.Request) (string, float64, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request:", err)
		return "", 0, err
	}
	defer resp.Body.Close()

	log.Printf("Response Status: %s\n", resp.Status)

	// Log the response headers
	log.Println("Response Headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			log.Printf("%s: %s\n", key, value)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return "", 0, err
	}
	log.Printf("Response Body: %s\n", string(body))

	op := OutputParser{
		ShowAll:        false,
		WithConfidence: true,
	}

	transcript, confidence, err := op.parse(string(body))
	if err != nil {
		return "", 0, err
	}

	return transcript, confidence, nil
}

func Send(audioData []byte) (string, float64, error) {
	req := buildRecogniserRequestGoogle(audioData)
	if req == nil {
		return "", 0, errors.New("failed to build request")
	}

	transcript, confidence, err := sendRecogniserRequestGoogle(req)
	if err != nil {
		return "", 0, err
	}

	log.Println("Sent")
	return transcript, confidence, nil
}
