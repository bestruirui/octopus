package model

// GeminiGenerateContentRequest represents a Gemini API request
// Shared by both inbound and outbound transformers.
type GeminiGenerateContentRequest struct {
	Contents          []*GeminiContent        `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	Tools             []*GeminiTool           `json:"tools,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []*GeminiSafetySetting  `json:"safetySettings,omitempty"`
}

// GeminiContent represents a message content in Gemini format
type GeminiContent struct {
	Role  string        `json:"role"`
	Parts []*GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content (text, function call, etc.)
type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	InlineData       *GeminiBlob             `json:"inlineData,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
	FileData         *GeminiFileData         `json:"fileData,omitempty"`
	VideoMetadata    *GeminiVideoMetadata    `json:"videoMetadata,omitempty"`

	// Thought indicates if the part is thought from the model
	Thought bool `json:"thought,omitempty"`

	// ThoughtSignature is an opaque signature for the thought
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

// GeminiBlob represents inline binary data
type GeminiBlob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

// GeminiFileData represents a reference to a file
type GeminiFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// GeminiVideoMetadata contains video-specific metadata
type GeminiVideoMetadata struct {
	StartOffset string `json:"startOffset,omitempty"`
	EndOffset   string `json:"endOffset,omitempty"`
}

// GeminiFunctionCall represents a function call from the model
type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// GeminiFunctionResponse represents a function call result
type GeminiFunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GeminiTool represents a tool/function definition
type GeminiTool struct {
	FunctionDeclarations []*GeminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
	CodeExecution        *GeminiCodeExecution         `json:"codeExecution,omitempty"`
}

// GeminiFunctionDeclaration describes a function that can be called
type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// GeminiCodeExecution represents code execution capability
type GeminiCodeExecution struct{}

// GeminiGenerationConfig controls generation parameters
type GeminiGenerationConfig struct {
	Temperature        *float64      `json:"temperature,omitempty"`
	TopP               *float64      `json:"topP,omitempty"`
	TopK               *int          `json:"topK,omitempty"`
	CandidateCount     int           `json:"candidateCount,omitempty"`
	MaxOutputTokens    int           `json:"maxOutputTokens,omitempty"`
	StopSequences      []string      `json:"stopSequences,omitempty"`
	ResponseMimeType   string        `json:"responseMimeType,omitempty"`
	ResponseSchema     *GeminiSchema `json:"responseSchema,omitempty"`
	ResponseModalities []string      `json:"responseModalities,omitempty"`

	// ThinkingConfig is the thinking features configuration
	ThinkingConfig *GeminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

// GeminiSchema for structured output
type GeminiSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]*GeminiSchema `json:"properties,omitempty"`
	Items      *GeminiSchema            `json:"items,omitempty"`
	Required   []string                 `json:"required,omitempty"`
	Enum       []string                 `json:"enum,omitempty"`
}

// GeminiThinkingConfig is the thinking features configuration
type GeminiThinkingConfig struct {
	// IncludeThoughts indicates whether to include thoughts in the response
	IncludeThoughts bool `json:"includeThoughts,omitempty"`

	// ThinkingBudget is the thinking budget in tokens
	ThinkingBudget *int64 `json:"thinkingBudget,omitempty"`

	// ThinkingLevel is the level of thoughts tokens that the model should generate
	ThinkingLevel string `json:"thinkingLevel,omitempty"`
}

// GeminiSafetySetting configures content safety filtering
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiGenerateContentResponse represents a Gemini API response
type GeminiGenerateContentResponse struct {
	Candidates     []*GeminiCandidate    `json:"candidates,omitempty"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *GeminiUsageMetadata  `json:"usageMetadata,omitempty"`
	ModelVersion   string                `json:"modelVersion,omitempty"`
}

// GeminiCandidate represents a generated response candidate
type GeminiCandidate struct {
	Content       *GeminiContent        `json:"content,omitempty"`
	FinishReason  *string               `json:"finishReason,omitempty"`
	Index         int                   `json:"index"`
	SafetyRatings []*GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiSafetyRating represents content safety evaluation
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
	Blocked     bool   `json:"blocked,omitempty"`
}

// GeminiPromptFeedback provides feedback on the prompt
type GeminiPromptFeedback struct {
	BlockReason   string                `json:"blockReason,omitempty"`
	SafetyRatings []*GeminiSafetyRating `json:"safetyRatings,omitempty"`
}

// GeminiUsageMetadata provides token usage information
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount"`

	// CachedContentTokenCount is the number of tokens in the cached content
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`

	// ThoughtsTokenCount is the number of tokens in the model's thoughts
	ThoughtsTokenCount int `json:"thoughtsTokenCount,omitempty"`
}
