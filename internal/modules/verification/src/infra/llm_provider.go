package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"
)


type LLMProvider interface {
	AnalyzeAsAdvocate(context.Context, *VerificationRequest) (*VerificationResult, error)
	AnalyzeAsSkeptic(context.Context, *VerificationRequest) (*VerificationResult, error)
	AnalyzeAsManager(context.Context, *VerificationRequest) (*VerificationResult, error)
	ProviderName() string
	ModelName() string
}

type baseHTTPProvider struct {
	apiKey, model, provider string
	client                  *http.Client
	limiter                 *rate.Limiter
}

func rolePrompt(role string, req *VerificationRequest) string {
	return fmt.Sprintf("Kamu adalah %s dalam verifikasi laporan infrastruktur publik. Jawab HANYA JSON valid dengan field verdict(boolean), confidence(number 0-1), category_slug(string), severity(string), argument(string). Data: title=%q description=%q source_type=%q severity=%q category=%q photo=%q address=%q advocate_arg=%q advocate_verdict=%t advocate_confidence=%.2f skeptic_arg=%q skeptic_verdict=%t skeptic_confidence=%.2f", role, req.ReportTitle, req.ReportDescription, req.ReportSourceType, req.ReportSeverity, req.ReportCategory, req.ReportPhotoURL, req.ReportAddress, req.AdvocateArgument, req.AdvocateVerdict, req.AdvocateConfidence, req.SkepticArgument, req.SkepticVerdict, req.SkepticConfidence)
}
func parseResult(text string) (*VerificationResult, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end >= start {
		text = text[start : end+1]
	}
	var r VerificationResult
	if err := json.Unmarshal([]byte(text), &r); err != nil {
		return nil, err
	}
	if r.Confidence < 0 {
		r.Confidence = 0
	}
	if r.Confidence > 1 {
		r.Confidence = 1
	}
	return &r, nil
}
func (p *baseHTTPProvider) doJSON(ctx context.Context, url string, headers map[string]string, body any) ([]byte, error) {
	if p.apiKey == "" {
		return nil, errors.New(p.provider + " api key is not configured")
	}
	if p.client == nil {
		p.client = &http.Client{Timeout: 90 * time.Second}
	}
	if p.limiter != nil {
		if err := p.limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}
	payload, _ := json.Marshal(body)
	var last error
	for i := 0; i < 3; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 429 {
			last = fmt.Errorf("%s rate limited: %s", p.provider, string(data))
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("%s request failed: status %d body %s", p.provider, resp.StatusCode, string(data))
		}
		return data, nil
	}
	return nil, last
}
