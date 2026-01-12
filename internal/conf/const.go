package conf

const (
	APP_NAME = "octopus"
	APP_DESC = "all ai service in one place"
)

// Provider domains that don't support request metadata
var NoMetadataProvider = []string{
	"cerebras.ai",
}

