package resourcemanager

import (
	"coffeeMachine/src/entities"
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		assert func(repository Repository)
	}{
		{
			name: "success | get repository",
			assert: func(repository Repository) {
				assert.NotNil(t, repository)
				assert.IsType(t, &repositoryImpl{}, repository)
				concreteRepositoryImpl := repository.(*repositoryImpl)
				assert.NotNil(t, concreteRepositoryImpl.availableResources)
				assert.NotNil(t, concreteRepositoryImpl.mutex)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New()
			tt.assert(got)
		})
	}
}

func Test_managerImpl_GetIngredient(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		mutex              map[string]*sync.RWMutex
		availableResources map[string]int
	}
	type args struct {
		ctx    context.Context
		getReq GetRequest
	}

	const (
		_IngredientID = "Ingredient1234"
	)

	mutexes := make(map[string]*sync.RWMutex, 0)
	tests := []struct {
		name   string
		fields fields
		args   args
		assert func(ingredient *entities.Ingredient, err error)
	}{
		{
			name: "success | new ingredient",
			args: args{
				ctx: ctx,
				getReq: GetRequest{
					IngredientID: _IngredientID,
				},
			},
			fields: fields{
				availableResources: map[string]int{
					_IngredientID: 5,
				},
				mutex: mutexes,
			},
			assert: func(ingredient *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ingredient)
				assert.Equal(t, _IngredientID, ingredient.ID)
			},
		},
		{
			name: "error | resource not found",
			args: args{
				ctx: ctx,
				getReq: GetRequest{
					IngredientID: _IngredientID,
				},
			},
			fields: fields{
				availableResources: make(map[string]int, 0),
				mutex:              mutexes,
			},
			assert: func(ingredient *entities.Ingredient, err error) {
				assert.EqualError(t, err, entities.ErrResourceNotAvailable.Error())
				assert.Nil(t, ingredient)
			},
		},
	}
	for _, testIdx := range tests {
		tt := testIdx
		m := &repositoryImpl{
			mutex:              tt.fields.mutex,
			availableResources: tt.fields.availableResources,
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := m.GetIngredient(tt.args.ctx, tt.args.getReq)
			tt.assert(got, err)
		})
	}
}

func Test_managerImpl_UpdateIngredient(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		mutex              map[string]*sync.RWMutex
		availableResources map[string]int
	}
	type args struct {
		ctx       context.Context
		updateReq UpdateRequest
	}

	const (
		_IngredientID = "Ingredient1234"
	)

	mutexes := make(map[string]*sync.RWMutex, 0)
	tests := []struct {
		name   string
		fields fields
		args   args
		assert func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error)
	}{
		{
			name: "success | update-action = refill, new ingredient",
			args: args{
				ctx: ctx,
				updateReq: UpdateRequest{
					IngredientID:     _IngredientID,
					UpdateType:       UpdateTypeRefill,
					ResourceQuantity: 5,
				},
			},
			fields: fields{
				mutex:              mutexes,
				availableResources: map[string]int{},
			},
			assert: func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ingredient)
				assert.Equal(t, 5, ingredient.Quantity)
			},
		},
		{
			name: "success | update-action = refill, ingredient already present with some quantity",
			args: args{
				ctx: ctx,
				updateReq: UpdateRequest{
					IngredientID:     _IngredientID,
					UpdateType:       UpdateTypeRefill,
					ResourceQuantity: 5,
				},
			},
			fields: fields{
				mutex: mutexes,
				availableResources: map[string]int{
					_IngredientID: 5,
				},
			},
			assert: func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ingredient)
				assert.Equal(t, 10, ingredient.Quantity)
			},
		},
		{
			name: "success | update-action = consume, ingredient not already present",
			args: args{
				ctx: ctx,
				updateReq: UpdateRequest{
					IngredientID:     _IngredientID,
					UpdateType:       UpdateTypeConsume,
					ResourceQuantity: 5,
				},
			},
			fields: fields{
				mutex:              mutexes,
				availableResources: map[string]int{},
			},
			assert: func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error) {
				assert.EqualError(t, err, entities.ErrResourceNotAvailable.Error())
			},
		},
		{
			name: "success | update-action = consume, ingredient already present with some quantity",
			args: args{
				ctx: ctx,
				updateReq: UpdateRequest{
					IngredientID:     _IngredientID,
					UpdateType:       UpdateTypeConsume,
					ResourceQuantity: 5,
				},
			},
			fields: fields{
				mutex: mutexes,
				availableResources: map[string]int{
					_IngredientID: 10,
				},
			},
			assert: func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.Equal(t, 5, ingredient.Quantity)
			},
		},
		{
			name: "success | update-action = consume, ingredient already present with same quantity, quantity becomes 0 after update",
			args: args{
				ctx: ctx,
				updateReq: UpdateRequest{
					IngredientID:     _IngredientID,
					UpdateType:       UpdateTypeConsume,
					ResourceQuantity: 5,
				},
			},
			fields: fields{
				mutex: mutexes,
				availableResources: map[string]int{
					_IngredientID: 5,
				},
			},
			assert: func(repositoryImpl repositoryImpl, ingredient *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.NotContains(t, repositoryImpl.availableResources, _IngredientID)
				assert.NotContains(t, repositoryImpl.mutex, _IngredientID)
			},
		},
	}
	for _, testIdx := range tests {
		tt := testIdx
		m := repositoryImpl{
			mutex:              tt.fields.mutex,
			availableResources: tt.fields.availableResources,
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := m.UpdateIngredient(tt.args.ctx, tt.args.updateReq)
			tt.assert(m, got, err)
		})
	}
}
