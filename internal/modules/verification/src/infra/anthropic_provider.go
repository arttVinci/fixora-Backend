package infra

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

type AnthropicProvider struct{ baseHTTPProvider }

func NewAnthropicProvider(config *viper.Viper) *AnthropicProvider {
	model := config.GetString("verification.anthropic_model")
	if model == "" {
		model = "claude-3-5-sonnet-latest"
	}
	key := config.GetString("anthropic.api_key")
	if key == "" {
		key = config.GetString("ANTHROPIC_API_KEY")
	}
	return &AnthropicProvider{baseHTTPProvider{apiKey: key, model: model, provider: "anthropic", limiter: rate.NewLimiter(rate.Every(2*time.Second), 1)}}
}
func (p *AnthropicProvider) ProviderName() string { return "anthropic" }
func (p *AnthropicProvider) ModelName() string    { return p.model }
func (p *AnthropicProvider) AnalyzeAsAdvocate(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Advocate", req)
}
func (p *AnthropicProvider) AnalyzeAsSkeptic(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Skeptic: cari kelemahan dan ketidakkonsistenan laporan", req)
}
func (p *AnthropicProvider) AnalyzeAsManager(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Manager", req)
}
func (p *AnthropicProvider) analyze(ctx context.Context, role string, req *VerificationRequest) (*VerificationResult, error) {
	body := map[string]any{"model": p.model, "max_tokens": 1000, "messages": []map[string]string{{"role": "user", "content": rolePrompt(role, req)}}}
	data, err := p.doJSON(ctx, "https://api.anthropic.com/v1/messages", map[string]string{"x-api-key": p.apiKey, "anthropic-version": "2023-06-01"}, body)
	if err != nil {
		return nil, err
	}
	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	text := ""
	for _, c := range res.Content {
		text += c.Text
	}
	return parseResult(text)
}
