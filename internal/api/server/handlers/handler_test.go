package handlers

import (
	"context"
	"log"
	"testing"

	"github.com/bz888/blab/internal/api/server/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) GetModels() ([]client.OpenAIModel, error) {
	args := m.Called()
	return args.Get(0).([]client.OpenAIModel), args.Error(1)
}

func (m *MockOpenAIClient) Chat(ctx context.Context, req *client.OpenAIChatRequest, fn func([]byte) error) error {
	return nil
}

func TestProcessOpenAiModels(t *testing.T) {
	mockOpenAIClient := new(MockOpenAIClient)
	handler := &Handler{
		openAIClient: mockOpenAIClient,
	}

	mockModels := []client.OpenAIModel{
		{ID: "dall-e-3", OwnedBy: "system"},
		{ID: "whisper-1", OwnedBy: "openai-internal"},
		{ID: "davinci-002", OwnedBy: "system"},
		{ID: "babbage-002", OwnedBy: "system"},
		{ID: "dall-e-2", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo-16k", OwnedBy: "openai-internal"},
		{ID: "tts-1-hd-1106", OwnedBy: "system"},
		{ID: "tts-1-hd", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo-1106", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo-instruct-0914", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo-instruct", OwnedBy: "system"},
		{ID: "tts-1", OwnedBy: "openai-internal"},
		{ID: "gpt-3.5-turbo-0301", OwnedBy: "openai"},
		{ID: "gpt-3.5-turbo-0125", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo", OwnedBy: "openai"},
		{ID: "tts-1-1106", OwnedBy: "system"},
		{ID: "text-embedding-3-large", OwnedBy: "system"},
		{ID: "text-embedding-3-small", OwnedBy: "system"},
		{ID: "gpt-3.5-turbo-0613", OwnedBy: "openai"},
		{ID: "text-embedding-ada-002", OwnedBy: "openai-internal"},
		{ID: "gpt-3.5-turbo-16k-0613", OwnedBy: "openai"},
	}

	client.CacheModels = make(map[string]string)

	mockOpenAIClient.On("GetModels").Return(mockModels, nil)
	client.AddOpenAIModelCache(mockModels)

	modelNames := handler.processOpenAiModels()

	log.Println(client.CacheModels)

	expectedModelNames := []string{"gpt-3.5-turbo-0301", "gpt-3.5-turbo", "gpt-3.5-turbo-0613", "gpt-3.5-turbo-16k-0613"}

	assert.ElementsMatch(t, expectedModelNames, modelNames, "The model names should match the expected ones")
	assert.Equal(t, "openai", client.CacheModels["gpt-3.5-turbo-0301"])
	assert.Equal(t, "openai", client.CacheModels["gpt-3.5-turbo"])
	assert.Equal(t, "openai", client.CacheModels["gpt-3.5-turbo-0613"])
	assert.Equal(t, "openai", client.CacheModels["gpt-3.5-turbo-16k-0613"])

	client.CacheModels = map[string]string{
		"gpt-3.5-turbo-0301":     "openai",
		"gpt-3.5-turbo":          "openai",
		"gpt-3.5-turbo-0613":     "openai",
		"gpt-3.5-turbo-16k-0613": "openai",
	}

	modelNames = handler.processOpenAiModels()

	assert.ElementsMatch(t, expectedModelNames, modelNames, "The model names should match the expected ones")
}
