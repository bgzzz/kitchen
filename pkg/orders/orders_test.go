package orders

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	shvs "github.com/bgzzz/kitchen/pkg/shelves"
	"github.com/stretchr/testify/assert"
)

func TestMaxOrder(t *testing.T) {

	tests := []struct {
		opts       OrderOptions
		shelfDecay int
		result     float64
	}{
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 300,
				DecayRate: 0.5,
			},
			shelfDecay: 1,
			result:     200.0,
		},
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 300,
				DecayRate: 0,
			},
			shelfDecay: 1,
			result:     300.0,
		},
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 300,
				DecayRate: 1,
			},
			shelfDecay: 1,
			result:     150.0,
		},
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 0,
				DecayRate: 1,
			},
			shelfDecay: 1,
			result:     0,
		},
	}

	dummyFunc := func(o *Order) {}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.opts.ID, i),
			func(t *testing.T) {
				t.Parallel()
				order := NewOrder(&test.opts, &Config{},
					dummyFunc, dummyFunc)
				assert.Equal(t, test.result,
					order.calculateMaxOrderAge(test.shelfDecay),
					"should be equal")
			})
	}

}

// // spoil timer

func TestOrderTimers(t *testing.T) {
	tests := []struct {
		cfg         Config
		opts        OrderOptions
		isDelivered bool
	}{
		{
			cfg: Config{
				CourierReadyMin: 10,
				CourierReadyMax: 11,
			},
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 1,
				DecayRate: 1,
			},
			isDelivered: false,
		},
		{
			cfg: Config{
				CourierReadyMin: 0,
				CourierReadyMax: 1,
			},
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 100,
				DecayRate: 1,
			},
			isDelivered: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.opts.ID, i),
			func(t *testing.T) {
				t.Parallel()

				delivery := make(chan bool)
				spoil := make(chan bool)

				deliveryCB := func(o *Order) {
					delivery <- true
				}

				spoilCB := func(o *Order) {
					spoil <- true
				}

				shelf := shvs.Shelf{
					Name:               "some",
					ShelfDecayModifier: 2,
				}

				order := NewOrder(&test.opts, &test.cfg,
					spoilCB, deliveryCB)

				order.Init(&shelf)

				select {
				case <-delivery:
					{
						if !test.isDelivered {
							t.Errorf("Order should be spoiled before delivered")
						}
					}
				case <-spoil:
					{
						if test.isDelivered {
							t.Errorf("Order should be delivered before spoiled")
						}
					}
				}

			})
	}

}

func TestCalculateValueOnCurrentShelf(t *testing.T) {
	tests := []struct {
		opts           OrderOptions
		shelf          shvs.Shelf
		elapsedSeconds float64
		result         float64
	}{
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 2,
				DecayRate: 1,
			},
			shelf: shvs.Shelf{
				Name:               "some",
				ShelfDecayModifier: 1,
			},
			elapsedSeconds: 2.0,
			result:         -1,
		},
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 2,
				DecayRate: 1,
			},
			shelf: shvs.Shelf{
				Name:               "some",
				ShelfDecayModifier: 1,
			},
			elapsedSeconds: 10.0,
			result:         -9,
		},
		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 2,
				DecayRate: 1,
			},
			shelf: shvs.Shelf{
				Name:               "some",
				ShelfDecayModifier: 0,
			},
			elapsedSeconds: 0.0,
			result:         1,
		},

		{
			opts: OrderOptions{
				ID:        "some_id",
				Name:      "pizza",
				Temp:      "hot",
				ShelfLife: 100,
				DecayRate: 0.5,
			},
			shelf: shvs.Shelf{
				Name:               "some",
				ShelfDecayModifier: 1,
			},
			elapsedSeconds: 20.0,
			result:         0.7,
		},
	}

	dummyFunc := func(o *Order) {}
	config := Config{
		CourierReadyMin: 10,
		CourierReadyMax: 12,
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.opts.ID, i),
			func(t *testing.T) {
				order := NewOrder(&test.opts, &config,
					dummyFunc, dummyFunc)

				order.Init(&test.shelf)
				defer order.Done()

				assert.Equal(t, test.result,
					order.calculateValueOnTheCurrentShelf(test.elapsedSeconds),
					"should be equal")

			})
	}

}

