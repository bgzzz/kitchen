# kitchen
Cloud kitchen simulation

## Description 
Utility that simulates processing orders via rack of shelves.
It consumes simulation configuration including list of orders and list of shelves in the rack and outputs realtime order processing information.

## How to build 
```
make build 
```

## How to run tests 
```
make test
```

## Run all make directives (build, test lint)
*NOTE: be sure that you have golangci-lint installed and accessible via PATH*
```
make all
```

## How to run application
```
./bin/kitchen --simulation-config ./kitchen.yaml
```
Program requires following files to be present:
- orders.json - list of orders for simulation 
- kitchen.yaml - simulation description 
- shelves.json - list of shelves that is going to be used in the simulation

## Docker build
In case of absence of developer infrastructure you can build docker image and use it as a cli command with mounting configuration files into tmp folder 
```
docker build -t kitchen .
docker run --rm  -v $(pwd):/tmp  kitchen ./tmp/kitchen.yaml
```

## Help example

```
./bin/kitchen
NAME:
   kitchen - A new cli application

USAGE:
   kitchen [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --simulation-config value  Path to file containing simulation config. ex: ./kitchen.yaml (default: ./kitchen.yaml) [$KITCHEN_SIMULATION_CONFIG_PATH]
   --debug                    Debug logging (default: false) [$KITCHEN_SIMULATION_DEBUG]
   --help, -h                 show help (default: false)
```

## Architecture decisions

1. Order live-cycle is made via golang timers (spoiling and delivery timer) to support the real-time simulation.
1. Time to spoil re-calculated each time we switch the shelf where order is located 
1. Shelf change event is handled via shelf change event loop of the order object
1. The main kitchen processing unit is rack of shelves (shelf_rack.go)
1. Shelf_rack has its own eventloop for interaction with shelfrack. All events are consumed sequentially. Sequential processing is done because we have to evaluate the state of the whole rack while we do the scheduling decision. This primarily is done to support more sophisticated scheduling algorithms.
1. Scheduling algorithm is done according to the rules described in the task.
1. The extension could be the scheduler that evaluates system performance of the rack in general ( ex: maximize weighted average of order values by shuffling orders on the rack shelves) 

## TODO 

1. More Unit tests
1. More sophisticated scheduling algorithms

