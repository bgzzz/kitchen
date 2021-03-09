package main

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/bgzzz/kitchen/pkg/config"
	ordrs "github.com/bgzzz/kitchen/pkg/orders"
	"github.com/bgzzz/kitchen/pkg/rack"
	shvs "github.com/bgzzz/kitchen/pkg/shelves"
	"github.com/bgzzz/kitchen/pkg/stats"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	flagConfig = "simulation-config"
	flagDebug  = "debug"
)

func main() {

	app := cli.App{
		Action: func(c *cli.Context) error {
			cfg, err := config.NewSimulationConfig(c.String(flagConfig))
			if err != nil {
				return err
			}

			ordOpts, err := config.FetchOrders(cfg.OrdersPath)
			if err != nil {
				return err
			}

			if err := config.ValidateOrderOptions(ordOpts); err != nil {
				return err
			}

			shelves, err := config.FetchShelves(cfg.ShelvesFilePath)
			if err != nil {
				return err
			}

			if err := config.ValidateShelves(shelves); err != nil {
				return err
			}

			logger := logrus.New()
			debug := c.Bool(flagDebug)

			if debug {
				logger.SetLevel(logrus.DebugLevel)
			}

			log := logrus.NewEntry(logger)

			return run(log, cfg, shelves, ordOpts)

		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagConfig,
				Usage:       "Path to file containing simulation config. ex: ./kitchen.yaml",
				DefaultText: "./kitchen.yaml",
				EnvVars:     []string{"KITCHEN_SIMULATION_CONFIG_PATH"},
			},
			&cli.BoolFlag{
				Name:    flagDebug,
				Usage:   "Debug logging",
				EnvVars: []string{"KITCHEN_SIMULATION_DEBUG"},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func run(log *logrus.Entry, cfg *config.SimulationConfig,
	shelves []*shvs.Shelf, ordOpts []*ordrs.OrderOptions) error {
	done := make(chan bool)
	st := stats.NewStats(len(ordOpts))
	sr := rack.NewShelfRack(log, st, shelves, len(ordOpts), func() {
		done <- true
	})
	sr.Init()

	orderTicker := time.NewTicker(1 * time.Second)

	go func(ordOpts []*ordrs.OrderOptions, sr *rack.ShelfRack) {
		for {
			if len(ordOpts) == 0 {
				break
			}
			<-orderTicker.C
			for i := 0; i < cfg.OrdersConfig.OrdersPerSecond; i++ {
				if len(ordOpts) == 0 {
					break
				}

				go func(orderOpts ordrs.OrderOptions) {
					time.Sleep(time.Duration(rand.Float64()) * time.Second)

					order := ordrs.NewOrder(&orderOpts, &ordrs.Config{
						CourierReadyMin: cfg.OrdersConfig.DeliveryMinSeconds,
						CourierReadyMax: cfg.OrdersConfig.DeliveryMaxSeconds,
					}, func(ord *ordrs.Order) {
						sr.Interact(&rack.OrderEvent{
							EventType: rack.OESpoiled,
							Order:     ord,
						})
					}, func(ord *ordrs.Order) {
						sr.Interact(&rack.OrderEvent{
							EventType: rack.OEDelivered,
							Order:     ord,
						})
					})

					sr.Interact(&rack.OrderEvent{
						EventType: rack.OECreated,
						Order:     order,
					})
				}(*ordOpts[0])

				ordOpts = ordOpts[1:]
			}

		}
	}(ordOpts, sr)

	<-done
	return nil
}
