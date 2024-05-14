package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gordonklaus/portaudio"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"log"
	"os"
)

type AudioData struct {
	frameData   []byte
	sampleRate  int
	sampleWidth int
}

func NewAudioData(frameData []byte, sampleRate, sampleWidth int) *AudioData {
	return &AudioData{
		frameData:   frameData,
		sampleRate:  sampleRate,
		sampleWidth: sampleWidth,
	}
}

func (a *AudioData) GetSegment(startMS, endMS *int) *AudioData {
	startByte := 0
	endByte := len(a.frameData)

	if startMS != nil {
		startByte = (*startMS * a.sampleRate * a.sampleWidth) / 1000
	}

	if endMS != nil {
		endByte = (*endMS * a.sampleRate * a.sampleWidth) / 1000
	}

	return NewAudioData(a.frameData[startByte:endByte], a.sampleRate, a.sampleWidth)
}

func (a *AudioData) GetRawData(convertRate, convertWidth *int) ([]byte, error) {
	rawData := a.frameData

	if convertRate != nil && *convertRate != a.sampleRate {
		rawData = resample(rawData, a.sampleRate, *convertRate, a.sampleWidth)
		a.sampleRate = *convertRate
	}

	if convertWidth != nil && *convertWidth != a.sampleWidth {
		rawData = convertSampleWidth(rawData, a.sampleWidth, *convertWidth)
		a.sampleWidth = *convertWidth
	}

	return rawData, nil
}

func convertSampleWidth(data []byte, fromWidth, toWidth int) []byte {
	inBuf := bytes.NewReader(data)
	decoder := wav.NewDecoder(inBuf)
	decoder.ReadInfo()
	pcm, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Error decoding PCM data: %v", err)
	}

	outBuf := new(bytes.Buffer)
	switch {
	case fromWidth == 1 && toWidth == 2:
		for _, sample := range pcm.AsIntBuffer().Data {
			binary.Write(outBuf, binary.LittleEndian, int16(sample)<<8)
		}
	case fromWidth == 2 && toWidth == 1:
		for _, sample := range pcm.AsIntBuffer().Data {
			outBuf.WriteByte(byte(sample >> 8))
		}
	case fromWidth == 2 && toWidth == 3:
		for _, sample := range pcm.AsIntBuffer().Data {
			sample24 := int32(sample) << 8
			binary.Write(outBuf, binary.LittleEndian, sample24)
		}
	case fromWidth == 3 && toWidth == 2:
		for i := 0; i < len(data); i += 3 {
			sample24 := int32(data[i]) | int32(data[i+1])<<8 | int32(data[i+2])<<16
			binary.Write(outBuf, binary.LittleEndian, int16(sample24>>8))
		}
	default:
		return data
	}

	return outBuf.Bytes()
}

func resample(data []byte, fromRate, toRate, sampleWidth int) []byte {

	if fromRate == toRate {
		return data
	}

	inBuf := bytes.NewReader(data)
	decoder := wav.NewDecoder(inBuf)
	decoder.ReadInfo()
	pcm, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Error decoding PCM data: %v", err)
	}
	src := audio.IntBuffer{Format: &audio.Format{SampleRate: fromRate}, Data: pcm.AsIntBuffer().Data}
	resampled := audio.IntBuffer{Format: &audio.Format{SampleRate: toRate}}

	srcLen := len(src.Data)
	dstLen := int(float64(srcLen) * float64(toRate) / float64(fromRate))
	resampled.Data = make([]int, dstLen)

	for i := 0; i < dstLen; i++ {
		srcIndex := float64(i) * float64(srcLen-1) / float64(dstLen-1)
		intPart := int(srcIndex)
		fracPart := srcIndex - float64(intPart)
		if intPart+1 < srcLen {
			resampled.Data[i] = int(float64(src.Data[intPart])*(1-fracPart) + float64(src.Data[intPart+1])*fracPart)
		} else {
			resampled.Data[i] = src.Data[intPart]
		}
	}
	outBuf := new(bytes.Buffer)
	switch sampleWidth {
	case 1:
		for _, sample := range resampled.Data {
			outBuf.WriteByte(byte(sample))
		}
	case 2:
		for _, sample := range resampled.Data {
			binary.Write(outBuf, binary.LittleEndian, int16(sample))
		}
	case 3:
		for _, sample := range resampled.Data {
			sample24 := int32(sample) << 8
			binary.Write(outBuf, binary.LittleEndian, sample24)
		}
	}

	return outBuf.Bytes()
}

