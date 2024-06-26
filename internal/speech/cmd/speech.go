package cmd

import (
	"context"
	"fmt"
	"github.com/bz888/blab/internal/logger"
	speechConfig "github.com/bz888/blab/internal/speech/config"
	"github.com/bz888/blab/internal/speech/convert"
	"github.com/bz888/blab/internal/speech/output_api"
	"github.com/bz888/blab/internal/speech/sound"
	vadlib "github.com/bz888/blab/internal/speech/vad"
	"github.com/go-audio/wav"
	"github.com/orcaman/writerseeker"
	"io"
	"math"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-audio/audio"
	"github.com/gordonklaus/portaudio"
)

const (
	minMicVolume       = 450
	sendToVADDelay     = time.Second
	maxSegmentDuration = time.Second * 25
)

var localLogger *logger.Logger

func Run() (string, error) {
	localLogger = speechConfig.LocalLogger

	if speechConfig.Disable {
		localLogger.Warn("GOOGLE_API_KEY is not set, voice recognition is disabled")
		return "", nil
	}

	portaudio.Initialize()
	defer portaudio.Terminate()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// If there is no selected device, print all of them and exit.
	var selectedDevice, _ = portaudio.DefaultInputDevice()

	done := make(chan bool)
	audioCtx, audioCancel := context.WithCancel(ctx)

	// Set up the audio stream parameters for LINEAR16 PCM
	in := make([]int16, 512*9) // Use int16 to capture 16-bit samples.
	audioStream, err := portaudio.OpenDefaultStream(
		selectedDevice.MaxInputChannels, 0, selectedDevice.DefaultSampleRate, len(in), &in,
	)

	if err != nil {
		localLogger.Fatal("opening stream: %v", err)
	}

	// Start the audio stream
	if err := audioStream.Start(); err != nil {
		localLogger.Fatal("starting stream: %v", err)
	}

	// Silero VAD - pre-trained Voice Activity Detector. See: https://github.com/snakers4/silero-vad
	sileroVAD, err := vadlib.NewSileroDetector(speechConfig.SileroFilePath)
	if err != nil {
		localLogger.Fatal("creating silero detector: %v", err)
	}

	var (
		startListening time.Time
		processChan    = make(chan []int16, 10)
		outChan        = make(chan audio.Buffer, 10)
		buffer         = make([]int16, 512*9)
		resultChan     = make(chan string)
	)

	go func() {
		for {
			select {
			case <-audioCtx.Done():
				if err := audioStream.Close(); err != nil {
					localLogger.Error(err)
				}
				localLogger.Error("got audioCtx.Done exit gracefully...")
				return
			default:
				// Read from the microphone
				if err := audioStream.Read(); err != nil {
					localLogger.Info("reading from stream: %v\n", err)
					continue
				}

				volume := calculateRMS16(in)
				if volume > minMicVolume {
					startListening = time.Now()
				}

				if time.Since(startListening) < sendToVADDelay && time.Since(startListening) < maxSegmentDuration {
					buffer = append(buffer, in...)

					localLogger.Info("listening...", volume)
				} else if len(buffer) > 0 {
					// Silero accept audio with SampleRate = 16000.

					// Resample also copies the buffer to another slice. Potentially, using a channel instead of a
					// buffer can achieve better performance.
					localLogger.Info("Sending to processChannel...", len(buffer))
					processChan <- sound.ResampleInt16(buffer, int(selectedDevice.DefaultSampleRate), 16000)
					buffer = buffer[:0]
				}
			}
		}
	}()

	// Responsible for checking recorded sections for the presence of the user's voice.
	go vad(sileroVAD, processChan, outChan)

	// Encodes the final sound into wav -> flac
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		process(outChan, resultChan, &wg)
	}()

	var result string
	go func() {
		for resp := range resultChan {
			localLogger.Info("Response:", resp)
			result += resp + "\n"
			wg.Done()
		}
	}()
	wg.Wait()
	close(resultChan)

	localLogger.Info("finished")

	// Shutdown.
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			localLogger.Error(fmt.Errorf("shutdown: %w", err))
		}
		audioCancel()
		close(done)
	}()

	return result, nil
}

