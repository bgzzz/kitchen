package main

import (
	"fmt"
	"math/rand"

	"github.com/sirupsen/logrus"
)

const (
	OECreated = iota
	OEDelivered
	OESpoiled
)

const (
	overflowShelfName = "overflow"
)

// OrderEvent represents the state of the order and is needed
// to interact with shelfrack
type OrderEvent struct {
	eventType int
	order     *Order
}

// ShelfSet represents shelf's properties in addition to
// orders located on this shelf
type ShelfSet struct {
	shelf  *Shelf
	orders map[string]*Order
}

// ShelfRack represents the set of shelves capable of processing
// orders
type ShelfRack struct {
	log     *logrus.Entry
	eventCh chan OrderEvent

	rack                   map[string]ShelfSet
	expectedOrdrsToProcess int
	onFinish               func()
}

// NewShelfRack creates shelf rack structure
// representing the rack of shelf processing the orders
func NewShelfRack(log *logrus.Entry, shelves []*Shelf,
	expectedToProcess int, onFinish func()) *ShelfRack {
	sr := &ShelfRack{
		log:                    log,
		eventCh:                make(chan OrderEvent),
		rack:                   make(map[string]ShelfSet),
		expectedOrdrsToProcess: expectedToProcess,
		onFinish:               onFinish,
	}

	for _, shelf := range shelves {
		sr.rack[shelf.Temp] = ShelfSet{
			shelf:  shelf,
			orders: make(map[string]*Order),
		}
	}

	return sr
}

// Init start event loop for processing the interaction with
// shelf rack
func (sr *ShelfRack) Init() {
	go sr.eventLoop()
}

// Interact allows to interact with the shelf rack by sending
// order events
func (sr *ShelfRack) Interact(event *OrderEvent) {
	sr.eventCh <- *event
}

// removeOrder removes order from the shelf
func (sr *ShelfRack) removeOrder(order *Order) {
	delete(sr.rack[order.shelf.Temp].orders, order.opts.ID)
}

// eventLoop is processing loop of shelf rack interaction events
func (sr *ShelfRack) eventLoop() {
	for oe := range sr.eventCh {
		switch oe.eventType {
		case OECreated:
			{
				sr.findShelf(oe.order)
			}
		case OEDelivered:
			{
				sr.removeOrder(oe.order)
				sr.expectedOrdrsToProcess--
			}
		case OESpoiled:
			{
				sr.removeOrder(oe.order)
				sr.expectedOrdrsToProcess--
			}
		default:
			{
				fmt.Println("wrong event supplied")
			}
		}
		if sr.expectedOrdrsToProcess == 0 {
			sr.onFinish()
		}
	}
}

// ShelfChangeSet support structure representing par of order
// and shelf where this order suppose to be put
type ShelfChangeSet struct {
	order *Order
	shelf *Shelf
}

// findShelf represents the main logic of processing the order
// via shelf rack
func (sr *ShelfRack) findShelf(order *Order) {
	ordersToChange, ordersToWaste, shelf := sr.resolveRackState(order)
	fmt.Println(ordersToWaste)
	fmt.Println(shelf)
	order.Init(shelf)
	fmt.Println("added " + order.opts.ID)

	for _, change := range ordersToChange {
		change.order.ChangeShelf(change.shelf)
	}

	for _, ord := range ordersToWaste {
		fmt.Println("wasted " + ord.opts.ID)
		ord.Done()
		sr.expectedOrdrsToProcess--
	}
}

// putOrdOnTheShelf return true if it successfully put order on the
// shelf. Shelf's capacity is taken into account
func (sr *ShelfRack) putOrdOnTheShelf(ord *Order, shelfTemp string) bool {
	if sr.rack[shelfTemp].shelf.Capacity >=
		(len(sr.rack[shelfTemp].orders) + 1) {
		sr.rack[shelfTemp].orders[ord.opts.ID] = ord
		return true
	}
	return false
}

// resolveRackState sets the newly created order to the appropriate
// shelf. It returns:
// - list of orders that are needed to change the shelf
// - list of orders that are considered wasted
// - shelf of the newly created order
func (sr *ShelfRack) resolveRackState(order *Order) (ordsToChange []*ShelfChangeSet,
	ordsToWaste []*Order, shelf *Shelf) {
	// trying to set order on the optimal shelf
	if sr.putOrdOnTheShelf(order, order.opts.Temp) {
		shelf = sr.rack[order.opts.Temp].shelf
		return
	}

	// trying to set order on the overflow
	if sr.putOrdOnTheShelf(order, overflowShelfName) {
		shelf = sr.rack[overflowShelfName].shelf
		return
	}

	// trying to free space on overflow
	for _, ord := range sr.rack[overflowShelfName].orders {
		if sr.putOrdOnTheShelf(ord, ord.opts.Temp) {
			ordsToChange = append(ordsToChange, &ShelfChangeSet{
				order: ord,
				shelf: sr.rack[ord.opts.Temp].shelf,
			})
			shelf = sr.rack[overflowShelfName].shelf
			return
		}
	}

	// put random order from overflow to waste
	wastedIndex := rand.Intn(len(sr.rack[overflowShelfName].orders))

	i := 0
	for _, ord := range sr.rack[overflowShelfName].orders {
		if i == wastedIndex {
			delete(sr.rack[overflowShelfName].orders, ord.opts.ID)
			ordsToWaste = append(ordsToWaste, ord)
			sr.rack[overflowShelfName].orders[order.opts.ID] = order
			shelf = sr.rack[overflowShelfName].shelf
			return
		}
		i++
	}

	return
}
