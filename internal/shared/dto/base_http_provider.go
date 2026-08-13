package dto

type LLMProvider struct {
	ApiKey  string
	BaseUrl string
}

type LLMGenerateContentRequest struct {
	Model   string
	Content string
}

type LLMRequest struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
}
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMGenerateContentResponse struct {
	Content string `json:"content"`
}

// Struct buat parsing response dari commandcode.ai
type LLMChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}