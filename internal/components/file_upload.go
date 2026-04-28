package components

// FileUploadProps captures all configuration for a file_upload component.
// Required fields: Name, Label, Placeholder. Everything else is optional and
// is omitted from the rendered Props map when its zero value is in effect.
type FileUploadProps struct {
	Name               string
	Label              string
	Placeholder        string
	Hint               string
	Accept             string
	MaxSizeBytes       int64
	ErrorMessageSize   string
	ErrorMessageFormat string
	PrefillFilename    string
	ReattachHint       string
}

// FileUpload creates a file_upload custom component. See
// spec/sdui-custom-components.md §4 for the contract.
func FileUpload(id string, p FileUploadProps) Component {
	props := map[string]any{
		"name":        p.Name,
		"label":       p.Label,
		"placeholder": p.Placeholder,
	}
	if p.Hint != "" {
		props["hint"] = p.Hint
	}
	if p.Accept != "" {
		props["accept"] = p.Accept
	}
	if p.MaxSizeBytes > 0 {
		props["max_size_bytes"] = p.MaxSizeBytes
	}
	if p.ErrorMessageSize != "" {
		props["error_message_size"] = p.ErrorMessageSize
	}
	if p.ErrorMessageFormat != "" {
		props["error_message_format"] = p.ErrorMessageFormat
	}
	if p.PrefillFilename != "" {
		props["prefill_filename"] = p.PrefillFilename
	}
	if p.ReattachHint != "" {
		props["reattach_hint"] = p.ReattachHint
	}
	return Component{
		Type:  "file_upload",
		ID:    id,
		Props: props,
	}
}
