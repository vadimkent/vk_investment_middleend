package snapshots

import (
	"strings"

	"github.com/project/vk-investment-middleend/internal/components"
	"github.com/project/vk-investment-middleend/internal/i18n"
)

// DeleteModalID is the DOM id of the snapshots delete confirmation modal.
const DeleteModalID = "snapshots-delete-modal"

// BuildDeleteModal returns the component tree for the delete-snapshot confirmation modal.
// s is the snapshot to delete; p is the current list context (preserved in the submit endpoint);
// lang is the UI language.
func BuildDeleteModal(s *Snapshot, p ListParams, lang string) components.Component {
	submitEndpoint := buildListURL("/actions/snapshots/"+s.ID, p.IsFullSnapshot, p.Offset)

	formattedDate := formatRecordedAt(s.RecordedAt)
	confirmTemplate := i18n.T(lang, "snapshots.delete.confirm")
	message := strings.ReplaceAll(confirmTemplate, "{date}", formattedDate)

	bodyText := components.Text("snapshots-delete-message", message, "md", "normal")
	bodyCol := components.ColumnWithGap("snapshots-delete-fields", "md", bodyText)

	// Cancel: replace ModalSlotID with an empty tree (dismiss pattern).
	cancelAction := components.Action{
		Trigger:  "click",
		Type:     "replace",
		TargetID: ModalSlotID,
	}
	cancelBtn := components.ButtonFull("snapshots-delete-cancel", i18n.T(lang, "common.cancel"), "", "secondary", "ghost",
		cancelAction)

	// Delete: submit DELETE to the snapshot endpoint.
	deleteBtn := components.ButtonFull("snapshots-delete-submit", i18n.T(lang, "snapshots.delete.submit"), "", "destructive", "solid",
		components.Submit(submitEndpoint, "DELETE", ScreenID),
	)

	actionsRow := components.RowWithGap("snapshots-delete-actions", []string{"1fr", "auto", "auto"}, "sm",
		components.Spacer("snapshots-delete-actions-spacer", "none"),
		cancelBtn,
		deleteBtn,
	)

	formBody := components.ColumnWithGap("snapshots-delete-form-body", "lg", bodyCol, actionsRow)
	form := components.Form("snapshots-delete-form", formBody)
	return components.ModalFull(DeleteModalID, i18n.T(lang, "snapshots.delete.title"), "dialog", true, true, form)
}
