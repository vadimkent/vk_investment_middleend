package trades

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
	"github.com/project/vk-investment-middleend/internal/shared/assetscatalog"
)

// ModalID is the DOM id of the trades modal root. Handlers target this id
// when re-rendering a modal on validation errors.
const ModalID = "trades-modal"

// BuildCreateModal returns the tree for the create-trade modal.
// catalog is the full asset catalog (the builder filters complex assets out).
// p is the current list context (filter + offset) so the submit endpoint
// preserves filter/pagination state across the mutation.
// inlineError, when non-empty, is rendered at the top of the form.
func BuildCreateModal(catalog []assetscatalog.Asset, p ListParams, lang, inlineError string) components.Component {
	submitEndpoint := buildSubmitURL("/actions/trades/create", p)

	eligible := filterNonComplex(catalog)
	assetOpts := make([]components.SelectOption, 0, len(eligible)+1)
	assetOpts = append(assetOpts, components.SelectOption{
		Value: "",
		Label: i18n.T(lang, "trades.filter.asset_any"),
	})
	for _, a := range eligible {
		assetOpts = append(assetOpts, components.SelectOption{Value: a.ID, Label: a.Ticker})
	}

	tradeTypeOpts := []components.SelectOption{
		{Value: "BUY", Label: i18n.T(lang, "trades.filter.type_buy")},
		{Value: "SELL", Label: i18n.T(lang, "trades.filter.type_sell")},
	}

	fields := []components.Component{}
	if inlineError != "" {
		fields = append(fields, modalError(inlineError))
	}

	fields = append(fields,
		modalSelect("trades-create-asset", "asset_id", i18n.T(lang, "trades.form.asset"), "", assetOpts, true),
		modalSelect("trades-create-trade-type", "trade_type", i18n.T(lang, "trades.form.trade_type"), "", tradeTypeOpts, true),
		modalInput("trades-create-quantity", "quantity", "text", i18n.T(lang, "trades.form.quantity"), "", true, 0, nil),
		modalInput("trades-create-price", "price_per_unit", "text", i18n.T(lang, "trades.form.price_per_unit"), "", true, 0, nil),
		modalInput("trades-create-fees", "fees", "text", i18n.T(lang, "trades.form.fees"), "0", false, 0, nil),
		modalInput("trades-create-date", "date", "date", i18n.T(lang, "trades.form.date"), "", true, 0, map[string]any{
			"max": time.Now().UTC().Format("2006-01-02"),
		}),
		modalTextarea("trades-create-notes", "notes", i18n.T(lang, "trades.form.notes"),
			i18n.T(lang, "trades.form.notes_placeholder"), "", 500, false),
	)

	noAssets := len(eligible) == 0
	if noAssets {
		fields = append(fields, components.TextStyled("trades-create-no-assets-hint",
			i18n.T(lang, "trades.form.no_assets_hint"),
			"sm", "normal", "", "muted", "", ""))
	}

	fieldsCol := components.ColumnWithGap("trades-create-fields", "md", fields...)

	cancelBtn := components.ButtonFull("trades-create-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("trades-create-submit", i18n.T(lang, "trades.create.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "POST",
			Endpoint: submitEndpoint,
			TargetID: "trades-create-form",
			Loading:  "section",
		},
	)
	if noAssets {
		submitBtn.Props["disabled"] = true
	}

	actionsRow := components.RowWithGap("trades-create-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("trades-create-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	formBody := components.ColumnWithGap("trades-create-form-body", "lg", fieldsCol, actionsRow)
	form := components.Form("trades-create-form", formBody)
	return components.ModalFull(ModalID, i18n.T(lang, "trades.create.title"), "dialog", true, true, form)
}

