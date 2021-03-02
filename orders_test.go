package main

import (
	"fmt"
	"testing"

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

// spoil timer

func TestOrderTimers(t *testing.T) {
	tests := []struct {
		cfg         Config
		opts        OrderOptions
		isDelivered bool
	}{
		{
			cfg: Config{
				courierReadyMin: 10,
				courierReadyMax: 11,
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
				courierReadyMin: 0,
				courierReadyMax: 1,
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

				shelf := Shelf{
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

func TestOrderWasted(t *testing.T) {
	cfg := Config{
		courierReadyMin: 10,
		courierReadyMax: 11,
	}
	opts := OrderOptions{
		ID:        "some_id",
		Name:      "pizza",
		Temp:      "hot",
		ShelfLife: 10,
		DecayRate: 1,
	}

	dummyFunc := func(o *Order) {}

	order := NewOrder(&opts, &cfg,
		dummyFunc, dummyFunc)

	shelf := Shelf{
		Name:               "some",
		ShelfDecayModifier: 2,
	}

	order.Init(&shelf)
	order.Done()

	t.Log("Should not be blocked after this line")
	for _ = range order.shelfChange {
	}

	assert.Equal(t,
		true, order.deliveryTimer.Stop(),
		"delivery timer has to be stopped")

	assert.Equal(t,
		true, order.spoilTimer.Stop(),
		"spoil timer has to be stopped")

}

// calculate value

// putOnTheShelf

// shelf switching in order for not being spoiled
