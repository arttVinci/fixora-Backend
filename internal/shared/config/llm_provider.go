package config

import (
	"github.com/arttVinci/fixora-Backend/internal/shared/dto"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewLLMProvider(config *viper.Viper, log *logrus.Logger) *dto.LLMProvider {
	apiKey := config.GetString("llm_provider.api_key")
	baseUrl := config.GetString("llm_provider.base_url")
	
	if apiKey == "" && baseUrl == "" {
		log.Warn("api key or base url is not configured")
		return nil
	}

	return &dto.LLMProvider{
		ApiKey:   apiKey,
		BaseUrl:  baseUrl,
	}
}