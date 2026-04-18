package assets

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// VisibleWhenValue is an alias exposing components.VisibleWhen so tests in this
// package can type-assert without importing components in the test file.
type VisibleWhenValue = components.VisibleWhen

// BuildCreateModal returns the tree for the create-asset modal.
// listParams is the current filter/offset so the submit endpoint preserves list state.
// errMsg, when non-empty, is rendered at the top of the form.
func BuildCreateModal(listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/create?" + mutationQuery(listParams)

	assetTypeOpts := []components.SelectOption{
		{Value: "STOCK", Label: "STOCK"},
		{Value: "ETF", Label: "ETF"},
		{Value: "CRYPTO", Label: "CRYPTO"},
		{Value: "BOND", Label: "BOND"},
	}
	currencyOpts := []components.SelectOption{
		{Value: "USD", Label: "USD"},
		{Value: "EUR", Label: "EUR"},
		{Value: "ARS", Label: "ARS"},
		{Value: "MXN", Label: "MXN"},
		{Value: "GBP", Label: "GBP"},
	}
	providerOpts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "COINGECKO", Label: "COINGECKO"},
		{Value: "TWELVE_DATA", Label: "TWELVE_DATA"},
		{Value: "ALPHA_VANTAGE", Label: "ALPHA_VANTAGE"},
	}

	fields := []components.Component{}
	if errMsg != "" {
		fields = append(fields, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}
	fields = append(fields,
		input("create-ticker", "ticker", "text", i18n.T(lang, "assets.col.ticker"), "", "", true, 20, map[string]any{
			"pattern":        `^[A-Z0-9.\-]+$`,
			"auto_uppercase": true,
		}, nil),
		input("create-name", "name", "text", i18n.T(lang, "assets.col.name"), "", "", true, 100, nil, nil),
		selectField("create-asset-type", "asset_type", i18n.T(lang, "assets.col.type"), "", assetTypeOpts, true, nil),
		selectField("create-currency", "currency", i18n.T(lang, "assets.col.currency"), "", currencyOpts, true, nil),
		checkboxField("create-is-complex", "is_complex", i18n.T(lang, "assets.form.is_complex"), false, nil),
		selectField("create-price-provider", "price_provider", i18n.T(lang, "assets.col.price_provider"), "", providerOpts, false,
			&components.VisibleWhen{Field: "is_complex", Op: "eq", Value: false}),
		input("create-external-ticker", "external_ticker", "text", i18n.T(lang, "assets.form.external_ticker"),
			i18n.T(lang, "assets.form.external_ticker_placeholder"), "", false, 100, nil,
			&components.VisibleWhen{Field: "price_provider", Op: "ne", Value: ""}),
	)

	fieldsCol := components.ColumnWithGap("assets-create-fields", "md", fields...)

	cancelBtn := components.ButtonFull("create-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("create-submit", i18n.T(lang, "assets.create.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "POST",
			Endpoint: submitEndpoint,
			TargetID: "assets-create-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("create-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("create-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	form := components.Form("assets-create-form", fieldsCol, actionsRow)
	return components.ModalFull("assets-create-modal", i18n.T(lang, "assets.create.title"), "dialog", true, true, form)
}

// BuildEditModal returns the tree for the edit-asset modal.
// `a` must be non-nil (handler is responsible for fetching and 404-mapping).
func BuildEditModal(a *Asset, listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/" + a.ID + "?" + mutationQuery(listParams)

	providerOpts := []components.SelectOption{
		{Value: "", Label: i18n.T(lang, "assets.filter.type_any")},
		{Value: "COINGECKO", Label: "COINGECKO"},
		{Value: "TWELVE_DATA", Label: "TWELVE_DATA"},
		{Value: "ALPHA_VANTAGE", Label: "ALPHA_VANTAGE"},
	}

	fields := []components.Component{}
	if errMsg != "" {
		fields = append(fields, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}

	// Immutable fields as static text (each a labeled line).
	fields = append(fields,
		staticField("edit-ticker-static", i18n.T(lang, "assets.col.ticker"), strings.ToUpper(a.Ticker)),
		staticField("edit-asset-type-static", i18n.T(lang, "assets.col.type"), a.AssetType),
		staticField("edit-currency-static", i18n.T(lang, "assets.col.currency"), strings.ToUpper(a.Currency)),
		staticField("edit-complex-static", i18n.T(lang, "assets.col.complex"), complexText(a.IsComplex)),
	)

	// Mutable fields.
	fields = append(fields,
		input("edit-name", "name", "text", i18n.T(lang, "assets.col.name"), "", a.Name, true, 100, nil, nil),
	)
	if !a.IsComplex {
		defaultProvider := ""
		if a.PriceProvider != nil {
			defaultProvider = *a.PriceProvider
		}
		fields = append(fields,
			selectField("edit-price-provider", "price_provider", i18n.T(lang, "assets.col.price_provider"), defaultProvider, providerOpts, false, nil),
		)
		defaultExt := ""
		if a.ExternalTicker != nil {
			defaultExt = *a.ExternalTicker
		}
		fields = append(fields,
			input("edit-external-ticker", "external_ticker", "text", i18n.T(lang, "assets.form.external_ticker"),
				i18n.T(lang, "assets.form.external_ticker_placeholder"), defaultExt, false, 100, nil,
				&components.VisibleWhen{Field: "price_provider", Op: "ne", Value: ""}),
		)
	}

	fieldsCol := components.ColumnWithGap("assets-edit-fields", "md", fields...)

	cancelBtn := components.ButtonFull("edit-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("edit-submit", i18n.T(lang, "assets.edit.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "PATCH",
			Endpoint: submitEndpoint,
			TargetID: "assets-edit-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("edit-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("edit-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	title := strings.ReplaceAll(i18n.T(lang, "assets.edit.title"), "{ticker}", strings.ToUpper(a.Ticker))
	form := components.Form("assets-edit-form", fieldsCol, actionsRow)
	return components.ModalFull("assets-edit-modal", title, "dialog", true, true, form)
}

// BuildDeleteModal returns the tree for the delete-asset confirmation modal.
func BuildDeleteModal(assetID, ticker string, listParams ListParams, lang, errMsg string) components.Component {
	submitEndpoint := "/actions/assets/" + assetID + "?" + mutationQuery(listParams)

	message := strings.ReplaceAll(i18n.T(lang, "assets.delete.confirm"), "{ticker}", strings.ToUpper(ticker))

	children := []components.Component{}
	if errMsg != "" {
		children = append(children, components.TextStyled("modal-error", errMsg, "sm", "normal", "", "negative", "", ""))
	}
	children = append(children,
		components.Text("delete-message", message, "md", "normal"),
		checkboxField("delete-force", "force", i18n.T(lang, "assets.delete.force_label"), false, nil),
	)

	cancelBtn := components.ButtonFull("delete-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("delete-submit", i18n.T(lang, "assets.delete.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "DELETE",
			Endpoint: submitEndpoint,
			TargetID: "assets-delete-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("delete-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("delete-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)
	children = append(children, actionsRow)

	form := components.Form("assets-delete-form", children...)
	return components.ModalFull("assets-delete-modal", i18n.T(lang, "assets.delete.title"), "dialog", true, true, form)
}

// -------- helpers --------

func mutationQuery(p ListParams) string {
	q := url.Values{}
	if p.AssetType != "" {
		q.Set("asset_type", p.AssetType)
	}
	q.Set("offset", strconv.Itoa(p.Offset))
	return q.Encode()
}

// input builds an input component with all optional props and a visible_when.
func input(id, name, inputType, label, placeholder, defaultValue string, required bool, maxLength int, extra map[string]any, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":       name,
		"input_type": inputType,
	}
	if label != "" {
		props["label"] = label
	}
	if placeholder != "" {
		props["placeholder"] = placeholder
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	for k, v := range extra {
		props[k] = v
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "input", ID: id, Props: props}
}

func selectField(id, name, label, defaultValue string, opts []components.SelectOption, required bool, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":    name,
		"options": opts,
	}
	if label != "" {
		props["label"] = label
	}
	if defaultValue != "" {
		props["default_value"] = defaultValue
	}
	if required {
		props["required"] = true
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "select", ID: id, Props: props}
}

func checkboxField(id, name, label string, checked bool, vw *components.VisibleWhen) components.Component {
	props := map[string]any{
		"name":  name,
		"label": label,
	}
	if checked {
		props["checked"] = true
	}
	if vw != nil {
		props["visible_when"] = *vw
	}
	return components.Component{Type: "checkbox", ID: id, Props: props}
}

func staticField(id, label, value string) components.Component {
	content := fmt.Sprintf("%s: %s", label, value)
	return components.Text(id, content, "sm", "normal")
}

func complexText(isComplex bool) string {
	if isComplex {
		return "✓"
	}
	return "—"
}