// BuildEditModal returns the tree for the edit-trade modal.
// `date` and `source` are rendered as static labeled text (immutable per the
// backend contract); all other financial fields are editable inputs with the
// trade's current values as defaults.
func BuildEditModal(t Trade, catalog []assetscatalog.Asset, p ListParams, lang, inlineError string) components.Component {
	submitEndpoint := buildSubmitURL("/actions/trades/"+t.ID, p)

	eligible := filterNonComplex(catalog)
	assetOpts := make([]components.SelectOption, 0, len(eligible)+1)
	assetOpts = append(assetOpts, components.SelectOption{
		Value: "",
		Label: i18n.T(lang, "trades.filter.asset_any"),
	})
	for _, a := range eligible {
		assetOpts = append(assetOpts, components.SelectOption{Value: a.ID, Label: a.Ticker})
	}

	tradeTypeOpts := []components.SelectOption{
		{Value: "BUY", Label: i18n.T(lang, "trades.filter.type_buy")},
		{Value: "SELL", Label: i18n.T(lang, "trades.filter.type_sell")},
	}

	// Resolve ticker for the title (UUID fallback if not in catalog).
	ticker := t.AssetID
	for _, a := range catalog {
		if a.ID == t.AssetID {
			ticker = a.Ticker
			break
		}
	}

	feesDefault := t.Fees
	if feesDefault == "" {
		feesDefault = "0"
	}

	fields := []components.Component{}
	if inlineError != "" {
		fields = append(fields, modalError(inlineError))
	}

	// Immutable fields as static labeled text.
	fields = append(fields,
		staticLabeled("trades-edit-date-static", i18n.T(lang, "trades.form.date_readonly"), dateOnly(t.Date)),
		staticLabeled("trades-edit-source-static", i18n.T(lang, "trades.form.source_readonly"), strings.ToUpper(t.Source)),
	)

	// Mutable fields.
	fields = append(fields,
		modalSelect("trades-edit-asset", "asset_id", i18n.T(lang, "trades.form.asset"), t.AssetID, assetOpts, true),
		modalSelect("trades-edit-trade-type", "trade_type", i18n.T(lang, "trades.form.trade_type"), t.TradeType, tradeTypeOpts, true),
		modalInput("trades-edit-quantity", "quantity", "text", i18n.T(lang, "trades.form.quantity"), t.Quantity, true, 0, nil),
		modalInput("trades-edit-price", "price_per_unit", "text", i18n.T(lang, "trades.form.price_per_unit"), t.PricePerUnit, true, 0, nil),
		modalInput("trades-edit-fees", "fees", "text", i18n.T(lang, "trades.form.fees"), feesDefault, false, 0, nil),
		modalTextarea("trades-edit-notes", "notes", i18n.T(lang, "trades.form.notes"),
			i18n.T(lang, "trades.form.notes_placeholder"), t.Notes, 500, false),
	)

	fieldsCol := components.ColumnWithGap("trades-edit-fields", "md", fields...)

	cancelBtn := components.ButtonFull("trades-edit-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("trades-edit-submit", i18n.T(lang, "trades.edit.submit"), "", "primary", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "PATCH",
			Endpoint: submitEndpoint,
			TargetID: "trades-edit-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("trades-edit-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("trades-edit-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	title := interpolateTitle(i18n.T(lang, "trades.edit.title"), dateOnly(t.Date), ticker)

	formBody := components.ColumnWithGap("trades-edit-form-body", "lg", fieldsCol, actionsRow)
	form := components.Form("trades-edit-form", formBody)
	return components.ModalFull(ModalID, title, "dialog", true, true, form)
}

// BuildDeleteModal returns the tree for the delete-trade confirmation modal.
func BuildDeleteModal(t Trade, catalog []assetscatalog.Asset, p ListParams, lang, inlineError string) components.Component {
	submitEndpoint := buildSubmitURL("/actions/trades/"+t.ID, p)

	ticker := t.AssetID
	for _, a := range catalog {
		if a.ID == t.AssetID {
			ticker = a.Ticker
			break
		}
	}

	message := interpolateDeleteConfirm(i18n.T(lang, "trades.delete.confirm"),
		strings.ToUpper(t.TradeType), t.Quantity, ticker, dateOnly(t.Date))

	bodyChildren := []components.Component{}
	if inlineError != "" {
		bodyChildren = append(bodyChildren, modalError(inlineError))
	}
	bodyChildren = append(bodyChildren,
		components.Text("trades-delete-message", message, "md", "normal"),
	)
	bodyCol := components.ColumnWithGap("trades-delete-fields", "md", bodyChildren...)

	cancelBtn := components.ButtonFull("trades-delete-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		components.Dismiss())
	submitBtn := components.ButtonFull("trades-delete-submit", i18n.T(lang, "trades.delete.submit"), "", "destructive", "solid",
		components.Action{
			Trigger:  "click",
			Type:     "submit",
			Method:   "DELETE",
			Endpoint: submitEndpoint,
			TargetID: "trades-delete-form",
			Loading:  "section",
		},
	)
	actionsRow := components.RowWithGap("trades-delete-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("trades-delete-actions-spacer", "none"),
		cancelBtn,
		submitBtn,
	)

	formBody := components.ColumnWithGap("trades-delete-form-body", "lg", bodyCol, actionsRow)
	form := components.Form("trades-delete-form", formBody)
	return components.ModalFull(ModalID, i18n.T(lang, "trades.delete.title"), "dialog", true, true, form)
}

// -------- helpers --------

// buildSubmitURL appends the current list context (asset_id, trade_type, offset)
// as query params to the given base path, so mutation handlers preserve the
// filter/pagination state the user was looking at. Empty params are omitted.
func buildSubmitURL(basePath string, p ListParams) string {
	q := url.Values{}
	if p.AssetID != "" {
		q.Set("asset_id", p.AssetID)
	}
	if p.TradeType != "" {
		q.Set("trade_type", p.TradeType)
	}
	if p.Offset > 0 {
		q.Set("offset", strconv.Itoa(p.Offset))
	}
	if len(q) == 0 {
		return basePath
	}
	return basePath + "?" + q.Encode()
}

func filterNonComplex(catalog []assetscatalog.Asset) []assetscatalog.Asset {
	out := make([]assetscatalog.Asset, 0, len(catalog))
	for _, a := range catalog {
		if a.IsComplex {
			continue
		}
		out = append(out, a)
	}
	return out
}

func modalError(message string) components.Component {
	return components.TextStyled("trades-modal-error", message, "sm", "normal", "", "negative", "", "")
}

func modalInput(id, name, inputType, label, defaultValue string, required bool, maxLength int, extra map[string]any) components.Component {
	props := map[string]any{
		"name":       name,
		"input_type": inputType,
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
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	for k, v := range extra {
		props[k] = v
	}
	return components.Component{Type: "input", ID: id, Props: props}
}

func modalSelect(id, name, label, defaultValue string, opts []components.SelectOption, required bool) components.Component {
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
	return components.Component{Type: "select", ID: id, Props: props}
}

func modalTextarea(id, name, label, placeholder, defaultValue string, maxLength int, required bool) components.Component {
	props := map[string]any{
		"name": name,
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
	if maxLength > 0 {
		props["max_length"] = maxLength
	}
	if required {
		props["required"] = true
	}
	return components.Component{Type: "textarea", ID: id, Props: props}
}

// staticLabeled renders an immutable field as plain labeled text (e.g.
// "Date: 2024-03-15"). Mirrors the approach used by the assets edit modal.
func staticLabeled(id, label, value string) components.Component {
	return components.Text(id, fmt.Sprintf("%s: %s", label, value), "sm", "normal")
}

// interpolateTitle replaces {date} / {ticker} in a registered i18n template.
func interpolateTitle(template, date, ticker string) string {
	return strings.NewReplacer("{date}", date, "{ticker}", ticker).Replace(template)
}

// interpolateDeleteConfirm replaces {type}/{quantity}/{ticker}/{date} in a
// registered i18n template.
func interpolateDeleteConfirm(template, tradeType, quantity, ticker, date string) string {
	return strings.NewReplacer(
		"{type}", tradeType,
		"{quantity}", quantity,
		"{ticker}", ticker,
		"{date}", date,
	).Replace(template)
}
