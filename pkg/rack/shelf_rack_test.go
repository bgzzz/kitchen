package rack

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	ordrs "github.com/bgzzz/kitchen/pkg/orders"
	shvs "github.com/bgzzz/kitchen/pkg/shelves"
	"github.com/bgzzz/kitchen/pkg/stats"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var testShelves = []*shvs.Shelf{
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
		orderOpt     ordrs.OrderOptions
		cfg          ordrs.Config
		resultLength int
	}{

		// spoiled
		{
			orderOpt: ordrs.OrderOptions{
				ShelfLife: 1,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 1,
			},
			cfg: ordrs.Config{
				CourierReadyMin: 10,
				CourierReadyMax: 11,
			},
			resultLength: 0,
		},

		// no yet spoiled
		{
			orderOpt: ordrs.OrderOptions{
				ShelfLife: 10,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 0.01,
			},
			cfg: ordrs.Config{
				CourierReadyMin: 10,
				CourierReadyMax: 11,
			},
			resultLength: 1,
		},

		// delivered
		{
			orderOpt: ordrs.OrderOptions{
				ShelfLife: 10,
				ID:        "test",
				Name:      "test",
				Temp:      "test",
				DecayRate: 0.01,
			},
			cfg: ordrs.Config{
				CourierReadyMin: 0,
				CourierReadyMax: 0.1,
			},
			resultLength: 0,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%s_%d", test.orderOpt.ID, i),
			func(t *testing.T) {

				sr := NewShelfRack(logrus.NewEntry(logrus.New()),
					&stats.Stats{}, testShelves, 10, func() {})
				sr.Init()

				order := ordrs.NewOrder(&test.orderOpt, &test.cfg,
					func(o *ordrs.Order) {
						sr.Interact(&OrderEvent{
							Order:     o,
							EventType: OESpoiled,
						})
					}, func(o *ordrs.Order) {
						sr.Interact(&OrderEvent{
							Order:     o,
							EventType: OEDelivered,
						})
					})

				sr.Interact(&OrderEvent{
					EventType: OECreated,
					Order:     order,
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

	shelves := []*shvs.Shelf{
		{
			Name:               "target",
			Temp:               "target",
			Capacity:           1,
			ShelfDecayModifier: 1,
		},
		{
			Name:               shvs.OverflowShelfTemp,
			Temp:               shvs.OverflowShelfTemp,
			Capacity:           1,
			ShelfDecayModifier: 1,
		},
	}

	tests := []struct {
		orderInOverflow    *ordrs.Order
		orderInTarget      *ordrs.Order
		orderNewInOverflow *ordrs.Order
		expectInOverflow   *ordrs.Order
		expectInTarget     *ordrs.Order
	}{
		{
			orderInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			orderInTarget: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			orderNewInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInTarget: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
		},
		{
			orderInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			orderInTarget: nil,
			orderNewInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "overflow",
					Name:      "overflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInTarget: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
		},
		{
			orderInOverflow: nil,
			orderInTarget: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			orderNewInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInOverflow: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "newInOverflow",
					Name:      "newInOverflow",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
			expectInTarget: ordrs.NewOrder(
				&ordrs.OrderOptions{
					ID:        "target",
					Name:      "target",
					Temp:      "target",
					ShelfLife: 100,
				},
				&ordrs.Config{
					CourierReadyMin: 100,
					CourierReadyMax: 100,
				},
				func(ord *ordrs.Order) {},
				func(ord *ordrs.Order) {},
			),
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("wasted_%d", i),
			func(t *testing.T) {

				sr := NewShelfRack(logrus.NewEntry(logrus.New()),
					&stats.Stats{}, shelves, 10, func() {})
				sr.Init()

				if test.orderInOverflow != nil {
					sr.rack[shvs.OverflowShelfTemp].orders[test.orderInOverflow.Opts.ID] = test.orderInOverflow
					test.orderInOverflow.Init(sr.rack[shvs.OverflowShelfTemp].shelf)
				}

				if test.orderInTarget != nil {
					sr.rack["target"].orders[test.orderInTarget.Opts.ID] = test.orderInTarget
					test.orderInTarget.Init(sr.rack["target"].shelf)
				}

				sr.Interact(&OrderEvent{
					EventType: OECreated,
					Order:     test.orderNewInOverflow,
				})

				// have to wait until it ll be actually put on the shelf
				time.Sleep(2 * time.Second)

				assert.Equal(t, 1, len(sr.rack["target"].orders), "should be equal")
				assert.Equal(t, 1, len(sr.rack[shvs.OverflowShelfTemp].orders), "should be equal")

				assert.Equal(t, true,
					reflect.DeepEqual(*sr.rack["target"].orders[test.expectInTarget.Opts.ID].Opts,
						*test.expectInTarget.Opts),
					"should be equal")

				assert.Equal(t, true,
					reflect.DeepEqual(sr.rack[shvs.OverflowShelfTemp].orders[test.expectInOverflow.Opts.ID].Opts,
						test.expectInOverflow.Opts),
					"should be equal")
			})
	}

}
