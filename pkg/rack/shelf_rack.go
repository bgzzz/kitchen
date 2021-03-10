package rack

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/bgzzz/kitchen/pkg/orders"
	ordrs "github.com/bgzzz/kitchen/pkg/orders"
	shvs "github.com/bgzzz/kitchen/pkg/shelves"
	"github.com/bgzzz/kitchen/pkg/stats"
	"github.com/sirupsen/logrus"
)

const (
	OECreated = iota
	OEDelivered
	OESpoiled
)

const (
	orderStateCreated     = "CREATED"
	orderStateDelivered   = "DELIVERED"
	orderStateSpoiled     = "SPOILED"
	orderStateShelfChange = "SHELF_CHANGE"
	orderStateWasted      = "WASTED"
)

// OrderEvent represents the state of the order and is needed
// to interact with shelfrack
type OrderEvent struct {
	EventType int
	Order     *ordrs.Order
}

// ShelfSet represents shelf's properties in addition to
// orders located on this shelf
type ShelfSet struct {
	shelf  *shvs.Shelf
	orders map[string]*ordrs.Order
}

// ShelfRack represents the set of shelves capable of processing
// orders
type ShelfRack struct {
	log                    *logrus.Entry
	eventCh                chan OrderEvent
	stats                  *stats.Stats
	rack                   map[string]ShelfSet
	shelfList              []string
	expectedOrdrsToProcess int
	onFinish               func()
}

