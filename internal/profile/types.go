package profile

// User mirrors GET /v1/user/me.
type User struct {
	ID          string      `json:"id"`
	Email       string      `json:"email"`
	DisplayName *string     `json:"display_name,omitempty"`
	Preferences Preferences `json:"preferences"`
	CreatedAt   string      `json:"created_at"`
}

// Preferences carries user preferences from /v1/user/me.
type Preferences struct {
	DefaultCurrency *string `json:"default_currency,omitempty"`
}

// AppConfig is the relevant slice of GET /v1/config we use here.
type AppConfig struct {
	Currencies []string `json:"currencies"`
}
