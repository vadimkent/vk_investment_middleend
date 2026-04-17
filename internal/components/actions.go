package components

// Action represents a user interaction attached to a component.
type Action struct {
	Trigger  string `json:"trigger"`
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Target   string `json:"target,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Method   string `json:"method,omitempty"`
	TargetID string `json:"target_id,omitempty"`
	Loading  string `json:"loading,omitempty"`
}

// ActionResponse is the standard response for submit/reload actions.
type ActionResponse struct {
	Action   string     `json:"action"`
	TargetID string     `json:"target_id,omitempty"`
	Tree     *Component `json:"tree,omitempty"`
	Feedback *Component `json:"feedback,omitempty"`
	Auth     *AuthInfo  `json:"auth,omitempty"`
}

// AuthInfo carries JWT info for the frontend to persist after login.
type AuthInfo struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// WithAuth attaches authentication info to the action response. Chainable.
func (r ActionResponse) WithAuth(token, expiresAt string) ActionResponse {
	r.Auth = &AuthInfo{Token: token, ExpiresAt: expiresAt}
	return r
}

// Navigate creates a navigation action.
func Navigate(url string) Action {
	return Action{Trigger: "click", Type: "navigate", URL: url, Target: "self"}
}

// NavigateBlank creates a navigation action that opens in a new tab.
func NavigateBlank(url string) Action {
	return Action{Trigger: "click", Type: "navigate", URL: url, Target: "blank"}
}

// NavigateBack creates a back navigation action.
func NavigateBack() Action {
	return Action{Trigger: "click", Type: "navigate_back"}
}

// Submit creates a form submission action.
func Submit(endpoint, method, targetID string) Action {
	return Action{Trigger: "click", Type: "submit", Endpoint: endpoint, Method: method, TargetID: targetID, Loading: "section"}
}

// Reload creates a component reload action.
func Reload(endpoint, targetID string) Action {
	return Action{Trigger: "click", Type: "reload", Endpoint: endpoint, TargetID: targetID, Loading: "section"}
}

// Refresh creates a refresh action.
func Refresh() Action {
	return Action{Trigger: "click", Type: "refresh"}
}

// OpenURL creates an external URL action.
func OpenURL(url string) Action {
	return Action{Trigger: "click", Type: "open_url", URL: url}
}

// Dismiss creates a dismiss action.
func Dismiss() Action {
	return Action{Trigger: "click", Type: "dismiss"}
}

// Logout creates a logout action.
func Logout() Action {
	return Action{Trigger: "click", Type: "logout"}
}

// ToggleSidebar creates a client-side action that toggles sidebar collapse state.
// No round-trip to the middleend. State is owned by the frontend (localStorage).
func ToggleSidebar() Action {
	return Action{Trigger: "click", Type: "toggle_sidebar"}
}

// ReplaceResponse creates an action response that replaces a component.
func ReplaceResponse(targetID string, tree Component, feedback *Component) ActionResponse {
	return ActionResponse{Action: "replace", TargetID: targetID, Tree: &tree, Feedback: feedback}
}

// NavigateResponse creates an action response that navigates to a URL.
func NavigateResponse(url string, feedback *Component) ActionResponse {
	return ActionResponse{Action: "navigate", TargetID: url, Feedback: feedback}
}

// RefreshResponse creates an action response that refreshes the current screen.
func RefreshResponse(feedback *Component) ActionResponse {
	return ActionResponse{Action: "refresh", Feedback: feedback}
}

// ErrorResponse creates an action response that only shows error feedback.
func ErrorResponse(message string) ActionResponse {
	fb := Snackbar("feedback", message, "error")
	return ActionResponse{Action: "none", Feedback: &fb}
}