// NewShelfRack creates shelf rack structure
// representing the rack of shelf processing the orders
func NewShelfRack(log *logrus.Entry, stats *stats.Stats, shelves []*shvs.Shelf,
	expectedToProcess int, onFinish func()) *ShelfRack {

	sr := &ShelfRack{
		log:                    log,
		eventCh:                make(chan OrderEvent),
		rack:                   make(map[string]ShelfSet),
		expectedOrdrsToProcess: expectedToProcess,
		onFinish:               onFinish,
		stats:                  stats,
	}

	for _, shelf := range shelves {
		sr.rack[shelf.Temp] = ShelfSet{
			shelf:  shelf,
			orders: make(map[string]*ordrs.Order),
		}
		sr.shelfList = append(sr.shelfList, shelf.Temp)
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
func (sr *ShelfRack) removeOrder(order *ordrs.Order, state string) {
	_, isTempShelf := sr.rack[order.Shelf.Temp].orders[order.Opts.ID]
	_, isOverflowShelf := sr.rack[shvs.OverflowShelfTemp].orders[order.Opts.ID]
	if isOverflowShelf || isTempShelf {
		delete(sr.rack[order.Shelf.Temp].orders, order.Opts.ID)
		delete(sr.rack[shvs.OverflowShelfTemp].orders, order.Opts.ID)
		ordrValue := order.CurrentValue(time.Now())
		sr.PrintState(order.Opts.ID, state, ordrValue)
		order.Done()
		sr.expectedOrdrsToProcess--
		if state == orderStateDelivered {
			sr.stats.Delivered(ordrValue)
			return
		}

		if state == orderStateSpoiled {
			sr.stats.Spoiled()
			return
		}

	}
}

// eventLoop is processing loop of shelf rack interaction events
func (sr *ShelfRack) eventLoop() {

	for oe := range sr.eventCh {

		switch oe.EventType {
		case OECreated:
			{
				sr.findShelf(oe.Order)
			}
		case OEDelivered:
			{
				sr.removeOrder(oe.Order, orderStateDelivered)
			}
		case OESpoiled:
			{

				sr.removeOrder(oe.Order, orderStateSpoiled)
			}
		default:
			{
				sr.log.Error("unsupported event supplied")
			}
		}
		if sr.expectedOrdrsToProcess == 0 {
			sr.log.Info(sr.stats.String())
			sr.onFinish()
		}
	}
}

// ShelfChangeSet support structure representing par of order
// and shelf where this order suppose to be put
type ShelfChangeSet struct {
	order *ordrs.Order
	shelf *shvs.Shelf
}

// findShelf represents the main logic of processing the order
// via shelf rack
func (sr *ShelfRack) findShelf(order *ordrs.Order) {
	// this is the place to implement different dispatching algorithms
	// currently it is the on that works according to the simplified
	// dispatching algorithm proposed in the task
	ordersToChange, ordersToWaste, shelf := sr.resolveRackState(order)
	order.Init(shelf)
	sr.PrintState(order.Opts.ID, orderStateCreated,
		order.CurrentValue(time.Now()))

	for _, change := range ordersToChange {
		change.order.ChangeShelf(change.shelf)
		sr.PrintState(order.Opts.ID, orderStateShelfChange,
			order.CurrentValue(time.Now()))
	}

	for _, ord := range ordersToWaste {
		ordrValue := order.CurrentValue(time.Now())
		sr.PrintState(order.Opts.ID, orderStateWasted,
			ordrValue)
		ord.Done()
		sr.stats.Wasted(ordrValue)

		sr.expectedOrdrsToProcess--
	}
}

// PrintState prints the state via the info message of the rc logger
func (sr *ShelfRack) PrintState(orderID, state string, value float64) {
	sr.log.Infof("\nOrder state\n\tID: %s\n\tState: %s\n\tValue: %f",
		orderID, state, value)
	sr.log.Info(sr.shelvesContent())
	sr.log.Info("-------------------------------------------------------")
}

// putOrdOnTheShelf return true if it successfully put order on the
// shelf. Shelf's capacity is taken into account
func (sr *ShelfRack) putOrdOnTheShelf(ord *ordrs.Order, shelfTemp string) bool {
	if sr.rack[shelfTemp].shelf.Capacity >=
		(len(sr.rack[shelfTemp].orders) + 1) {
		sr.rack[shelfTemp].orders[ord.Opts.ID] = ord
		return true
	}
	return false
}

// resolveRackState sets the newly created order to the appropriate
// shelf. It returns:
// - list of orders that are needed to change the shelf
// - list of orders that are considered wasted
// - shelf of the newly created order
func (sr *ShelfRack) resolveRackState(order *ordrs.Order) (ordsToChange []*ShelfChangeSet,
	ordsToWaste []*ordrs.Order, shelf *shvs.Shelf) {
	// trying to set order on the optimal shelf
	if sr.putOrdOnTheShelf(order, order.Opts.Temp) {
		shelf = sr.rack[order.Opts.Temp].shelf
		return
	}

	// trying to set order on the overflow
	if sr.putOrdOnTheShelf(order, shvs.OverflowShelfTemp) {
		shelf = sr.rack[shvs.OverflowShelfTemp].shelf
		return
	}

	// trying to free space on overflow
	for _, ord := range sr.rack[shvs.OverflowShelfTemp].orders {
		if sr.putOrdOnTheShelf(ord, ord.Opts.Temp) {
			ordsToChange = append(ordsToChange, &ShelfChangeSet{
				order: ord,
				shelf: sr.rack[ord.Opts.Temp].shelf,
			})
			sr.log.Infof("SHELF_CHANGE %v", ord.Opts.ID)
			delete(sr.rack[shvs.OverflowShelfTemp].orders, ord.Opts.ID)
			if !sr.putOrdOnTheShelf(order, shvs.OverflowShelfTemp) {
				panic("this should no happen")
			}
			shelf = sr.rack[shvs.OverflowShelfTemp].shelf
			return
		}
	}

	// put random order from overflow to waste
	wastedIndex := rand.Intn(len(sr.rack[shvs.OverflowShelfTemp].orders))

	i := 0
	for _, ord := range sr.rack[shvs.OverflowShelfTemp].orders {
		if i == wastedIndex {
			delete(sr.rack[shvs.OverflowShelfTemp].orders, ord.Opts.ID)
			ordsToWaste = append(ordsToWaste, ord)
			sr.rack[shvs.OverflowShelfTemp].orders[order.Opts.ID] = order
			shelf = sr.rack[shvs.OverflowShelfTemp].shelf
			return
		}
		i++
	}

	return
}

// shelvesContent return the prepared output string showing
// content of the shelves
func (sr *ShelfRack) shelvesContent() string {
	output := "\nRack state:\n"
	for _, shelfTemp := range sr.shelfList {
		output = fmt.Sprintf("%sShelf %s %d/%d:\n%s",
			output, sr.rack[shelfTemp].shelf.Temp,
			len(sr.rack[shelfTemp].orders),
			sr.rack[shelfTemp].shelf.Capacity,
			mapToString(sr.rack[shelfTemp].orders))
	}
	return output
}

// mapToString returns sorted list of orders on the shelf
func mapToString(m map[string]*orders.Order) string {
	keys := []string{}
	for k := range m {
		keys = append(keys, fmt.Sprintf("\t%s\n", k))
	}

	sort.Strings(keys)

	return strings.Join(keys, "")
}
