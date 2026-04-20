package trades

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/project/vk-investment-middleend/internal/components"
)

// DeleteModalHandler serves GET /actions/trades/delete_modal: fetches the
// trade (by id) and the asset catalog — both are needed because the
// confirmation message interpolates the trade's type/quantity/date and the
// ticker from the catalog. Returns the Delete confirmation modal tree.
type DeleteModalHandler struct {
	client  tradeGetter
	catalog catalogFetcher
}

func NewDeleteModalHandler(client tradeGetter, catalog catalogFetcher) *DeleteModalHandler {
	return &DeleteModalHandler{client: client, catalog: catalog}
}

// Get validates the id, parses list-context params, fetches the trade and
// the catalog, and returns an ActionResponse that replaces the
// trades-modal-slot with the Delete modal tree. Error priority mirrors the
// Edit handler: missing id → bad query → trade errors → catalog errors.
func (h *DeleteModalHandler) Get(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		respondBadRequest(c, "missing id")
		return
	}
	params, err := parseListParams(c)
	if err != nil {
		respondBadRequest(c, err.Error())
		return
	}
	auth := c.GetHeader("Authorization")
	lang := parseLang(c)

	trade, err := h.client.GetTrade(c.Request.Context(), auth, id)
	if respondTradeFetchError(c, err, "could not load trade") {
		return
	}

	cat, err := h.catalog.List(c.Request.Context(), auth)
	if respondCatalogFetchError(c, err, "could not load assets") {
		return
	}

	modal := BuildDeleteModal(*trade, cat, params, lang, "")
	c.JSON(http.StatusOK, components.ActionResponse{
		Action:   "replace",
		TargetID: ModalSlotID,
		Tree:     &modal,
	})
}
