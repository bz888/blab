package convert

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func EncodeFLAC(wavData []byte, sampleRate, sampleWidth int) ([]byte, error) {
	buf := new(bytes.Buffer)

	md5Sum := md5.Sum(wavData)
	streamInfo := &meta.StreamInfo{
		BlockSizeMin:  16,
		BlockSizeMax:  4096, // Typical block size for FLAC
		FrameSizeMin:  0,
		FrameSizeMax:  0,
		SampleRate:    uint32(sampleRate),
		NChannels:     1,
		BitsPerSample: uint8(sampleWidth * 8),
		NSamples:      uint64(len(wavData) / sampleWidth),
		MD5sum:        md5Sum,
	}

	enc, err := flac.NewEncoder(buf, streamInfo)
	if err != nil {
		return nil, fmt.Errorf("creating FLAC encoder: %w", err)
	}
	defer enc.Close()

	frameData := make([]int32, len(wavData)/sampleWidth)
	for i := 0; i < len(wavData); i += sampleWidth {
		switch sampleWidth {
		case 1:
			frameData[i/sampleWidth] = int32(wavData[i])
		case 2:
			frameData[i/sampleWidth] = int32(binary.LittleEndian.Uint16(wavData[i:]))
		case 3:
			frameData[i/sampleWidth] = int32(binary.LittleEndian.Uint32(wavData[i:]) & 0xFFFFFF)
		case 4:
			frameData[i/sampleWidth] = int32(binary.LittleEndian.Uint32(wavData[i:]))
		}
	}

	blockSize := 4096 // A typical block size for FLAC
	for i := 0; i < len(frameData); i += blockSize {
		end := i + blockSize
		if end > len(frameData) {
			end = len(frameData)
		}
		flacFrame := &frame.Frame{
			Header: frame.Header{
				SampleRate:    uint32(sampleRate),
				Channels:      1,
				BitsPerSample: uint8(sampleWidth * 8),
				BlockSize:     uint16(end - i),
			},
			Subframes: []*frame.Subframe{
				{
					Samples: frameData[i:end],
				},
			},
		}

		//log.Printf("Writing FLAC frame with %d samples\n", len(frameData[i:end]))

		if err := enc.WriteFrame(flacFrame); err != nil {
			return nil, fmt.Errorf("writing FLAC frame: %w", err)
		}
	}

	return buf.Bytes(), nil
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

	// Set up the command
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
