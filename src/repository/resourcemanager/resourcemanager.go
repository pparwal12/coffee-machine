package resourcemanager

import (
	"coffeeMachine/src/entities"
	"context"
	"sync"
	"time"
)

// Repository manages the inventory - provides methods to get/modify the inventory
type Repository interface {
	UpdateIngredient(ctx context.Context, updateReq UpdateRequest) (*entities.Ingredient, error)
	GetIngredient(ctx context.Context, getReq GetRequest) (*entities.Ingredient, error)
}

/*
	Currently available resources are stored as a map[ingredient-id]quantity
	To prevent concurrent writes to the map, we use read-write mutex
*/
type repositoryImpl struct {
	mutex              sync.RWMutex
	availableResources map[string]int
}

func New() Repository {
	return &repositoryImpl{
		mutex:              sync.RWMutex{},
		availableResources: make(map[string]int, 0),
	}
}

func (m *repositoryImpl) UpdateIngredient(ctx context.Context, updateReq UpdateRequest) (*entities.Ingredient, error) {
	time.Sleep(1 * time.Microsecond)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch updateReq.UpdateType {
	case UpdateTypeConsume:
		if val, ok := m.availableResources[updateReq.IngredientID]; !ok || val < updateReq.ResourceQuantity {
			return nil, entities.ErrResourceNotAvailable{ResourceID: updateReq.IngredientID}
		}

		m.availableResources[updateReq.IngredientID] = m.availableResources[updateReq.IngredientID] - updateReq.ResourceQuantity
	case UpdateTypeRefill:
		m.availableResources[updateReq.IngredientID] = m.availableResources[updateReq.IngredientID] + updateReq.ResourceQuantity
	}

	return &entities.Ingredient{
		ID:       updateReq.IngredientID,
		Quantity: m.availableResources[updateReq.IngredientID],
	}, nil
}

func (m *repositoryImpl) GetIngredient(ctx context.Context, getReq GetRequest) (*entities.Ingredient, error) {
	time.Sleep(1 * time.Microsecond)

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if _, ok := m.availableResources[getReq.IngredientID]; !ok {
		return nil, entities.ErrResourceNotAvailable{ResourceID: getReq.IngredientID}
	}
	return &entities.Ingredient{
		ID:       getReq.IngredientID,
		Quantity: m.availableResources[getReq.IngredientID],
	}, nil
}
