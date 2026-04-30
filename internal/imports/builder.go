package imports

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

const (
	maxAnalyzeBytes = 5 * 1024 * 1024
	maxRestoreBytes = 10 * 1024 * 1024

	acceptAnalyze = ".csv,.tsv,.xls,.xlsx,.txt"
	acceptRestore = ".csv"
)

// BuildRoot is the top-level screen tree at GET /screens/import.
func BuildRoot(lang string) components.Component {
	header := buildHeader(lang)
	section := components.Component{
		Type: "column",
		ID:   "import-section",
		Props: map[string]any{"gap": "lg"},
		Children: []components.Component{
			BuildAIImportCardIdle(lang, "", "", ""),
			buildExportRestoreGroup(lang),
		},
	}
	root := components.Component{
		Type: "column",
		ID:   "import-root",
		Props: map[string]any{"gap": "lg"},
		Children: []components.Component{header, section, BuildEmptyModalSlot()},
	}
	return components.Screen("import-screen", i18n.T(lang, "import.title"), root)
}

// BuildEmptyModalSlot is the empty modal-slot sibling under import-root.
func BuildEmptyModalSlot() components.Component {
	return components.Component{
		Type:  "column",
		ID:    "import-modal-slot",
		Props: map[string]any{},
	}
}

func buildHeader(lang string) components.Component {
	return components.Component{
		Type: "row",
		ID:   "import-header",
		Props: map[string]any{
			"align_items": "center",
		},
		Children: []components.Component{
			components.Text("import-title", i18n.T(lang, "import.title"), "xl", "bold"),
		},
	}
}

func buildExportRestoreGroup(lang string) components.Component {
	return components.Component{
		Type: "row",
		ID:   "export-restore-group",
		Props: map[string]any{
			"gap":             "md",
			"grid_template":   []string{"1fr", "1fr"},
			"stack_on_mobile": true,
			"align_items":     "stretch",
		},
		Children: []components.Component{
			BuildExportCard(lang),
			BuildRestoreCardIdle(lang, "", ""),
		},
	}
}

// BuildAIImportCardIdle returns the ai-import-card in idle state. errorMessage,
// prefillFilename, and prefillHint are all optional. When errorMessage is set,
// an inline error banner is rendered above the file upload. When
// prefillFilename is set, the file upload renders the "previously uploaded"
// state with a re-attach hint. When prefillHint is set, it pre-fills the hint
// textarea.
func BuildAIImportCardIdle(lang, errorMessage, prefillFilename, prefillHint string) components.Component {
	children := make([]components.Component, 0, 6)

	children = append(children,
		components.Text("ai-import-title", i18n.T(lang, "import.ai.title"), "lg", "bold"),
		components.Text("ai-import-description", i18n.T(lang, "import.ai.description"), "sm", "normal"),
	)

	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "banner",
			ID:   "ai-import-error",
			Props: map[string]any{
				"variant": "error",
				"message": errorMessage,
			},
		})
	}

	upload := components.FileUpload("import-file", components.FileUploadProps{
		Name:               "file",
		Label:              i18n.T(lang, "import.upload.label"),
		Placeholder:        i18n.T(lang, "import.upload.placeholder"),
		Hint:               i18n.T(lang, "import.upload.hint_ai"),
		Accept:             acceptAnalyze,
		MaxSizeBytes:       maxAnalyzeBytes,
		ErrorMessageSize:   i18n.T(lang, "import.upload.error_size"),
		ErrorMessageFormat: i18n.T(lang, "import.upload.error_format"),
		PrefillFilename:    prefillFilename,
		ReattachHint:       i18n.T(lang, "import.upload.reattach_hint"),
	})
	children = append(children, upload)

	hint := components.Component{
		Type: "textarea",
		ID:   "import-hint",
		Props: map[string]any{
			"name":        "hint",
			"label":       i18n.T(lang, "import.hint.label"),
			"placeholder": i18n.T(lang, "import.hint.placeholder"),
			"max_length":  500,
		},
	}
	if prefillHint != "" {
		hint.Props["value"] = prefillHint
	}
	children = append(children, hint)

	submitBtn := components.ButtonFull("import-analyze-btn", i18n.T(lang, "import.analyze"),
		"", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/analyze",
			Method:   "POST",
			TargetID: "import-modal-slot",
			Loading:  components.LoadingFullWithMessages(analyzeLoadingMessages(lang)),
		},
	)
	submitBtn.Props["size"] = "sm"
	actions := components.RowWithGap("ai-import-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("ai-import-actions-spacer", "none"),
		submitBtn,
	)
	children = append(children, actions)

	body := components.ColumnWithGap("ai-import-card-body", "lg", children...)
	return components.Card("ai-import-card", body)
}

