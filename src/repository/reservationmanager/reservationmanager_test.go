package reservationmanager

import (
	"coffeeMachine/src/entities"
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		assert func(r Repository)
	}{
		{
			name: "success",
			assert: func(r Repository) {
				assert.NotNil(t, r)
				assert.IsType(t, r, &repositoryImpl{})
				concreteImpl := r.(*repositoryImpl)
				assert.NotNil(t, concreteImpl.reservations)
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

func Test_repositoryImpl_Create(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		mutex        sync.RWMutex
		reservations map[string]int
	}
	type args struct {
		ctx     context.Context
		request CreateReservationRequest
	}

	const _IngredientID = "1234"
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: ctx,
				request: CreateReservationRequest{
					IngredientID:    _IngredientID,
					ReserveQuantity: 5,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			if err := r.Create(tt.args.ctx, tt.args.request); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_repositoryImpl_Delete(t *testing.T) {
	ctx := context.Background()

	type args struct {
		ctx     context.Context
		request DeleteReservationRequest
	}

	const _IngredientID = "1234"
	tests := []struct {
		name           string
		args           args
		initQuantities []entities.Ingredient
		assert         func(err error)
	}{
		{
			name: "error - not present quantity",
			args: args{
				ctx: ctx,
				request: DeleteReservationRequest{
					IngredientID:   _IngredientID,
					DeleteQuantity: 10,
				},
			},
			assert: func(err error) {
				assert.Error(t, err)
				assert.IsType(t, err, entities.ErrInsufficientResource{})
			},
		},
		{
			name: "success - already present quantity",
			args: args{
				ctx: ctx,
				request: DeleteReservationRequest{
					IngredientID:   _IngredientID,
					DeleteQuantity: 10,
				},
			},
			initQuantities: []entities.Ingredient{
				{ID: _IngredientID, Quantity: 20},
			},
			assert: func(err error) {
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			for _, ing := range tt.initQuantities {
				createReq := CreateReservationRequest{
					IngredientID:    ing.ID,
					ReserveQuantity: ing.Quantity,
				}
				err := r.Create(ctx, createReq)
				if err != nil {
					log.Fatal(err)
				}
			}
			err := r.Delete(tt.args.ctx, tt.args.request)
			tt.assert(err)
		})
	}
}

func Test_repositoryImpl_Get(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		mutex        sync.RWMutex
		reservations map[string]int
	}
	type args struct {
		ctx     context.Context
		request GetReservationRequest
	}

	const _IngredientID = "1234"

	tests := []struct {
		name   string
		fields fields
		args   args
		assert func(ing *entities.Ingredient, err error)
	}{
		{
			name: "success",
			fields: fields{
				mutex:        sync.RWMutex{},
				reservations: make(map[string]int, 0),
			},
			args: args{
				ctx: ctx,
				request: GetReservationRequest{
					IngredientID: _IngredientID,
				},
			},
			assert: func(ing *entities.Ingredient, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ing)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &repositoryImpl{
				mutex:        tt.fields.mutex,
				reservations: tt.fields.reservations,
			}
			got, err := r.Get(tt.args.ctx, tt.args.request)
			tt.assert(got, err)
		})
	}
}
