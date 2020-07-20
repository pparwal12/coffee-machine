package resourcemanager

type UpdateType string

const (
	UpdateTypeConsume = "CONSUME"
	UpdateTypeRefill  = "REFILL"
)

type UpdateRequest struct {
	IngredientID     string
	UpdateType       UpdateType // can i use option???
	ResourceQuantity int
}

type GetRequest struct {
	IngredientID string
}
