package transformers

// ErrorCode represents an error code
type ErrorCode string

// Error codes
const (
	ErrSuccess            ErrorCode = "success"
	ErrInvalidRequest     ErrorCode = "invalid_request_error"
	ErrAuthentication     ErrorCode = "authentication_error"
	ErrPermission         ErrorCode = "permission_error"
	ErrNotFound           ErrorCode = "not_found_error"
	ErrConflict           ErrorCode = "conflict_error"
	ErrRateLimit          ErrorCode = "rate_limit_error"
	ErrInternal           ErrorCode = "internal_error"
	ErrServiceUnavailable ErrorCode = "service_unavailable_error"
)

// TraeErrorCode represents Trae CN error codes
type TraeErrorCode string

// Trae CN error codes
const (
	TraeErrSuccess            TraeErrorCode = "0"
	TraeErrInvalidParam       TraeErrorCode = "400"
	TraeErrUnauthorized       TraeErrorCode = "401"
	TraeErrForbidden          TraeErrorCode = "403"
	TraeErrNotFound           TraeErrorCode = "404"
	TraeErrConflict           TraeErrorCode = "409"
	TraeErrRateLimit          TraeErrorCode = "429"
	TraeErrInternal           TraeErrorCode = "500"
	TraeErrServiceUnavailable TraeErrorCode = "503"
)

// ErrorMapper maps Trae CN errors to OpenAI errors
type ErrorMapper struct {
	codeMap map[TraeErrorCode]ErrorCode
	httpMap map[TraeErrorCode]int
}

// NewErrorMapper creates a new error mapper
func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{
		codeMap: buildErrorCodeMap(),
		httpMap: buildHTTPCodeMap(),
	}
}

// buildErrorCodeMap builds the error code mapping
func buildErrorCodeMap() map[TraeErrorCode]ErrorCode {
	return map[TraeErrorCode]ErrorCode{
		TraeErrSuccess:            ErrSuccess,
		TraeErrInvalidParam:       ErrInvalidRequest,
		TraeErrUnauthorized:       ErrAuthentication,
		TraeErrForbidden:          ErrPermission,
		TraeErrNotFound:           ErrNotFound,
		TraeErrConflict:           ErrConflict,
		TraeErrRateLimit:          ErrRateLimit,
		TraeErrInternal:           ErrInternal,
		TraeErrServiceUnavailable: ErrServiceUnavailable,
	}
}

// buildHTTPCodeMap builds the HTTP status code mapping
func buildHTTPCodeMap() map[TraeErrorCode]int {
	return map[TraeErrorCode]int{
		TraeErrSuccess:            200,
		TraeErrInvalidParam:       400,
		TraeErrUnauthorized:       401,
		TraeErrForbidden:          403,
		TraeErrNotFound:           404,
		TraeErrConflict:           409,
		TraeErrRateLimit:          429,
		TraeErrInternal:           500,
		TraeErrServiceUnavailable: 503,
	}
}

// MapErrorCode maps a Trae CN error code to OpenAI error code
func (em *ErrorMapper) MapErrorCode(traeCode TraeErrorCode) ErrorCode {
	if code, ok := em.codeMap[traeCode]; ok {
		return code
	}
	return ErrInternal
}

// MapHTTPCode maps a Trae CN error code to HTTP status code
func (em *ErrorMapper) MapHTTPCode(traeCode TraeErrorCode) int {
	if code, ok := em.httpMap[traeCode]; ok {
		return code
	}
	return 500
}

// ErrorResponse represents an OpenAI-compatible error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Message string    `json:"message"`
	Type    ErrorCode `json:"type"`
	Param   *string   `json:"param,omitempty"`
	Code    *string   `json:"code,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, errType ErrorCode) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
		},
	}
}

// NewErrorResponseWithParam creates a new error response with parameter
func NewErrorResponseWithParam(message string, errType ErrorCode, param string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
			Param:   &param,
		},
	}
}
