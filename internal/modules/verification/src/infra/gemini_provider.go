package infra

import (
	"context"
	"errors"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
)

type GeminiProvider struct {
	client  *genai.Client
	model   string
	limiter *rate.Limiter
}

func NewGeminiProvider(config *viper.Viper, client *genai.Client) *GeminiProvider {
	model := config.GetString("verification.gemini_model")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return &GeminiProvider{client: client, model: model, limiter: rate.NewLimiter(rate.Every(4*time.Second), 1)}
}
func (p *GeminiProvider) ProviderName() string { return "gemini" }
func (p *GeminiProvider) ModelName() string    { return p.model }
func (p *GeminiProvider) AnalyzeAsAdvocate(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Advocate: cari alasan laporan VALID dan layak tayang publik", req)
}
func (p *GeminiProvider) AnalyzeAsSkeptic(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Skeptic", req)
}
func (p *GeminiProvider) AnalyzeAsManager(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	return p.analyze(ctx, "Manager", req)
}
func (p *GeminiProvider) analyze(ctx context.Context, role string, req *VerificationRequest) (*VerificationResult, error) {
	if p.client == nil {
		return nil, errors.New("gemini client is not configured")
	}
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, err
	}
	m := p.client.GenerativeModel(p.model)
	m.ResponseMIMEType = "application/json"
	resp, err := m.GenerateContent(ctx, genai.Text(rolePrompt(role, req)))
	if err != nil {
		return nil, err
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("gemini returned empty response")
	}
	text := ""
	for _, part := range resp.Candidates[0].Content.Parts {
		text += string(part.(genai.Text))
	}
	return parseResult(text)
}
