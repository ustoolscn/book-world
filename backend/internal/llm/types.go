package llm

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type StreamRequest struct {
	BaseURL        string
	APIKey         string
	Model          string
	ThinkingEffort string
	Messages       []Message
}
