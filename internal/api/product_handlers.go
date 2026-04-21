package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/jsonutils"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/services"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/usecase/product"
)

func (api *Api) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	data, problems, err := jsonutils.DecodeValidJSON[product.CreateProductReq](r)
	if err != nil {
		jsonutils.EncodeJSON(w, r, http.StatusUnprocessableEntity, problems)
		return
	}
	userID, ok := api.Sessions.Get(r.Context(), "AuthenticatedUserID").(uuid.UUID)
	if !ok {
		jsonutils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]any{
			"error": "unexpected error",
		})
		return
	}

	productId, err := api.ProductService.CreateProduct(
		r.Context(),
		userID,
		data.ProductName,
		data.Description,
		data.Baseprice,
		data.AuctionEnd,
	)
	if err != nil {
		jsonutils.EncodeJSON(w, r, http.StatusInternalServerError, map[string]any{
			"error": "failed to create product auction",
		})
		return
	}

	ctx, _ := context.WithDeadline(context.Background(), data.AuctionEnd)

	auctionRoom := services.NewAuctionRoom(ctx, productId, api.BidsService)

	go auctionRoom.Run()

	api.AuctionLobby.Lock()
	api.AuctionLobby.Rooms[productId] = auctionRoom
	api.AuctionLobby.Unlock()

	jsonutils.EncodeJSON(w, r, http.StatusCreated, map[string]any{
		"message":    "auction started succesfully",
		"product_id": productId,
	})
}
