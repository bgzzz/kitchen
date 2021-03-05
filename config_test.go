package main

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
			fPath:       "./fixtures/wrong.yaml",
			expectedCfg: nil,
			err:         errors.New("unable to parse yaml"),
		},
		{
			fPath: "./fixtures/simulation-config.yaml",
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
		expectedOpts []*OrderOptions
		err          error
	}{
		{
			fPath:        "./not-existing-path",
			expectedOpts: []*OrderOptions{},
			err:          errors.New("unable to read orders"),
		},

		{
			fPath:        "./fixtures/wrong.yaml",
			expectedOpts: []*OrderOptions{},
			err:          errors.New("unable to parse"),
		},

		{
			fPath: "./fixtures/orders.json",
			expectedOpts: []*OrderOptions{
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
			expectedOpts: []*OrderOptions{
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
				var ordrs []*OrderOptions
				var err error
				if strings.Contains(test.fPath, "http") {

					t.Log("IAMHERE " + test.fPath + fmt.Sprintf(" %d", i))
					var port int
					port, err = freeport.GetFreePort()
					if err != nil {
						t.Errorf("unable to get free port: %v", err)
					}
					server := &http.Server{Addr: fmt.Sprintf(":%d",
						port),
						Handler: http.HandlerFunc(func(rw http.ResponseWriter,
							req *http.Request) {

							b, err := ioutil.ReadFile("./fixtures/orders.json")
							if err != nil {
								t.Errorf("unable to get ./fixtures/orders.json: %v",
									err)
							}
							rw.Write(b)
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

// validate orders

// validate shelves
