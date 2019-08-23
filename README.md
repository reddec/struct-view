# Tools to manipulate with structs

Installation: `go get -v github.com/reddec/struct-view/cmd/...`

## Events generator


```
Usage:
  events-gen [OPTIONS] [Directories...]

Application Options:
  -p, --package=     Package name (can be override by output dir) (default: events) [$PACKAGE]
  -o, --output=      Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -P, --private      Make generated event structures be private by prefix 'event' [$PRIVATE]
      --event-bus=   Generate structure that aggregates all events [$EVENT_BUS]
  -m, --mirror       Mirror all events to the universal emitter [$MIRROR]
  -f, --from-mirror  Create producer events as from mirror (only for event bus) [$FROM_MIRROR]
  -i, --ignore-case  Ignore event case for universal source (--from-mirror) [$IGNORE_CASE]
  -s, --sink         Make a sink method for event bus to subscribe to all events [$SINK]
  -H, --hint=        Give a hint about events (eventName -> struct name) [$HINT]

Help Options:
  -h, --help         Show this help message
```

### Basic usage


You should declare go file with types that will be used as event types. For example:

```go
package basic
//go:generate events-gen -p basic -o events.go .

// event:"UserCreated"
// event:"UserRemoved"
type User struct {
	ID   int64
	Name string
}

// event:"UserSubscribed"
// event:"UserLeaved"
type Subscription struct {
	UserID int64
	Mail   string
}
```

Magic comment `event:""` gives an instruction to the event generator hint that this type is used as parameter for event
with name defined in commas (`UserCreated`, `UserRemoved` for type `User`).

Instruction for go generator `events-gen -p basic -o events.go .` tells us to generate events to file `events.go` with
package `basic` and look for source files in current (`.`) directory. 

Feel free to look in **examples** directory to see generated file.

Finally you can embed event emitter to you struct like this: 
```go
package basic

type UserAp struct {
    Subscribed UserSubscribed
    Leaved     UserLeaved
    //...
}
```