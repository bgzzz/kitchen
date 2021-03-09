package config

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	ordrs "github.com/bgzzz/kitchen/pkg/orders"
	shvs "github.com/bgzzz/kitchen/pkg/shelves"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
)

func TestSimulationConfig(t *testing.T) {
	tests := []struct {
		fPath       string
		expectedCfg *SimulationConfig
		err         error
	}{
		{
			fPath:       "./no-existing-path",
			expectedCfg: nil,
			err:         errors.New("unable to read config file"),
		},
		{
			fPath:       "./../../fixtures/wrong.yaml",
			expectedCfg: nil,
			err:         errors.New("unable to parse yaml"),
		},
		{
			fPath: "./../../fixtures/simulation-config.yaml",
			expectedCfg: &SimulationConfig{
				ShelvesFilePath: "./shelves.json",
				OrdersPath:      "./orders.json",
				OrdersConfig: OrdersConfig{
					OrdersPerSecond:    2,
					DeliveryMinSeconds: 4,
					DeliveryMaxSeconds: 7,
				},
			},
			err: nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("simulation_config_%d", i),
			func(t *testing.T) {
				t.Parallel()
				cfg, err := NewSimulationConfig(test.fPath)
				if test.err != nil {
					assert.Contains(t, err.Error(), test.err.Error())
				} else if err != nil {
					t.Errorf("error %v, must not be handled here", err)
				}
				assert.Equal(t, true, reflect.DeepEqual(*test.expectedCfg, *cfg))
			})
	}

}

func TestFetchOrders(t *testing.T) {
	tests := []struct {
		fPath        string
		expectedOpts []*ordrs.OrderOptions
		err          error
	}{
		{
			fPath:        "./not-existing-path",
			expectedOpts: []*ordrs.OrderOptions{},
			err:          errors.New("unable to read orders"),
		},

		{
			fPath:        "./../../fixtures/wrong.yaml",
			expectedOpts: []*ordrs.OrderOptions{},
			err:          errors.New("unable to parse"),
		},

		{
			fPath: "./../../fixtures/orders.json",
			expectedOpts: []*ordrs.OrderOptions{
				{
					ID:        "0ff534a7-a7c4-48ad-b6ec-7632e36af950",
					Name:      "Cheese Pizza",
					Temp:      "hot",
					ShelfLife: 300,
					DecayRate: 0.45,
				},
			},
			err: nil,
		},

		{
			fPath: "https://some-url",
			expectedOpts: []*ordrs.OrderOptions{
				{
					ID:        "0ff534a7-a7c4-48ad-b6ec-7632e36af950",
					Name:      "Cheese Pizza",
					Temp:      "hot",
					ShelfLife: 300,
					DecayRate: 0.45,
				},
			},
			err: nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("fetch_orders_%d", i),
			func(t *testing.T) {
				var ordrs []*ordrs.OrderOptions
				var err error
				if strings.Contains(test.fPath, "http") {

					var port int
					port, err = freeport.GetFreePort()
					if err != nil {
						t.Errorf("unable to get free port: %v", err)
					}
					server := &http.Server{Addr: fmt.Sprintf(":%d",
						port),
						Handler: http.HandlerFunc(func(rw http.ResponseWriter,
							req *http.Request) {

							b, err := ioutil.ReadFile("./../../fixtures/orders.json")
							if err != nil {
								t.Errorf("unable to get ./../../fixtures/orders.json: %v",
									err)
							}
							if _, err := rw.Write(b); err != nil {
								t.Errorf("unable to write rsp data: %s",
									err.Error())
							}
						})}

					go func() {
						if err := server.ListenAndServe(); err != nil {
							t.Log(err.Error())
						}
					}()

					defer func() {
						if err := server.Shutdown(context.Background()); err != nil {
							t.Errorf("unable to shutdown server: %v",
								err)
						}
						t.Log("shutdown")
					}()
					for i := 0; i < 3; i++ {
						ordrs, err = FetchOrders(fmt.Sprintf("http://0.0.0.0:%d",
							port))
						if err == nil {
							break
						}
						time.Sleep(1 * time.Second)
					}
				} else {
					ordrs, err = FetchOrders(test.fPath)
				}
				if test.err != nil {
					assert.Contains(t, err.Error(), test.err.Error())
				} else if err != nil {
					t.Errorf("error %v, must not be handled here", err)
				}

				for i, ordr := range ordrs {
					assert.Equal(t, true,
						reflect.DeepEqual(*test.expectedOpts[i], *ordr))
				}
			})
	}
}

