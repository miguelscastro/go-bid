package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/jsonutils"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/services"
)

func (api *Api) HandleSubscribeUserToAuction(w http.ResponseWriter, r *http.Request) {
	rawProductId := chi.URLParam(r, "product_id")
	productId, err := uuid.Parse(rawProductId)
	if err != nil {
		jsonutils.EncodeJSON(w, r, http.StatusBadRequest, map[string]any{
			"message": "invalid product id - must be a valid uuid",
		})
		return
	}

	_, err = api.ProductService.GetProductById(r.Context(), productId)
	if err != nil {
		if errors.Is(err, services.ErrProductNotFound) {
			jsonutils.EncodeJSON(w, r, http.StatusNotFound, map[string]any{
				"message": "no product with given id",
			})
			return
		}
		jsonutils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]any{
			"message": "unexpected error, try again later",
		})
		return
	}
	userId, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		jsonutils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]any{
			"message": "unexpected error, try again later",
		})
		return
	}

	api.AuctionLobby.Lock()
	room, ok := api.AuctionLobby.Rooms[productId]
	api.AuctionLobby.Unlock()

	if !ok {
		jsonutils.EncodeJSON(w, r, http.StatusNotFound, map[string]any{
			"message": "this auction has ended",
		})
		return
	}

	conn, err := api.WsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		jsonutils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]any{
			"message": "could not upgrade connection to a websocket protocol",
		})
	}

	client := services.NewClient(room, conn, userId)

	room.Register <- client
	// go client.ReadEventLoop()
	// go client.WriteEventLoop()
	for {
	}
}
