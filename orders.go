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

type Pair struct {
	Shelf *Shelf
	TS    time.Time
}

// Order is a structure defining the order in the kitchen
type Order struct {
	opts *OrderOptions
	cfg  *Config

	startTS       *time.Time
	shelfSwitchTS time.Time

	deliveryTime time.Duration

	spoilTimer    *time.Timer
	deliveryTimer *time.Timer

	shelfChange chan Pair
	wasted      chan bool

	shelf *Shelf

	inverseValue float64
	wg           sync.WaitGroup

	timersDone chan bool

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

func NewOrder(opts *OrderOptions,
	cfg *Config,
	onSpoil func(ord *Order), onDeliver func(ord *Order)) *Order {
	return &Order{
		opts:      opts,
		cfg:       cfg,
		OnSpoil:   onSpoil,
		OnDeliver: onDeliver,
	}
}

func (ord *Order) Init(shelf *Shelf) {
	ord.inverseValue = 0
	ord.timersDone = make(chan bool)
	ord.shelfChange = make(chan Pair)

	go ord.shelfChangerLoop()
	ord.putOnTheShelf(shelf, time.Now())
}

func (ord *Order) ChangeShelf(shelf *Shelf, ts time.Time) {
	ord.shelfChange <- Pair{
		Shelf: shelf,
		TS:    ts,
	}
}

func (ord *Order) Done() {
	ord.timersStopSync()
	close(ord.shelfChange)
}

func (ord *Order) calculateValue(elapsedSeconds float64) float64 {
	return (float64(ord.opts.ShelfLife) -
		elapsedSeconds -
		(ord.opts.DecayRate *
			elapsedSeconds *
			float64(ord.shelf.ShelfDecayModifier))) /
		float64(ord.opts.ShelfLife)
}

func (ord *Order) putOnTheShelf(shelf *Shelf, eventTS time.Time) {

	// on shelf change
	if ord.startTS != nil {

		currentTime := eventTS
		currentTS := currentTime.UnixNano()

		elapsedSeconds := float64(ord.shelfSwitchTS.UnixNano()-currentTS) / 1000000000

		ord.inverseValue = ord.inverseValue +
			(1-ord.inverseValue)*(1-ord.calculateValue(elapsedSeconds))

		// v = (shelfLife - age - order.decay * age * shelf.decay) / shelfLife
		decayRate := 1 / (float64(ord.opts.ShelfLife) /
			(1 + ord.opts.DecayRate*float64(shelf.ShelfDecayModifier)))
		timeToSpoil := (1 - ord.inverseValue) / decayRate
		timeToDeliver := ord.deliveryTime.Seconds() -
			(float64(currentTS) - float64(ord.startTS.UnixNano()/1000000000))

		ord.shelfSwitchTS = currentTime
		ord.shelf = shelf

		ord.timersStart(time.Duration(timeToDeliver)*
			time.Second, time.Duration(timeToSpoil)*
			time.Second)

		return
	}

	// initalisation
	// this one happens only once at start
	startTS := time.Now()

	ord.deliveryTime = time.Duration(ord.cfg.courierReadyMin+
		rand.Float64()*(ord.cfg.courierReadyMax-ord.cfg.courierReadyMin)) * time.Second
	spoilTime := time.Duration(ord.calculateMaxOrderAge(shelf.ShelfDecayModifier)) *
		time.Second

	ord.shelf = shelf
	ord.startTS = &startTS
	ord.shelfSwitchTS = startTS
	ord.inverseValue = 0

	ord.timersStart(ord.deliveryTime, spoilTime)
}

func (ord *Order) shelfChangerLoop() {
	for pair := range ord.shelfChange {
		ord.timersStopSync()
		ord.putOnTheShelf(pair.Shelf, pair.TS)
	}
}

func (ord *Order) timersStopSync() {
	// to release timers faster in case of
	// wasted state (state in which we throw away random order)
	select {
	case ord.timersDone <- true:
		{

		}
	default:
		{

		}
	}
	ord.wg.Wait()
}

func (ord *Order) timersStart(deliveryTime, spoilTime time.Duration) {
	ord.deliveryTimer = time.NewTimer(deliveryTime)
	ord.spoilTimer = time.NewTimer(spoilTime)
	go ord.timersWait()
}

func (ord *Order) timersWait() {

	// need to wait before running timers again
	ord.wg.Add(1)
	defer ord.wg.Done()

	select {
	case <-ord.deliveryTimer.C:
		{
			fmt.Println("delivered")
			ord.OnDeliver(ord)
		}
	case <-ord.spoilTimer.C:
		{
			fmt.Println("spolied")
			ord.OnSpoil(ord)
		}
	case <-ord.timersDone:
		{
			fmt.Println("timers done")
		}
	}

	// release timers
	if !ord.deliveryTimer.Stop() {
		<-ord.deliveryTimer.C
	}

	if !ord.spoilTimer.Stop() {
		<-ord.spoilTimer.C
	}
}

func (ord *Order) calculateMaxOrderAge(shelfDecayModifier int) float64 {
	return float64(ord.opts.ShelfLife) /
		(1 + ord.opts.DecayRate*float64(shelfDecayModifier))
}
