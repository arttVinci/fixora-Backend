package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type LLMClient struct {
	apiKey string
	baseUrl string
	limitter *rate.Limiter
	log *logrus.Logger
}

func NewLLMClient(config dto.LLMProvider, log *logrus.Logger) *LLMClient {
	return &LLMClient{
		apiKey:   config.ApiKey,
		baseUrl:  config.BaseUrl,
		limitter: rate.NewLimiter(rate.Every(2*time.Second), 1),
		log:      log,
	}
}

func (l *LLMClient) GenerateContent(ctx context.Context ,req dto.LLMGenerateContentRequest) (*dto.LLMGenerateContentResponse,error) {
	if req.Content == "" {
		l.log.Warnf("Invalid request : content is empty")
		return nil, fiber.NewError(fiber.StatusBadRequest, "Content tidak boleh kosong")
	}
	if req.Model == "" {
		l.log.Warnf("Invalid request : model is empty")
		return nil, fiber.NewError(fiber.StatusBadRequest, "Model tidak boleh kosong")
	}

	if err := l.limitter.Wait(ctx); err != nil {
		l.log.Warnf("Rate limiter wait failed : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses permintaan ke LLM")
	}

	body := &dto.LLMRequest{
		Model: req.Model,
		Messages: []dto.LLMMessage{
			{
				Role: "user", 
				Content: req.Content,
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		l.log.Warnf("Failed to marshal LLM request body : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat permintaan ke LLM")
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		l.log.Warnf("Failed to create LLM HTTP request : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membuat permintaan ke LLM")
	}

	httpRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.apiKey))
	httpRequest.Header.Set("Content-Type", "application/json")
	
	resp, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		l.log.Warnf("Failed to call LLM provider : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal menghubungi penyedia LLM")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		l.log.Warnf("Failed to read LLM response body : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal membaca respons dari LLM")
	}

	if resp.StatusCode != http.StatusOK {
		l.log.Warnf("LLM provider returned non-200 status : %d, body: %s", resp.StatusCode, string(respBody))
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Penyedia LLM mengembalikan kesalahan")
	}

	var parsed dto.LLMChatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		l.log.Warnf("Failed to unmarshal LLM response : %+v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memproses respons dari LLM")
	}

	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		l.log.Warn("LLM returned empty response")
		return nil, fiber.NewError(fiber.StatusInternalServerError, "LLM tidak mengembalikan respons yang valid")
	}

	return &dto.LLMGenerateContentResponse{
		Content: parsed.Choices[0].Message.Content,
	}, nil
}
