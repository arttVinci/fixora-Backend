package config

import (
	"context"
	"log"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

func NewGoogleAiStudio(config *viper.Viper) *genai.Client {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, option.WithAPIKey(config.GetString("google_ai_studio.api_key")))
	if err != nil {
		log.Printf("Failed to create Google AI Studio client: %s", config.GetString("google_ai_studio.api_key"))
		log.Fatal(err)
	}

	return client
}