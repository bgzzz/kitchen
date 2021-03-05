package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testShelves = []*Shelf{
	{
		Name:               "test",
		Temp:               "test",
		Capacity:           1,
		ShelfDecayModifier: 1,
	},
	{
		Name:               "test",
		Temp:               "overflow",
		Capacity:           1,
		ShelfDecayModifier: 2,
	},
}

func TestRemoveOrder(t *testing.T) {

	tests := []struct {
		orderOpt     OrderOptions
		cfg          Config
		resultLength int
	}{

		// spoiled
		{
			orderOpt: OrderOptions{
				ShelfLife: 1,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 1,
			},
			cfg: Config{
				courierReadyMin: 10,
				courierReadyMax: 11,
			},
			resultLength: 0,
		},

		// no yet spoiled
		{
			orderOpt: OrderOptions{
				ShelfLife: 10,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 0.01,
			},
			cfg: Config{
				courierReadyMin: 10,
				courierReadyMax: 11,
			},
			resultLength: 1,
		},

		// delivered
		{
			orderOpt: OrderOptions{
				ShelfLife: 10,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 0.01,
			},
			cfg: Config{
				courierReadyMin: 0,
				courierReadyMax: 0.1,
			},
			resultLength: 0,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.orderOpt.ID, i),
			func(t *testing.T) {
				t.Parallel()

				sr := NewShelfRack(testShelves)
				sr.Init()

				order := NewOrder(&test.orderOpt, &test.cfg,
					func(o *Order) {
						sr.Interact(&OrderEvent{
							order:     o,
							eventType: OESpoiled,
						})
					}, func(o *Order) {
						sr.Interact(&OrderEvent{
							order:     o,
							eventType: OEDelivered,
						})
					})

				sr.Interact(&OrderEvent{
					eventType: OECreated,
					order:     order,
				})

				// have to wait until it ll be actually put on the shelf
				time.Sleep(2 * time.Second)
				assert.Equal(t, test.resultLength,
					len(sr.rack["test"].orders),
					"should be equal")
			})
	}

}

func TestShelfMapping(t *testing.T) {

	shelves := []*Shelf{
		{
			Name:               "target",
			Temp:               "target",
			Capacity:           1,
			ShelfDecayModifier: 1,
		},
		{
			Name:               overflowShelfName,
			Temp:               overflowShelfName,
			Capacity:           1,
			ShelfDecayModifier: 1,
		},
	}

	tests := []struct {
		orderInOverflow    *Order
		orderInTarget      *Order
		orderNewInOverflow *Order
		expectInOverflow   *Order
		expectInTarget     *Order
	}{
		{
			orderInOverflow: NewOrder(
				&OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			orderInTarget: NewOrder(
				&OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			orderNewInOverflow: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInOverflow: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInTarget: NewOrder(
				&OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
		},
		{
			orderInOverflow: NewOrder(
				&OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			orderInTarget: nil,
			orderNewInOverflow: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInOverflow: NewOrder(
				&OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInTarget: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
		},
		{
			orderInOverflow: nil,
			orderInTarget: NewOrder(
				&OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			orderNewInOverflow: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInOverflow: NewOrder(
				&OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
			expectInTarget: NewOrder(
				&OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&Config{
					courierReadyMin: 100,
					courierReadyMax: 100,
				},
				func(ord *Order) {},
				func(ord *Order) {},
			),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("wasted_%d", i),
			func(t *testing.T) {
				t.Parallel()

				sr := NewShelfRack(shelves)
				sr.Init()

				if test.orderInOverflow != nil {
					sr.rack[overflowShelfName].orders[test.orderInOverflow.opts.ID] = test.orderInOverflow
					test.orderInOverflow.Init(sr.rack[overflowShelfName].shelf)
				}

				if test.orderInTarget != nil {
					sr.rack["target"].orders[test.orderInTarget.opts.ID] = test.orderInTarget
					test.orderInTarget.Init(sr.rack["target"].shelf)
				}

				sr.Interact(&OrderEvent{
					eventType: OECreated,
					order:     test.orderNewInOverflow,
				})

				// have to wait until it ll be actually put on the shelf
				time.Sleep(2 * time.Second)

				assert.Equal(t, 1, len(sr.rack["target"].orders), "should be equal")
				assert.Equal(t, 1, len(sr.rack[overflowShelfName].orders), "should be equal")

				fmt.Println(sr.rack["target"].orders[test.expectInTarget.opts.ID].opts)
				fmt.Println(test.expectInTarget.opts)
				assert.Equal(t, true,
					reflect.DeepEqual(*sr.rack["target"].orders[test.expectInTarget.opts.ID].opts,
						*test.expectInTarget.opts),
					"should be equal")

				assert.Equal(t, true,
					reflect.DeepEqual(sr.rack[overflowShelfName].orders[test.expectInOverflow.opts.ID].opts,
						test.expectInOverflow.opts),
					"should be equal")
			})
	}

}
