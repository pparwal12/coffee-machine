package reservationmanager

type CreateReservationRequest struct {
	IngredientID    string
	ReserveQuantity int
}

type GetReservationRequest struct {
	IngredientID string
}

type DeleteReservationRequest struct {
	IngredientID   string
	DeleteQuantity int
}
