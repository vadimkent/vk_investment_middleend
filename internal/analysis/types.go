package analysis

// BackendError carries a code + message that the BE returns on validation
// failures (4xx/5xx with a JSON body). The middleend forwards code and
// message to the FE so the analysis_chat component can map the code to a
// localized message client-side.
type BackendError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *BackendError) Error() string {
	return e.Code + ": " + e.Message
}
