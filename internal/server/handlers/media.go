package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/relay"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/router"
)

func init() {
	// JSON endpoints: require API key auth + JSON content type
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/images/generations", http.MethodPost).
				Handle(imageGeneration),
		).
		AddRoute(
			router.NewRoute("/audio/speech", http.MethodPost).
				Handle(audioSpeech),
		).
		AddRoute(
			router.NewRoute("/videos/generations", http.MethodPost).
				Handle(videoGeneration),
		).
		AddRoute(
			router.NewRoute("/music/generations", http.MethodPost).
				Handle(musicGeneration),
		).
		AddRoute(
			router.NewRoute("/search", http.MethodPost).
				Handle(search),
		).
		AddRoute(
			router.NewRoute("/rerank", http.MethodPost).
				Handle(rerank),
		).
		AddRoute(
			router.NewRoute("/moderations", http.MethodPost).
				Handle(moderation),
		)

	// Multipart endpoints: require API key auth only (no RequireJSON)
	router.NewGroupRouter("/v1").
		Use(middleware.APIKeyAuth()).
		AddRoute(
			router.NewRoute("/images/edits", http.MethodPost).
				Handle(imageEdit),
		).
		AddRoute(
			router.NewRoute("/images/variations", http.MethodPost).
				Handle(imageVariation),
		).
		AddRoute(
			router.NewRoute("/audio/transcriptions", http.MethodPost).
				Handle(audioTranscription),
		)
}

func imageGeneration(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointImageGeneration, c)
}

func imageEdit(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointImageEdit, c)
}

func imageVariation(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointImageVariation, c)
}

func audioSpeech(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointAudioSpeech, c)
}

func audioTranscription(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointAudioTranscription, c)
}

func videoGeneration(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointVideoGeneration, c)
}

func musicGeneration(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointMusicGeneration, c)
}

func search(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointSearch, c)
}

func rerank(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointRerank, c)
}

func moderation(c *gin.Context) {
	relay.MediaHandler(relay.MediaEndpointModeration, c)
}