func (a *AudioData) GetFLACData(convertRate, convertWidth *int) ([]byte, error) {
	if convertWidth != nil && (*convertWidth < 1 || *convertWidth > 3) {
		return nil, fmt.Errorf("sample width to convert to must be between 1 and 3 inclusive")
	}

	if a.sampleWidth > 3 && convertWidth == nil {
		convertWidth = new(int)
		*convertWidth = 3
	}

	wavData, err := a.GetWAVData(convertRate, convertWidth)
	if err != nil {
		return nil, err
	}

	return EncodeFLAC(wavData, a.sampleRate, a.sampleWidth)
}

func (a *AudioData) GetWAVData(convertRate, convertWidth *int) ([]byte, error) {
	rawData, err := a.GetRawData(convertRate, convertWidth)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint32(0x46464952)) // "RIFF"
	binary.Write(buf, binary.LittleEndian, uint32(36+len(rawData)))
	binary.Write(buf, binary.LittleEndian, uint32(0x45564157)) // "WAVE"
	binary.Write(buf, binary.LittleEndian, uint32(0x20746d66)) // "fmt "
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(a.sampleWidth))
	binary.Write(buf, binary.LittleEndian, uint32(a.sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(a.sampleRate*int(a.sampleWidth)))
	binary.Write(buf, binary.LittleEndian, uint16(a.sampleWidth))
	binary.Write(buf, binary.LittleEndian, uint16(a.sampleWidth*8))
	binary.Write(buf, binary.LittleEndian, uint32(0x61746164)) // "data"
	binary.Write(buf, binary.LittleEndian, uint32(len(rawData)))
	buf.Write(rawData)

	return buf.Bytes(), nil
}

func EncodeFLAC(wavData []byte, sampleRate, sampleWidth int) ([]byte, error) {
	buf := new(bytes.Buffer)

	md5Sum := md5.Sum(wavData)
	streamInfo := &meta.StreamInfo{
		BlockSizeMin:  16,
		BlockSizeMax:  65535,
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
		return nil, err
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
			frameData[i/sampleWidth] = int32(binary.LittleEndian.Uint32(wavData[i:])) & 0xFFFFFF
		case 4:
			frameData[i/sampleWidth] = int32(binary.LittleEndian.Uint32(wavData[i:]))
		}
	}

	flacFrame := &frame.Frame{
		Header: frame.Header{
			SampleRate:    uint32(sampleRate),
			Channels:      1,
			BitsPerSample: uint8(sampleWidth * 8),
		},
		Subframes: []*frame.Subframe{
			{
				Samples: frameData,
			},
		},
	}

	if err := enc.WriteFrame(flacFrame); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func recordAudio(durationSec int) (*AudioData, error) {
	portaudio.Initialize()
	defer portaudio.Terminate()

	in := make([]int16, 44100*durationSec)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in), in)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	log.Println("Starting audio recording")
	if err := stream.Start(); err != nil {
		return nil, err
	}
	if err := stream.Read(); err != nil {
		return nil, err
	}

	log.Println("Stopping audio recording")
	if err := stream.Stop(); err != nil {
		return nil, err
	}

	frameData := make([]byte, len(in)*2)
	for i, sample := range in {
		binary.LittleEndian.PutUint16(frameData[i*2:], uint16(sample))
	}

	return NewAudioData(frameData, 44100, 2), nil
}

func Run() {
	audioData, err := recordAudio(5)
	if err != nil {
		log.Fatalf("Failed to record audio: %v", err)
	}

	flacData, err := audioData.GetFLACData(nil, nil)
	if err != nil {
		log.Fatalf("Failed to convert to FLAC: %v", err)
	}

	if err := os.WriteFile("output.flac", flacData, 0644); err != nil {
		log.Fatalf("Failed to write FLAC file: %v", err)
	}

	fmt.Println("FLAC file written successfully")
}

func main() {
	Run()
}
