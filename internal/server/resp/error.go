package resp

const (
	ErrBadRequest        = "Invalid request parameters"
	ErrInvalidJSON       = "Invalid JSON format"
	ErrInvalidParam      = "Invalid parameter"
	ErrValidation        = "Input validation failed"
	ErrDuplicateResource = "Resource already exists"
	ErrResourceNotFound  = "Resource not found"
	ErrInternalServer    = "An unexpected error occurred"
	ErrDatabase          = "Database operation failed"
	ErrUnauthorized      = "Authentication failed"
	ErrTooManyRequests   = "Too many requests"
)
