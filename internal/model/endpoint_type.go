package model

import "strings"

const (
	EndpointTypeAll                = "*"
	EndpointTypeChat               = "chat"
	EndpointTypeDeepSeek           = "deepseek"
	EndpointTypeResponses          = "responses"
	EndpointTypeMessages           = "messages"
	EndpointTypeEmbeddings         = "embeddings"
	EndpointTypeRerank             = "rerank"
	EndpointTypeModerations        = "moderations"
	EndpointTypeImageGeneration    = "image_generation"
	EndpointTypeAudioSpeech        = "audio_speech"
	EndpointTypeAudioTranscription = "audio_transcription"
	EndpointTypeVideoGeneration    = "video_generation"
	EndpointTypeMusicGeneration    = "music_generation"
	EndpointTypeSearch             = "search"
)

func NormalizeEndpointType(endpointType string) string {
	endpointType = strings.TrimSpace(endpointType)
	if endpointType == "" {
		return EndpointTypeAll
	}
	return strings.ToLower(endpointType)
}

func IsConversationEndpointType(endpointType string) bool {
	switch NormalizeEndpointType(endpointType) {
	case EndpointTypeChat, EndpointTypeDeepSeek, EndpointTypeResponses, EndpointTypeMessages:
		return true
	default:
		return false
	}
}
