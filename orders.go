package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Config struct {
	courierReadyMin float64
	courierReadyMax float64
}

type OrderOptions struct {
	ID   string
	Name string
	// Temp is referred shelf storage temperature
	Temp string
	// ShelfLife is shelf wait max duration (seconds)
	ShelfLife int
	// DecayRate is value deterioration modifier
	DecayRate float64
}

// Order is a structure defining the order in the kitchen
type Order struct {
	opts *OrderOptions
	cfg  *Config

	startTS       *time.Time
	shelfSwitchTS time.Time

	// timers and handlers release channels
	spoilTimer        *time.Timer
	deliveryTimer     *time.Timer
	stopSpoilingCh    chan bool
	stopDeliveryingCh chan bool

	shelfChange chan Shelf
	shelf       *Shelf

	valueLock sync.RWMutex
	value     float64

	OnSpoil   func(ord *Order)
	OnDeliver func(ord *Order)
}

// Shelf is a structure defining the kitchen shelf options
type Shelf struct {
	Name               string
	Temp               string
	Capacity           int
	ShelfDecayModifier int
}

// NewOrder creates order based on specified options
// and configuration
func NewOrder(opts *OrderOptions,
	cfg *Config,
	onSpoil func(ord *Order), onDeliver func(ord *Order)) *Order {
	return &Order{
		opts:        opts,
		cfg:         cfg,
		OnSpoil:     onSpoil,
		OnDeliver:   onDeliver,
		shelfChange: make(chan Shelf),
	}
}

// Init initializes the order structure and starts
// order shelf change (event listener) loop in addition to
// setup of spoiling and delivery timers. It has to be supplied with
// shelf object which determines the shelf where order will be initially put
func (ord *Order) Init(shelf *Shelf) {
	go ord.shelfChangerLoop()
	ord.putOnTheShelf(shelf)
}

// ChangeShelf changes current shelf of the order and eventually
// retriggers the spoil timer
func (ord *Order) ChangeShelf(shelf *Shelf) {
	fmt.Println("hello")
	ord.shelfChange <- *shelf
}

// Done releases spawned go routines regarding timers and shelf change
// event loop
func (ord *Order) Done() {
	ord.stopTimers()
	close(ord.shelfChange)
}

// onSpoilTimerFired waits until either timer
// or done channel will be fired
func (ord *Order) onSpoilTimerFired() {
	select {
	case <-ord.spoilTimer.C:
		{
			ord.OnSpoil(ord)
		}
	case <-ord.stopSpoilingCh:
		{

		}
	}
}

func (ord *Order) onDeliveryTimerFired() {
	select {
	case <-ord.deliveryTimer.C:
		{
			ord.OnDeliver(ord)
		}
	case <-ord.stopDeliveryingCh:
		{

		}
	}
}

// stopTimers stops both (spoil, delivery) timers
func (ord *Order) stopTimers() {
	ord.stopSpoiling()
	ord.stopDeliverying()
}

// calculateValueOnTheCurrentShelf calculates value of the order on the
// currently set shelf, taking into account time spent on this shelf
func (ord *Order) calculateValueOnTheCurrentShelf(elapsedSeconds float64) float64 {

	value := (float64(ord.opts.ShelfLife) -
		elapsedSeconds -
		(ord.opts.DecayRate *
			elapsedSeconds *
			float64(ord.shelf.ShelfDecayModifier))) /

		float64(ord.opts.ShelfLife)

	if value < 0 {
		value = 0
	}

	return value
}

