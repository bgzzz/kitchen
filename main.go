package main

import (
	"log"
	"math/rand"
	"os"
	"time"

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
			cfg, err := NewSimulationConfig(c.String(flagConfig))
			if err != nil {
				return err
			}

			ordOpts, err := FetchOrders(cfg.OrdersPath)
			if err != nil {
				return err
			}

			if err := ValidateOrderOptions(ordOpts); err != nil {
				return err
			}

			shelves, err := FetchShelves(cfg.ShelvesFilePath)
			if err != nil {
				return err
			}

			if err := ValidateShelves(shelves); err != nil {
				return err
			}

			logger := logrus.New()
			debug := c.Bool(flagDebug)

			if debug {
				logger.SetLevel(logrus.DebugLevel)
			}

			log := logrus.NewEntry(logger)

			return run(log, cfg, shelves, ordOpts)

			// jwtSecret, err := decodeJWTSecret(c.String(flagJWTSecret))
			// if err != nil {
			// 	return err
			// }

			// var hc *http.Client
			// issuerCaCertPath := c.String(flagIssuerCACert)
			// if issuerCaCertPath != "" {
			// 	c, err := httpClientForRootCAs(issuerCaCertPath)
			// 	if err != nil {
			// 		return err
			// 	}
			// 	hc = c
			// }
			// if hc == nil {
			// 	hc = http.DefaultClient
			// }

			// ctx := oidc.ClientContext(context.Background(), hc)
			// issuerURL := c.String(flagIssuerURL)
			// provider, err := oidc.NewProvider(ctx, issuerURL)
			// if err != nil {
			// 	return errors.Wrap(err, "could not create new oidc provider")
			// }

			// clientID := c.String(flagClientID)

			// s := &kaideauth.Server{
			// 	Log:          logrus.NewEntry(logger),
			// 	JWTSecret:    jwtSecret,
			// 	OIDCClient:   hc,
			// 	OIDCurl:      issuerURL,
			// 	OIDCVerifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
			// 	OAuth2Config: &oauth2.Config{
			// 		ClientID:     clientID,
			// 		ClientSecret: c.String(flagClientSecret),
			// 		Endpoint:     provider.Endpoint(),
			// 		Scopes:       []string{"openid", "email", "profile", "groups", "offline_access"},
			// 		RedirectURL:  c.String(flagRedirectURL),
			// 	},
			// }

			// log.Println("listening on :8080...")
			// return http.ListenAndServe(":8080", s.Handler())
			// return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        flagConfig,
				Required:    true,
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

func run(log *logrus.Entry, cfg *SimulationConfig,
	shelves []*Shelf, ordOpts []*OrderOptions) error {
	done := make(chan bool)
	sr := NewShelfRack(log, shelves, len(ordOpts), func() {
		done <- true
	})
	sr.Init()

	orderTicker := time.NewTicker(1 * time.Second)

	go func(ordOpts []*OrderOptions, sr *ShelfRack) {
		for {
			if len(ordOpts) == 0 {
				break
			}
			<-orderTicker.C
			for i := 0; i < cfg.OrdersConfig.OrdersPerSecond; i++ {
				if len(ordOpts) == 0 {
					break
				}

				go func(orderOpts OrderOptions) {
					time.Sleep(time.Duration(rand.Float64()) * time.Second)

					order := NewOrder(&orderOpts, &Config{
						courierReadyMin: cfg.OrdersConfig.DeliveryMinSeconds,
						courierReadyMax: cfg.OrdersConfig.DeliveryMaxSeconds,
					}, func(ord *Order) {
						sr.Interact(&OrderEvent{
							eventType: OESpoiled,
							order:     ord,
						})
					}, func(ord *Order) {
						sr.Interact(&OrderEvent{
							eventType: OEDelivered,
							order:     ord,
						})
					})

					sr.Interact(&OrderEvent{
						eventType: OECreated,
						order:     order,
					})
				}(*ordOpts[0])

				ordOpts = ordOpts[1:]
			}

		}
	}(ordOpts, sr)

	<-done
	return nil
}