func vad(silero *vadlib.SileroDetector, input <-chan []int16, output chan audio.Buffer) {
	soundIntBuffer := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: 16000, NumChannels: 1},
	}

	for {
		soundIntBuffer.Data = sound.ConvertInt16ToInt(<-input)

		start := time.Now()
		detected, err := silero.DetectVoice(soundIntBuffer)
		if err != nil {
			localLogger.Error(fmt.Errorf("detect voice: %w", err))
			continue
		}
		localLogger.Info("voice detecting result", time.Since(start), detected)

		if detected {
			localLogger.Info("sending to output...")
			output <- soundIntBuffer.Clone()
		}
	}
}

// google api
func process(in <-chan audio.Buffer, resultChan chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		data := <-in

		// Emulate a file in RAM so that we don't have to create a real file.
		file := &writerseeker.WriterSeeker{}
		encoder := wav.NewEncoder(file, 16000, 16, 1, 1)

		// Write the audio buffer to the WAV file using the encoder
		if err := encoder.Write(data.AsIntBuffer()); err != nil {
			localLogger.Info(fmt.Errorf("encoder write buffer: %w", err))
			return
		}

		// Close the encoder to finalize the WAV file headers
		if err := encoder.Close(); err != nil {
			localLogger.Info(fmt.Errorf("encoder close: %w", err))
			return
		}

		// Read all data from the reader into memory
		wavData, err := io.ReadAll(file.Reader())
		if err != nil {
			localLogger.Info(fmt.Errorf("reading WAV file into memory: %w", err))
			return
		}

		localLogger.Info("WAV data length: %d bytes\n", len(wavData))
		if len(wavData) == 0 {
			localLogger.Error("WAV data is empty")
			return
		}

		localLogger.Info("Encode to FLAC beginning")
		//flacData, err := convert.EncodeFLAC(wavData)
		flacData, err := convert.EncodeFLACExecutable(wavData, 16000, 2)
		localLogger.Info("FLAC data length: %d bytes\n", len(flacData))

		if err != nil {
			localLogger.Error(fmt.Errorf("FLAC encoding error: %w", err))
			return
		}
		localLogger.Info("Encode to FLAC successfully")

		if len(flacData) == 0 {
			localLogger.Error("FLAC data is empty")
			return
		}

		start := time.Now()
		localLogger.Info("Sending to out")
		resp, conf, err := output_api.Send(flacData)
		if err != nil {
			localLogger.Error(fmt.Errorf("sending multipart form: %w", err))
			return
		}

		resultChan <- resp
		localLogger.Info(fmt.Sprintf("done in: %s, confidence: %s, result: %s", time.Since(start), int(conf), resp))
	}
}

func printAvailableDevices() {
	devices, err := portaudio.Devices()
	if err != nil {
		localLogger.Fatal("portaudio.Devices %s", err)
		return
	}
	for i, device := range devices {
		localLogger.Info(
			"ID: %d, Name: %s, MaxInputChannels: %d, Sample rate: %f\n",
			i,
			device.Name,
			device.MaxInputChannels,
			device.DefaultSampleRate,
		)
	}
}

func selectInputDevice(args []string) (*portaudio.DeviceInfo, error) {
	deviceID, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, fmt.Errorf("parce int %w", err)
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("select input device %w", err)
	}

	selectedDevice, err := portaudio.DefaultInputDevice()
	if err != nil {
		return nil, fmt.Errorf("find default device %w", err)
	}

	// Set default device to device with particular id
	selectedDevice = devices[deviceID]

	localLogger.Info("selected device:", selectedDevice.Name, selectedDevice.DefaultSampleRate)

	return selectedDevice, nil
}

// calculateRMS16 calculates the root-mean-square of the audio buffer for int16 samples.
func calculateRMS16(buffer []int16) float64 {
	var sumSquares float64
	for _, sample := range buffer {
		val := float64(sample) // Convert int16 to float64 for calculation
		sumSquares += val * val
	}
	meanSquares := sumSquares / float64(len(buffer))
	return math.Sqrt(meanSquares)
}