// putOnTheShelf puts order on the shelf by changin current shelf to the
// supplied. Restarts the spoiled timer according to the newly supplied
// shelf parameter
// In case supplied shelf is initial both timers initial and delivery are
// setup
func (ord *Order) putOnTheShelf(shelf *Shelf) {

	currentTime := time.Now()

	// on shelf change
	if ord.startTS != nil {

		elapsedTillNow := float64(currentTime.UnixNano()-
			ord.startTS.UnixNano()) / 1000000000

		ord.valueLock.Lock()
		ord.value = ord.currentValue(currentTime)
		ord.valueLock.Unlock()

		timeToSpoil := ord.calculateMaxOrderAge(shelf.ShelfDecayModifier) - elapsedTillNow

		ord.shelfSwitchTS = currentTime
		ord.shelf = shelf

		ord.stopSpoiling()
		ord.startSpoiling(time.Duration(timeToSpoil) *
			time.Second)

		return
	}

	// initalisation
	// this one happens only once at start

	timeToDeliver := time.Duration(ord.cfg.courierReadyMin+
		rand.Float64()*(ord.cfg.courierReadyMax-ord.cfg.courierReadyMin)) * time.Second
	timeToSpoil := time.Duration(ord.calculateMaxOrderAge(shelf.ShelfDecayModifier)) *
		time.Second

	ord.shelf = shelf
	ord.shelfSwitchTS = currentTime
	ord.startTS = &currentTime
	ord.valueLock.Lock()
	ord.value = 1
	ord.valueLock.Unlock()

	ord.startDeliverying(timeToDeliver)
	ord.startSpoiling(timeToSpoil)
}

func (ord *Order) currentValue(currentTime time.Time) float64 {
	elapsedTillNow := float64(currentTime.UnixNano()-
		ord.startTS.UnixNano()) / 1000000000
	valueNow := ord.calculateValueOnTheCurrentShelf(elapsedTillNow)

	elapsedTillPrevShelfSwitch := float64(ord.shelfSwitchTS.UnixNano()-
		ord.startTS.UnixNano()) / 1000000000
	valueOnPrevShelfSwitch := ord.calculateValueOnTheCurrentShelf(elapsedTillPrevShelfSwitch)

	return ord.value + valueNow - valueOnPrevShelfSwitch
}

func (ord *Order) CurrentValue(currentTime time.Time) float64 {
	ord.valueLock.RLock()
	defer ord.valueLock.RUnlock()
	return ord.currentValue(currentTime)
}

// shelfChangerLoop event loop to process on shelf change events
func (ord *Order) shelfChangerLoop() {
	fmt.Println("run the loop")
	for shelf := range ord.shelfChange {
		fmt.Println(shelf)
		ord.putOnTheShelf(&shelf)
	}
}

// startDeliverying explicitly starts timer and its handler
func (ord *Order) startDeliverying(timeToDeliver time.Duration) {
	ord.deliveryTimer = time.NewTimer(timeToDeliver)
	ord.stopDeliveryingCh = make(chan bool)
	go ord.onDeliveryTimerFired()
}

func (ord *Order) startSpoiling(timeToSpoil time.Duration) {
	ord.spoilTimer = time.NewTimer(timeToSpoil)
	ord.stopSpoilingCh = make(chan bool)
	go ord.onSpoilTimerFired()
}

// stopDeliverying stops timer and releases the handler
func (ord *Order) stopDeliverying() {
	if !ord.deliveryTimer.Stop() {
		<-ord.deliveryTimer.C
	}
	// prevent go routine leaking
	select {
	case ord.stopDeliveryingCh <- true:
		{

		}
	default:
		{

		}
	}
}

func (ord *Order) stopSpoiling() {
	if !ord.spoilTimer.Stop() {
		<-ord.spoilTimer.C
	}
	// prevent go routine leaking
	select {
	case ord.stopSpoilingCh <- true:
		{

		}
	default:
		{

		}
	}
}

// calculateMaxOrderAge calculates max age of the order
// returned value is used for spoil timer calculation
func (ord *Order) calculateMaxOrderAge(shelfDecayModifier int) float64 {
	return float64(ord.opts.ShelfLife) /
		(1 + ord.opts.DecayRate*float64(shelfDecayModifier))
}