func analyzeLoadingMessages(lang string) []string {
	keys := []string{
		"import.loading.analyze.1",
		"import.loading.analyze.2",
		"import.loading.analyze.3",
		"import.loading.analyze.4",
		"import.loading.analyze.5",
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, i18n.T(lang, k))
	}
	return out
}

// BuildExportCard renders the static Export card.
func BuildExportCard(lang string) components.Component {
	exportBtn := components.ButtonFull("export-btn", i18n.T(lang, "import.export.submit"),
		"", "primary", "solid",
		components.Download("/actions/import/export"),
	)
	exportBtn.Props["size"] = "sm"
	actions := components.RowWithGap("export-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("export-actions-spacer", "none"),
		exportBtn,
	)
	body := components.ColumnWithGap("export-card-body", "lg",
		components.Text("export-title", i18n.T(lang, "import.export.title"), "lg", "bold"),
		components.Text("export-description", i18n.T(lang, "import.export.description"), "sm", "normal"),
		actions,
	)
	return components.Card("export-card", body)
}

// BuildRestoreCardIdle returns the restore-card in idle state.
func BuildRestoreCardIdle(lang, errorMessage, prefillFilename string) components.Component {
	children := make([]components.Component, 0, 6)
	children = append(children,
		components.Text("restore-title", i18n.T(lang, "import.restore.title"), "lg", "bold"),
		components.Text("restore-description", i18n.T(lang, "import.restore.description"), "sm", "normal"),
	)
	if errorMessage != "" {
		children = append(children, components.Component{
			Type: "banner", ID: "restore-error",
			Props: map[string]any{"variant": "error", "message": errorMessage},
		})
	}

	upload := components.FileUpload("restore-file", components.FileUploadProps{
		Name:               "file",
		Label:              i18n.T(lang, "import.upload.label"),
		Placeholder:        i18n.T(lang, "import.upload.placeholder"),
		Hint:               i18n.T(lang, "import.upload.hint_restore"),
		Accept:             acceptRestore,
		MaxSizeBytes:       maxRestoreBytes,
		ErrorMessageSize:   i18n.T(lang, "import.upload.error_size"),
		ErrorMessageFormat: i18n.T(lang, "import.upload.error_format"),
		PrefillFilename:    prefillFilename,
		ReattachHint:       i18n.T(lang, "import.upload.reattach_hint"),
	})
	children = append(children, upload)

	submitBtn := components.ButtonFull("restore-submit-btn", i18n.T(lang, "import.restore.submit"),
		"", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Endpoint: "/actions/import/restore",
			Method:   "POST",
			TargetID: "restore-card",
			Loading:  "section",
		},
	)
	submitBtn.Props["size"] = "sm"
	actions := components.RowWithGap("restore-actions", []string{"1fr", "auto"}, "sm",
		components.Spacer("restore-actions-spacer", "none"),
		submitBtn,
	)
	children = append(children, actions)

	body := components.ColumnWithGap("restore-card-body", "lg", children...)
	return components.Card("restore-card", body)
}
