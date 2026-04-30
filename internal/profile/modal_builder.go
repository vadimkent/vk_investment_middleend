package profile

import (
	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// BuildDeleteModal renders the delete-account confirmation modal. errMessage is
// non-empty when re-rendering after a validation error.
func BuildDeleteModal(lang, errMessage string) components.Component {
	body := []components.Component{
		components.Text("delete-modal-body", i18n.T(lang, "profile.danger.modal.body"), "sm", "regular"),
	}
	if errMessage != "" {
		body = append(body, components.TextStyled("delete-modal-error", errMessage, "sm", "regular", "block", "error", "", ""))
	}
	passwordInput := components.InputAdvanced(components.InputOptions{
		ID: "input-delete-password", Name: "password", InputType: "password",
		Label:           i18n.T(lang, "profile.danger.modal.password_label"),
		Required:        true,
		RequiredMessage: i18n.T(lang, "validation.required"),
	})
	cancelBtn := components.ButtonFull("delete-cancel-btn",
		i18n.T(lang, "profile.danger.modal.cancel"),
		"", "secondary", "ghost",
		components.Dismiss())
	confirmBtn := components.ButtonFull("delete-confirm-btn",
		i18n.T(lang, "profile.danger.modal.confirm"),
		"", "destructive", "solid",
		components.Submit("/actions/profile/delete_account", "POST", "delete-form"))
	actionsRow := components.RowWithGap("delete-actions",
		[]string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("delete-actions-spacer", "none"),
		cancelBtn,
		confirmBtn,
	)
	formBody := components.ColumnWithGap("delete-form-body", "lg", passwordInput, actionsRow)
	form := components.Form("delete-form", formBody)
	body = append(body, form)
	return components.ModalFull(DeleteModalID,
		i18n.T(lang, "profile.danger.modal.title"),
		"dialog", true, true, body...)
}
