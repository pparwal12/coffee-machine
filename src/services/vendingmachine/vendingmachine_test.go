package vendingmachine

import (
	"coffeeMachine/src/entities"
	"coffeeMachine/src/repository/reservationmanager"
	"coffeeMachine/src/repository/resourcemanager"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"runtime"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		numWorkers int
	}
	tests := []struct {
		name   string
		args   args
		assert func(vendingMachine CoffeeMachine)
	}{
		{
			name: "success",
			args: args{
				numWorkers: 3,
			},
			assert: func(vendingMachine CoffeeMachine) {
				assert.NotNil(t, vendingMachine)
				assert.IsType(t, &coffeeMachineImpl{}, vendingMachine)
				vendingMachineImpl := vendingMachine.(*coffeeMachineImpl)
				assert.Equal(t, 3, vendingMachineImpl.numOfOutlets)
				assert.NotNil(t, vendingMachineImpl.resourceManager)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			params := getParams(ctrl, tt.args.numWorkers)
			got := New(params)
			tt.assert(got)
		})
	}
}

func getParams(ctrl *gomock.Controller, numWorkers int) Params {
	return Params{
		ResourceManager: resourcemanager.NewMockRepository(ctrl),
		NumOfOutlets:    numWorkers,
	}
}

type testFileStructure struct {
	Machine struct {
		Outlets struct {
			NumOutlets int `json:"count_n"`
		} `json:"outlets"`
		Quantities map[string]int            `json:"total_items_quantity"`
		Beverages  map[string]map[string]int `json:"beverages"`
	} `json:"machine"`
}

