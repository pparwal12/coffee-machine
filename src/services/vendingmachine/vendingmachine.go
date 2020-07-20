package vendingmachine

import (
	"coffeeMachine/src/entities"
	"coffeeMachine/src/repository/reservationmanager"
	"coffeeMachine/src/repository/resourcemanager"
	"context"
	"sync"

	"github.com/avast/retry-go"
)

// CoffeeMachine is the interface which exposes functionalities of our coffee-machine
type CoffeeMachine interface {
	PourDrinks(ctx context.Context, items []entities.Item) <-chan *entities.GetItemResponse
	Refill(ctx context.Context, ingredient entities.Ingredient) error
}

/*
	To allow multiple requests for pouring items to work concurrently, we use a map of mutexes,
	where each ingredient-id will have its corresponding mutex.
	This allows concurrent operations on different ingredients from different items.

	Since map is not thread safe, we use another mutex to guard writes to mutexesMap
*/
type coffeeMachineImpl struct {
	resourceManager             resourcemanager.Repository
	reservationManager          reservationmanager.Repository
	numOfOutlets                int
	mutexForAccessingMutexesMap sync.Mutex
	mutexesMap                  map[string]*sync.Mutex
}

type Params struct {
	ReservationManager reservationmanager.Repository
	ResourceManager    resourcemanager.Repository
	NumOfOutlets       int
}

func New(p Params) CoffeeMachine {
	return &coffeeMachineImpl{
		resourceManager:             p.ResourceManager,
		numOfOutlets:                p.NumOfOutlets,
		mutexesMap:                  make(map[string]*sync.Mutex, 0),
		reservationManager:          p.ReservationManager,
		mutexForAccessingMutexesMap: sync.Mutex{},
	}
}

// PourDrinks uses a pool of workers to call pour-drink utility
func (c *coffeeMachineImpl) PourDrinks(ctx context.Context, items []entities.Item) <-chan *entities.GetItemResponse {
	inputCh := make(chan entities.Item, 5)
	result := make(chan *entities.GetItemResponse, len(items))

	wg := sync.WaitGroup{}
	for i := 0; i < c.numOfOutlets; i += 1 {
		wg.Add(1)
		go c.worker(ctx, i, inputCh, result, &wg)
	}

	for _, item := range items {
		inputCh <- item
	}
	close(inputCh)
	wg.Wait()

	close(result)
	return result
}

func (c *coffeeMachineImpl) worker(ctx context.Context, workerID int, inputCh <-chan entities.Item, result chan<- *entities.GetItemResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	for item := range inputCh {
		result <- c.pourDrink(ctx, item)
	}
}

// pourDrink will try pouring a particular drink, retry if needed.
// Note - retry is done only in case of ErrResourceTemporarilyNotAvailable
// since it could possibly be a transient error
func (c *coffeeMachineImpl) pourDrink(ctx context.Context, item entities.Item) *entities.GetItemResponse {
	err := retry.Do(
		func() error {
			return c.attemptPouringDrink(ctx, item)
		},
		retry.RetryIf(func(err error) bool {
			if _, ok := err.(entities.ErrResourceTemporarilyNotAvailable); ok {
				return true
			}
			return false
		}),
		retry.Attempts(3),
	)

	return c.toPourDrinkResponse(item, err)
}

func (c *coffeeMachineImpl) toPourDrinkResponse(item entities.Item, err error) *entities.GetItemResponse {
	if err == nil {
		return &entities.GetItemResponse{
			Item:    item,
			Outcome: entities.GetItemOutcomePrepared,
		}
	}
	return &entities.GetItemResponse{
		Item:    item,
		Outcome: entities.GetItemOutcomeNotPrepared,
		RejectReasons: []entities.RejectReason{
			{
				err.Error(),
			},
		},
	}
}