// fetch shelves
func TestFetchShelves(t *testing.T) {
	tests := []struct {
		fPath   string
		shelves []*shvs.Shelf
		err     error
	}{
		{
			fPath:   "./not-existing-path.json",
			shelves: []*shvs.Shelf{},
			err:     errors.New("unable to read shelves"),
		},
		{
			fPath:   "./../../fixtures/wrong.yaml",
			shelves: []*shvs.Shelf{},
			err:     errors.New("unable to parse shelves json"),
		},
		{
			fPath: "./../../fixtures/shelves.json",
			shelves: []*shvs.Shelf{
				{
					Name:               "hot shelf",
					Temp:               "hot",
					Capacity:           10,
					ShelfDecayModifier: 1,
				},
				{
					Name:               "cold shelf",
					Temp:               "cold",
					Capacity:           10,
					ShelfDecayModifier: 1,
				},
				{
					Name:               "frozen shelf",
					Temp:               "frozen",
					Capacity:           10,
					ShelfDecayModifier: 1,
				},
				{
					Name:               "overflow shelf",
					Temp:               "any",
					Capacity:           15,
					ShelfDecayModifier: 2,
				},
			},
			err: nil,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("fetch_shelves_%d", i),
			func(t *testing.T) {
				shlvs, err := FetchShelves(test.fPath)
				if err != nil && test.err != nil {
					assert.Contains(t, err.Error(), test.err.Error())
					return
				} else if err != nil {
					t.Error("there should not be an error")
				} else if test.err != nil {
					t.Errorf("error is not returned, should be %v",
						test.err)
				}

				for i, shelf := range shlvs {
					assert.Equal(t, true, reflect.DeepEqual(
						*test.shelves[i], *shelf), fmt.Sprintf("%v != %v",
						test.shelves[i], shelf))
				}
			})
	}
}

// validate orders
func TestValidateOrders(t *testing.T) {
	tests := []struct {
		orders  []*ordrs.OrderOptions
		isError bool
	}{
		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "",
					Name:      "",
					Temp:      "",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
			},
			isError: true,
		},

		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "23q4234234",
					Name:      "",
					Temp:      "",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
			},
			isError: true,
		},

		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "234234",
					Name:      "234234234",
					Temp:      "",
					ShelfLife: -1,
					DecayRate: 0.2,
				},
			},
			isError: true,
		},

		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "aasdasdf",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 1,
					DecayRate: -0.2,
				},
			},
			isError: true,
		},

		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "aasdasdf",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 0,
					DecayRate: 0.2,
				},
			},
			isError: true,
		},

		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "aasdasdf",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
				{
					ID:        "aasdasdf",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
			},
			isError: true,
		},
		{
			orders: []*ordrs.OrderOptions{
				{
					ID:        "test1",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
				{
					ID:        "test2",
					Name:      "asdasdasd",
					Temp:      "asdasdasd",
					ShelfLife: 1,
					DecayRate: 0.2,
				},
			},
			isError: false,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("validate_orders_%d", i),
			func(t *testing.T) {
				t.Parallel()
				err := ValidateOrderOptions(test.orders)
				t.Logf("errors is: %v", err)
				assert.Equal(t, test.isError, err != nil,
					fmt.Sprintf("this subset %v has to return error",
						test.orders))
			})
	}
}

// validate shelves
