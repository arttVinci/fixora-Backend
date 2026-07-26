package client

import (
	"context"

	"github.com/sirupsen/logrus"
)

type ExtractionResult struct {
	Location   string `json:"location"`
	CategoryID string `json:"category_id"`
	Severity   string `json:"severity"`
	IsRelevant bool   `json:"is_relevant"`
}

type LlmClient interface {
	ExtractNewsInfo(ctx context.Context, title, content string) (*ExtractionResult, error)
}

type llmClientImpl struct {
	Log *logrus.Logger
}

func NewLlmClient(log *logrus.Logger) LlmClient {
	return &llmClientImpl{Log: log}
}

func (c *llmClientImpl) ExtractNewsInfo(ctx context.Context, title, content string) (*ExtractionResult, error) {
	// TODO: Implement actual HTTP call to OpenAI/Gemini/Anthropic API
	c.Log.Infof("Mock LLM extracting data for title: %s", title)
	
	return &ExtractionResult{
		Location:   "Jalan Sudirman, Jakarta",
		CategoryID: "00000000-0000-0000-0000-000000000000", // Ganti dengan ID real nanti
		Severity:   "sedang",
		IsRelevant: true,
	}, nil
}