func TestShelfChange(t *testing.T) {

	delivered := make(chan bool)

	ordr := NewOrder(&OrderOptions{
		ID:        "some",
		Name:      "some",
		Temp:      "some",
		ShelfLife: 100,
		DecayRate: 0.001,
	}, &Config{
		CourierReadyMin: 1,
		CourierReadyMax: 2,
	}, func(ord *Order) {}, func(ord *Order) {
		delivered <- true
	})

	shelf1 := &shvs.Shelf{
		Name:               "shelf1",
		Temp:               "some",
		Capacity:           1,
		ShelfDecayModifier: 1,
	}

	shelf2 := &shvs.Shelf{
		Name:               "shelf2",
		Temp:               "some",
		Capacity:           1,
		ShelfDecayModifier: 1,
	}

	ordr.Init(shelf1)

	ordr.ChangeShelf(shelf2)

	<-delivered
	assert.Equal(t, true, reflect.DeepEqual(*(ordr.Shelf), *shelf2),
		fmt.Sprintf("shelf in order %v should be equal %v",
			ordr.Shelf, shelf2))

}

func TestCurrentValue(t *testing.T) {
	tests := []struct {
		ordr    *OrderOptions
		shelves []*shvs.Shelf
		timeSec int
		result  float64
	}{
		{
			ordr: &OrderOptions{
				ID:        "some",
				Name:      "some",
				Temp:      "some",
				ShelfLife: 1,
				DecayRate: 1,
			},
			shelves: []*shvs.Shelf{
				{
					Name:               "some",
					Temp:               "some",
					Capacity:           1,
					ShelfDecayModifier: 1,
				},
			},
			timeSec: 1,
			result:  -1.01,
		},
		{
			ordr: &OrderOptions{
				ID:        "some",
				Name:      "some",
				Temp:      "some",
				ShelfLife: 6,
				DecayRate: 1,
			},
			shelves: []*shvs.Shelf{
				{
					Name:               "some",
					Temp:               "some",
					Capacity:           1,
					ShelfDecayModifier: 1,
				},
				{
					Name:               "some",
					Temp:               "some",
					Capacity:           1,
					ShelfDecayModifier: 1,
				},
			},
			timeSec: 1,
			result:  0.33,
		},
		{
			ordr: &OrderOptions{
				ID:        "some",
				Name:      "some",
				Temp:      "some",
				ShelfLife: 6,
				DecayRate: 1,
			},
			shelves: []*shvs.Shelf{
				{
					Name:               "some",
					Temp:               "some",
					Capacity:           1,
					ShelfDecayModifier: 1,
				},
				{
					Name:               "some",
					Temp:               "some",
					Capacity:           1,
					ShelfDecayModifier: 2,
				},
			},
			timeSec: 1,
			result:  0.16,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.ordr.ID, i),
			func(t *testing.T) {
				order := NewOrder(test.ordr, &Config{
					CourierReadyMin: 10,
					CourierReadyMax: 11,
				},
					func(ord *Order) {}, func(ord *Order) {})

				order.Init(test.shelves[0])
				defer order.Done()
				time.Sleep(time.Duration(test.timeSec) * time.Second)

				for i := 1; i < len(test.shelves); i++ {
					order.ChangeShelf(test.shelves[i])
					time.Sleep(time.Duration(test.timeSec) * time.Second)
				}

				assert.Equal(t, test.result,
					math.Floor(order.CurrentValue(time.Now())*100)/100,
					"should be equal")

			})
	}
}
