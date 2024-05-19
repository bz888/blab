package convert

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/orcaman/writerseeker"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	blockSize = 4096 // Block size in samples
)

func EncodeFLAC(wavData []byte) ([]byte, error) {
	file := &writerseeker.WriterSeeker{}

	// Parse WAV header
	wavReader := bytes.NewReader(wavData)
	var header [44]byte
	if _, err := io.ReadFull(wavReader, header[:]); err != nil {
		return nil, fmt.Errorf("failed to read WAV header: %w", err)
	}

	// Extract format information from WAV header
	audioFormat := binary.LittleEndian.Uint16(header[20:22])
	if audioFormat != 1 {
		return nil, fmt.Errorf("unsupported WAV format: %d", audioFormat)
	}
	numChannels := binary.LittleEndian.Uint16(header[22:24])
	sampleRate := binary.LittleEndian.Uint32(header[24:28])
	bitsPerSample := binary.LittleEndian.Uint16(header[34:36])

	// Set up FLAC metadata
	streamInfo := &meta.StreamInfo{
		NChannels:     uint8(numChannels),
		BitsPerSample: uint8(bitsPerSample),
		SampleRate:    sampleRate,
	}

	// Write FLAC metadata
	encoder, err := flac.NewEncoder(file, streamInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create FLAC encoder: %w", err)
	}
	defer encoder.Close()

	// Calculate block align
	blockAlign := int(numChannels) * int(bitsPerSample) / 8

	// Create a buffer to hold the sample data for each block
	sampleData := make([]byte, blockSize*blockAlign)
	frameNumber := uint64(0)

	for {
		// Read the correct number of samples for the block
		n, err := io.ReadFull(wavReader, sampleData)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Handle the case where the last block may be smaller than the block size
				sampleData = sampleData[:n]
			} else {
				return nil, fmt.Errorf("failed to read WAV samples: %w", err)
			}
		}

		if n == 0 {
			break
		}

		// Create a frame header
		frameHeader := frame.Header{
			HasFixedBlockSize: true,
			BlockSize:         uint16(len(sampleData) / blockAlign),
			SampleRate:        sampleRate,
			Channels:          frame.Channels(numChannels - 1),
			BitsPerSample:     uint8(bitsPerSample),
			Num:               frameNumber,
		}

		// Encode the sample data into FLAC frames
		flacFrame := &frame.Frame{
			Header:    frameHeader,
			Subframes: make([]*frame.Subframe, numChannels),
		}

		// For each channel, create a subframe
		for ch := 0; ch < int(numChannels); ch++ {
			samples := make([]int32, len(sampleData)/blockAlign)
			for i := 0; i < len(sampleData)/blockAlign; i++ {
				offset := i*blockAlign + ch*int(bitsPerSample/8)
				samples[i] = int32(binary.LittleEndian.Uint16(sampleData[offset:]))
			}
			flacFrame.Subframes[ch] = &frame.Subframe{
				Samples: samples,
			}
		}

		// Write frame to FLAC encoder
		if err := encoder.WriteFrame(flacFrame); err != nil {
			return nil, fmt.Errorf("failed to write FLAC frame: %w", err)
		}

		frameNumber++
	}

	// Seek back to the beginning of the emulated file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek in emulated file: %w", err)
	}

	// Read the entire FLAC data from the emulated file
	flacData, err := io.ReadAll(file.Reader())
	if err != nil {
		return nil, fmt.Errorf("failed to read FLAC data: %w", err)
	}

	return flacData, nil
}

func getExecutableFLAC() (string, error) {
	// Check if the 'flac' utility is installed
	flacConverter, err := exec.LookPath("flac")
	if err != nil {
		// 'flac' utility is not installed, check for bundled binaries
		basePath, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return "", err
		}
		system, machine := runtime.GOOS, runtime.GOARCH
		// todo add support for windows
		if system == "darwin" && (machine == "amd64" || machine == "arm64") {
			flacConverter = filepath.Join(basePath, "flac-mac")
		} else {
			return "", errors.New("FLAC conversion utility not available - consider installing the FLAC command line application")
		}

		// Ensure the FLAC converter is executable
		if err := ensureExecutable(flacConverter); err != nil {
			return "", err
		}
	}

	return flacConverter, nil
}

// ensureExecutable ensures that the file at the given path is executable.
func ensureExecutable(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}
	return nil
}

// AudioData represents the audio data structure.
type AudioData struct {
	sampleWidth int
}

func EncodeFLACExecutable(wavData []byte, sampleWidth, convertWidth int) ([]byte, error) {
	// EncodeFLACExecutable returns a byte slice representing the contents of a FLAC file containing the audio represented by the AudioData instance.
	if convertWidth != 0 && (convertWidth < 1 || convertWidth > 3) {
		return nil, errors.New("sample width to convert to must be between 1 and 3 inclusive")
	}

	if sampleWidth > 3 && convertWidth == 0 {
		convertWidth = 3 // Limit the sample width to 24-bit if the original is 32-bit
	}

	// Get FLAC converter path
	flacConverter, err := getExecutableFLAC()
	if err != nil {
		return nil, fmt.Errorf("failed to get FLAC converter: %w", err)
	}

	cmd := exec.Command(flacConverter, "--stdout", "--totally-silent", "--best", "-")
	cmd.Stdin = bytes.NewReader(wavData)
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run FLAC converter: %w", err)
	}

	// todo support windows

	return out.Bytes(), nil
}