func Test_coffeeMachineImpl_GetItems(t *testing.T) {
	runtime.GOMAXPROCS(8)
	ctx := context.Background()

	type fields struct {
		resourceManager    resourcemanager.Repository
		reservationManager reservationmanager.Repository
		numWorkers         int
	}
	type args struct {
		ctx              context.Context
		testDataFileName string
	}

	numOfItemWithOutcome := func(itemResponses []*entities.GetItemResponse, outcome entities.GetItemOutcome) int {
		cnt := 0
		for _, resp := range itemResponses {
			if resp.Outcome == outcome {
				cnt += 1
			}
		}
		return cnt
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		assert func(resp []*entities.GetItemResponse)
	}{
		{
			name: "success | given test input - not all items can be prepared",
			fields: fields{
				resourceManager:    resourcemanager.New(),
				reservationManager: reservationmanager.New(),
				numWorkers:         3,
			},
			args: args{
				ctx:              ctx,
				testDataFileName: "../testdata/testdata1.json",
			},
			assert: func(resp []*entities.GetItemResponse) {
				assert.Len(t, resp, 4)
				for _, ele := range resp {
					fmt.Println(ele.String())
				}
				// atleast one item should be prepared, and atleast one should be not prepared
				assert.NotZero(t, numOfItemWithOutcome(resp, entities.GetItemOutcomeNotPrepared))
				assert.NotZero(t, numOfItemWithOutcome(resp, entities.GetItemOutcomePrepared))
			},
		},
		{
			name: "success | only few items can be prepared",
			fields: fields{
				resourceManager:    resourcemanager.New(),
				reservationManager: reservationmanager.New(),
				numWorkers:         5,
			},
			args: args{
				ctx:              ctx,
				testDataFileName: "../testdata/testdata2.json",
			},
			assert: func(resp []*entities.GetItemResponse) {
				assert.Len(t, resp, 8)
				for _, ele := range resp {
					fmt.Println(ele.String())
				}
				assert.Greater(t, numOfItemWithOutcome(resp, entities.GetItemOutcomeNotPrepared), numOfItemWithOutcome(resp, entities.GetItemOutcomePrepared))
			},
		},
		{
			name: "success | all items can be prepared",
			fields: fields{
				resourceManager:    resourcemanager.New(),
				reservationManager: reservationmanager.New(),
				numWorkers:         5,
			},
			args: args{
				ctx:              ctx,
				testDataFileName: "../testdata/testdata3.json",
			},
			assert: func(resp []*entities.GetItemResponse) {
				assert.Len(t, resp, 8)
				for _, ele := range resp {
					fmt.Println(ele.String())
				}
				assert.Equal(t, numOfItemWithOutcome(resp, entities.GetItemOutcomeNotPrepared), 0)
			},
		},
		{
			name: "success | no items can be prepared",
			fields: fields{
				resourceManager:    resourcemanager.New(),
				reservationManager: reservationmanager.New(),
				numWorkers:         5,
			},
			args: args{
				ctx:              ctx,
				testDataFileName: "../testdata/testdata4.json",
			},
			assert: func(resp []*entities.GetItemResponse) {
				assert.Len(t, resp, 8)
				for _, ele := range resp {
					fmt.Println(ele.String())
				}
				assert.Equal(t, numOfItemWithOutcome(resp, entities.GetItemOutcomePrepared), 0)
			},
		},
		{
			name: "success | no items to prepare",
			fields: fields{
				resourceManager:    resourcemanager.New(),
				reservationManager: reservationmanager.New(),
				numWorkers:         5,
			},
			args: args{
				ctx:              ctx,
				testDataFileName: "../testdata/testdata5.json",
			},
			assert: func(resp []*entities.GetItemResponse) {
				assert.Len(t, resp, 0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			inputParams, err := getTestItems(tt.args.testDataFileName)
			if err != nil {
				panic(err)
			}

			params := Params{
				NumOfOutlets:       inputParams.Outlets,
				ResourceManager:    resourcemanager.New(),
				ReservationManager: reservationmanager.New(),
			}
			c := New(params)

			for _, ingredient := range inputParams.InitialInventory {
				err := c.Refill(ctx, ingredient)
				if err != nil {
					panic(err)
				}
			}

			got := c.PourDrinks(tt.args.ctx, inputParams.ItemRequests)
			respList := make([]*entities.GetItemResponse, 0)
			for elem := range got {
				respList = append(respList, elem)
			}
			tt.assert(respList)
		})
	}
}

type inputParams struct {
	Outlets          int
	InitialInventory []entities.Ingredient
	ItemRequests     []entities.Item
}

func getTestItems(fileName string) (*inputParams, error) {
	testFileContent, err := getTestFileContents(fileName)
	if err != nil {
		panic(err)
	}

	initialQuantities := fromMapToIngredients(testFileContent.Machine.Quantities)

	items := make([]entities.Item, 0)
	for k, v := range testFileContent.Machine.Beverages {
		item := entities.Item{
			ID:          k,
			Ingredients: fromMapToIngredients(v),
		}
		items = append(items, item)
	}
	return &inputParams{
		Outlets:          testFileContent.Machine.Outlets.NumOutlets,
		InitialInventory: initialQuantities,
		ItemRequests:     items,
	}, nil
}

func getTestFileContents(fileName string) (*testFileStructure, error) {
	fileContents, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	var input testFileStructure
	err = json.Unmarshal(fileContents, &input)
	if err != nil {
		return nil, err
	}
	return &input, nil
}

func fromMapToIngredients(ingredientsMap map[string]int) []entities.Ingredient {
	ingredientList := make([]entities.Ingredient, len(ingredientsMap))

	for k, v := range ingredientsMap {
		ingredient := entities.Ingredient{
			ID:       k,
			Quantity: v,
		}
		ingredientList = append(ingredientList, ingredient)
	}
	return ingredientList
}

func Benchmark_coffeeMachineImpl_GetItems1(t *testing.B) {
	runtime.GOMAXPROCS(1)
	ctx := context.Background()

	for n := 0; n < t.N; n++ {

		type fields struct {
			resourceManager    resourcemanager.Repository
			reservationManager reservationmanager.Repository
			numWorkers         int
		}
		type args struct {
			ctx              context.Context
			testDataFileName string
		}

		tests := []struct {
			name   string
			fields fields
			args   args
			assert func(resp []*entities.GetItemResponse)
		}{
			{
				name: "success | given test input - all items can be prepared",
				fields: fields{
					resourceManager:    resourcemanager.New(),
					reservationManager: reservationmanager.New(),
					numWorkers:         1,
				},
				args: args{
					ctx:              ctx,
					testDataFileName: "../testdata/benchmarkdata/testdata1_pretty.json",
				},
				assert: func(resp []*entities.GetItemResponse) {},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.B) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				params := Params{
					NumOfOutlets:       tt.fields.numWorkers,
					ResourceManager:    resourcemanager.New(),
					ReservationManager: reservationmanager.New(),
				}
				c := New(params)

				inputParams, err := getTestItems(tt.args.testDataFileName)
				if err != nil {
					panic(err)
				}

				for _, ingredient := range inputParams.InitialInventory {
					err := c.Refill(ctx, ingredient)
					if err != nil {
						panic(err)
					}
				}

				got := c.PourDrinks(tt.args.ctx, inputParams.ItemRequests)
				respList := make([]*entities.GetItemResponse, 0)
				for elem := range got {
					respList = append(respList, elem)
				}
				tt.assert(respList)
			})
		}
	}
}
