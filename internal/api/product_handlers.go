package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/miguelscastro/ignite/go/05-gobid/internal/jsonutils"
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

	id, err := api.ProductService.CreateProduct(
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

	jsonutils.EncodeJSON(w, r, http.StatusCreated, map[string]any{
		"message":    "product created succesfully",
		"product_id": id,
	})
}
