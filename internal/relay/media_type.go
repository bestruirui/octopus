package relay

// MediaEndpointType identifies the type of media endpoint being handled.
type MediaEndpointType int

const (
	MediaEndpointImageGeneration MediaEndpointType = iota
	MediaEndpointAudioSpeech
	MediaEndpointAudioTranscription
	MediaEndpointVideoGeneration
	MediaEndpointMusicGeneration
	MediaEndpointSearch
	MediaEndpointRerank
	MediaEndpointModeration
)

// mediaEndpointConfig holds per-endpoint routing configuration.
type mediaEndpointConfig struct {
	// UpstreamPath is the path appended to the channel base URL when forwarding.
	UpstreamPath string
	// BinaryResponse indicates the upstream returns non-JSON binary data (e.g. audio/mpeg).
	BinaryResponse bool
	// MultipartInput indicates the inbound request uses multipart/form-data.
	MultipartInput bool
}

// mediaEndpointConfigs maps each media endpoint type to its configuration.
var mediaEndpointConfigs = map[MediaEndpointType]mediaEndpointConfig{
	MediaEndpointImageGeneration: {
		UpstreamPath: "/v1/images/generations",
	},
	MediaEndpointAudioSpeech: {
		UpstreamPath:  "/v1/audio/speech",
		BinaryResponse: true,
	},
	MediaEndpointAudioTranscription: {
		UpstreamPath:   "/v1/audio/transcriptions",
		MultipartInput: true,
	},
	MediaEndpointVideoGeneration: {
		UpstreamPath: "/v1/videos/generations",
	},
	MediaEndpointMusicGeneration: {
		UpstreamPath: "/v1/music/generations",
	},
	MediaEndpointSearch: {
		UpstreamPath: "/v1/search",
	},
	MediaEndpointRerank: {
		UpstreamPath: "/v1/rerank",
	},
	MediaEndpointModeration: {
		UpstreamPath: "/v1/moderations",
	},
}

// getMediaEndpointConfig returns the configuration for the given endpoint type.
func getMediaEndpointConfig(endpointType MediaEndpointType) mediaEndpointConfig {
	return mediaEndpointConfigs[endpointType]
}
