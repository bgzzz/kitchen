package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// OrdersConfig general configuration of the orders
// created in the simulation
type OrdersConfig struct {
	OrdersPerSecond    int     `yaml:"orders-per-second"`
	DeliveryMinSeconds float64 `yaml:"delivery-min-seconds"`
	DeliveryMaxSeconds float64 `yaml:"delivery-max-seconds"`
}

// SimulationConfig general simulation configuration
type SimulationConfig struct {
	ShelvesFilePath string       `yaml:"shelves-path"`
	OrdersPath      string       `yaml:"orders-path"`
	OrdersConfig    OrdersConfig `yaml:"orders-config"`
}

// NewSimulationConfig reads configuration file and parses it
// into the structure
// return error in case of problems with file reading and yaml parsing
func NewSimulationConfig(configFilePath string) (*SimulationConfig, error) {

	b, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read config file")
	}

	var sc SimulationConfig
	if err := yaml.Unmarshal(b, &sc); err != nil {
		return nil, errors.Wrap(err, "unable to parse yaml")
	}

	return &sc, nil
}

// FetchOrders downloads order definition from a file/link parse it
// and return in list
// return errors in case of reading/parsing problems
func FetchOrders(ordersFilePath string) (opts []*OrderOptions, err error) {
	opts = []*OrderOptions{}
	err = nil
	if strings.Contains(ordersFilePath, "http") {
		res, e := http.Get(ordersFilePath)
		if e != nil {
			err = errors.Wrap(e, "unable to fetch orders")
			return
		}

		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			err = errors.Wrap(e, "unable to read http responce body")
			return
		}

		if e := json.Unmarshal(body, &opts); e != nil {
			err = errors.Wrap(e, "unable to parse orders json")
			return
		}

		return
	}

	b, e := ioutil.ReadFile(ordersFilePath)
	if e != nil {
		err = errors.Wrap(e, "unable to read orders file")
		return
	}

	if e := json.Unmarshal(b, &opts); e != nil {
		err = errors.Wrap(e, "unable to parse json")
		return
	}

	return
}

// FetchShelves reads shelves definition
// return errors in case of file reading structure parsing problems
func FetchShelves(shelvesPath string) (shelves []*Shelf, err error) {
	err = nil
	shelves = []*Shelf{}
	b, e := ioutil.ReadFile(shelvesPath)
	if e != nil {
		err = errors.Wrap(e, "unable to read shelves file")
		return
	}

	if e := json.Unmarshal(b, &shelves); e != nil {
		err = errors.Wrap(e, "unable to parse shelves json")
		return
	}

	return
}

func validateShelf(shelf *Shelf) error {
	if shelf.Name == "" {
		return errors.New("name of shelf can't be empty")
	}
	if shelf.Capacity < 0 {
		return errors.New(fmt.Sprintf("shelf %s: capacity has to be set >= 0",
			shelf.Name))
	}
	if shelf.ShelfDecayModifier < 0 {
		return errors.New(fmt.Sprintf("shelf %s: shelf decay modifier has to be integer >= 0",
			shelf.Name))
	}
	return nil
}

func validateOrderOptions(opts *OrderOptions) error {
	if opts.ID == "" {
		return errors.New("ID of order can't be empty")
	}
	if opts.Name == "" {
		return errors.New("Name of order can't be empty")
	}
	if opts.ShelfLife <= 0 {
		return errors.New(fmt.Sprintf("order %s: shelf life <= 0", opts.ID))
	}
	if opts.DecayRate < 0 {
		return errors.New(fmt.Sprintf("order %s: decay rate < 0", opts.ID))
	}
	return nil
}

// ValidateShelves validates supplied shelves structures
// return non nil error in case of invalid shelflist
func ValidateShelves(shelves []*Shelf) error {

	shelvesTempMap := map[string]struct{}{}
	shelvesNameMap := map[string]struct{}{}

	for _, shelf := range shelves {
		if _, ok := shelvesTempMap[shelf.Temp]; ok {
			return errors.New(fmt.Sprintf("shelf %s: shelf with this temp was already defined",
				shelf.Name))
		}
		if _, ok := shelvesNameMap[shelf.Name]; ok {
			return errors.New(fmt.Sprintf("shelf %s: shelf with this name was already defined",
				shelf.Name))
		}
		if err := validateShelf(shelf); err != nil {
			return errors.Wrap(err, "shelf definition is not valid")
		}
		shelvesTempMap[shelf.Temp] = struct{}{}
		shelvesNameMap[shelf.Name] = struct{}{}
	}

	return nil
}

// ValidateOrderOptions validates order option list
// return non nil error in case of non valid order options list
func ValidateOrderOptions(orderOptions []*OrderOptions) error {
	orderIDMap := map[string]struct{}{}

	for _, opts := range orderOptions {
		if _, ok := orderIDMap[opts.ID]; ok {
			return errors.New(fmt.Sprintf("order with id %s was previously defined",
				opts.ID))
		}
		if err := validateOrderOptions(opts); err != nil {
			return errors.Wrap(err, "order defintion is not valid")
		}
	}

	return nil
}