/*
	The logic for pouring drink is as follows:
	For each ingredient of the item, we check if the ingredient & corresponding quantity is available or not,
	If yes, then we go ahead and take a reservation on the given quantity [ the actual resource quantity is still
 	the same, its just a reservation saying that this much quantity is unusable currently by other requests ].

	Now, if reservations were successful for all ingredients, we actually update the ingredient quantities,
	Then we delete the reservations.

	In case where during acquiring reservations, for some ingredient reservation wasn't possible
	[ probably because enough quantity is absent ], we release all already taken reservations.

	Now it can happen that for an item, reservations were taken for some of the ingredients [ but not all ].
 	If another request for such a reserved ingredient came up, we return an error - resourceTemporarilyUnavailable
	The caller will retry a fixed number of times if it recieves this error [ so that the probability of user
	getting a drink improves ]

*/
func (c *coffeeMachineImpl) attemptPouringDrink(ctx context.Context, item entities.Item) (err error) {
	reservedItemsIdx := len(item.Ingredients)

	defer func() {
		// Note: we are deleting all reservations as part of defer.
		// A more efficient approach would be to delete right after the resource-consume call,
		// but then we would have to trach which reservations have already been deleted.
		deleteErr := c.deleteReservations(ctx, item.Ingredients[:reservedItemsIdx])
		if deleteErr != nil {
			err = deleteErr
		}
	}()

	for idx, ingredient := range item.Ingredients {
		err := c.reserveIngredientIfPossible(ctx, ingredient)
		if err != nil {
			reservedItemsIdx = idx
			return err
		}
	}

	// if all reservations were successful, reflect the consumption from the actual resource them
	// ideally since all reservations were successful, consumeIngredient would fail in cases of unreliable storage,
	// which we are not considering for now. If we were to consider, we will have re-fill all the already consumed ingredients
	for _, ingredient := range item.Ingredients {
		err = c.consumeIngredient(ctx, ingredient)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *coffeeMachineImpl) reserveIngredientIfPossible(ctx context.Context, toReserveIngredient entities.Ingredient) error {
	mutex := c.getOrCreateMutex(ctx, toReserveIngredient)
	mutex.Lock()
	defer mutex.Unlock()

	resourceGetReq := resourcemanager.GetRequest{
		IngredientID: toReserveIngredient.ID,
	}
	availableIngredient, err := c.resourceManager.GetIngredient(ctx, resourceGetReq)
	if err != nil {
		return err
	}

	reservationGetReq := reservationmanager.GetReservationRequest{
		IngredientID: toReserveIngredient.ID,
	}
	reservedIngredient, err := c.reservationManager.Get(ctx, reservationGetReq)
	if err != nil {
		return err
	}

	// if the desired quantity is already readily available, create a new reservation
	if availableIngredient.Quantity-reservedIngredient.Quantity >= toReserveIngredient.Quantity {
		reservationCreateReq := reservationmanager.CreateReservationRequest{
			IngredientID:    toReserveIngredient.ID,
			ReserveQuantity: toReserveIngredient.Quantity,
		}
		err = c.reservationManager.Create(ctx, reservationCreateReq)
		if err != nil {
			return err
		}
		return nil
	}

	// if enough resource is not available readily right now, still there could be a case where
	// there is some existing reservation for the ingredient, which could possibly fail later on - if all ingredients aren't available
	// so if there is a chance of the request quantity being available from (availableQuantity + reservedQuantity), throw a custom error, and retry
	if availableIngredient.Quantity >= toReserveIngredient.Quantity {
		return entities.ErrResourceTemporarilyNotAvailable{ResourceID: toReserveIngredient.ID}
	}

	return entities.ErrInsufficientResource{ResourceID: toReserveIngredient.ID}
}

func (c *coffeeMachineImpl) deleteReservations(ctx context.Context, ingredients []entities.Ingredient) error {
	for _, ingredient := range ingredients {
		mutex := c.getOrCreateMutex(ctx, ingredient)
		mutex.Lock()
		deleteReq := reservationmanager.DeleteReservationRequest{
			IngredientID:   ingredient.ID,
			DeleteQuantity: ingredient.Quantity,
		}
		err := c.reservationManager.Delete(ctx, deleteReq)
		if err != nil {
			// since can't use defer in a loop
			mutex.Unlock()
			return err
		}
		mutex.Unlock()
	}
	return nil
}

func (c *coffeeMachineImpl) getOrCreateMutex(ctx context.Context, ingredient entities.Ingredient) *sync.Mutex {
	c.mutexForAccessingMutexesMap.Lock()
	defer c.mutexForAccessingMutexesMap.Unlock()

	if _, ok := c.mutexesMap[ingredient.ID]; !ok {
		c.mutexesMap[ingredient.ID] = &sync.Mutex{}
	}
	return c.mutexesMap[ingredient.ID]
}

func (c *coffeeMachineImpl) consumeIngredient(ctx context.Context, ingredient entities.Ingredient) error {
	mutex := c.getOrCreateMutex(ctx, ingredient)
	mutex.Lock()
	defer mutex.Unlock()

	updateReq := resourcemanager.UpdateRequest{
		IngredientID:     ingredient.ID,
		UpdateType:       resourcemanager.UpdateTypeConsume,
		ResourceQuantity: ingredient.Quantity,
	}
	_, err := c.resourceManager.UpdateIngredient(ctx, updateReq)
	if err != nil {
		return err
	}
	return nil
}

// Refill allows refilling some ingredient
func (c *coffeeMachineImpl) Refill(ctx context.Context, ingredient entities.Ingredient) error {
	mutex := c.getOrCreateMutex(ctx, ingredient)
	mutex.Lock()
	mutex.Unlock()

	updateReq := resourcemanager.UpdateRequest{
		IngredientID:     ingredient.ID,
		UpdateType:       resourcemanager.UpdateTypeRefill,
		ResourceQuantity: ingredient.Quantity,
	}
	_, err := c.resourceManager.UpdateIngredient(ctx, updateReq)
	if err != nil {
		return err
	}
	return nil
}
