package reservationmanager

import (
	"coffeeMachine/src/entities"
	"context"
	"sync"
	"time"
)

// Repository allows us to perform CRUD operatins for reservations
// TODO: implement TTL for unlocking reservations - in case some are pending for long time
type Repository interface {
	Create(ctx context.Context, request CreateReservationRequest) error
	Get(ctx context.Context, request GetReservationRequest) (*entities.Ingredient, error)
	Delete(ctx context.Context, request DeleteReservationRequest) error
}

/*
	Current reservations are stored as a map[ingredient-id]reservedQuantity
	To prevent concurrent writes to the map, we use read-write mutex
*/
type repositoryImpl struct {
	mutex        sync.RWMutex
	reservations map[string]int
}

func New() Repository {
	return &repositoryImpl{
		mutex:        sync.RWMutex{},
		reservations: make(map[string]int, 0),
	}
}

func (r *repositoryImpl) Create(ctx context.Context, request CreateReservationRequest) error {
	time.Sleep(1 * time.Microsecond)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.reservations[request.IngredientID] += request.ReserveQuantity
	return nil
}

func (r *repositoryImpl) Get(ctx context.Context, request GetReservationRequest) (*entities.Ingredient, error) {
	time.Sleep(1 * time.Microsecond)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return &entities.Ingredient{
		ID:       request.IngredientID,
		Quantity: r.reservations[request.IngredientID],
	}, nil
}

func (r *repositoryImpl) Delete(ctx context.Context, request DeleteReservationRequest) error {
	time.Sleep(1 * time.Microsecond)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	curVal := r.reservations[request.IngredientID]
	if request.DeleteQuantity > curVal {
		return entities.ErrInsufficientResource{
			ResourceID: request.IngredientID,
		}
	}
	r.reservations[request.IngredientID] -= request.DeleteQuantity
	return nil
}
