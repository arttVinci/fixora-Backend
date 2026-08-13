package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

type OpenAIProvider struct{ baseHTTPProvider }

func NewOpenAIProvider(config *viper.Viper) *OpenAIProvider {
	model := config.GetString("verification.openai_model")
	if model == "" {
		model = "gpt-4o-mini"
	}
	key := config.GetString("openai.api_key")
	if key == "" {
		key = config.GetString("OPENAI_API_KEY")
	}
	return &OpenAIProvider{baseHTTPProvider{apiKey: key, model: model, provider: "openai", limiter: rate.NewLimiter(rate.Every(time.Second), 1)}}
}
func (p *OpenAIProvider) ProviderName() string { return "openai" }
func (p *OpenAIProvider) ModelName() string    { return p.model }
func (p *OpenAIProvider) AnalyzeAsAdvocate(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Advocate", req)
}
func (p *OpenAIProvider) AnalyzeAsSkeptic(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Skeptic", req)
}
func (p *OpenAIProvider) AnalyzeAsManager(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Manager/Hakim: putuskan final dari argumen advocate dan skeptic", req)
}
func (p *OpenAIProvider) analyze(ctx context.Context, role string, req *VerificationRequest) (*VerificationResult, error) {
	body := map[string]any{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": rolePrompt(role, req)}},
		"response_format": map[string]string{"type": "json_object"},
	}
	
	data, err := p.doJSON(ctx, "https://api.openai.com/v1/chat/completions", map[string]string{"Authorization": "Bearer " + p.apiKey}, body)
	if err != nil {
		return nil, err
	}
	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	if len(res.Choices) == 0 {
		return parseResult("")
	}
	return parseResult(res.Choices[0].Message.Content)
}
