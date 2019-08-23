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

Feel free to look in **examples/basic** directory to see generated file.

Finally you can embed event emitter to you struct like this: 
```go
package basic

type UserApp struct {
    Subscribed UserSubscribed
    Leaved     UserLeaved
    //...
}
```


### Advanced usage

Quite often we need to have some aggregated event source (aka event bus) that aggregates several event emitters in one 
place.

By using example above we may imagine situation that all our events actually relates to the one application. Let's call
it `App`.

We also what to let other developers easily extend logic of our product by adding event listeners and (probably) let them
also emits events using only one source of truth.

In terms of `events-gen` this pattern called `EventBus`.

Let's improve our previous application and change generator invocation to 
`events-gen --event-bus Events -P -p advance -o events.go .`. It means generate event bus (events aggregator) 
called `Events` and make all generated definitions private (`-P`) except bus itself. We also changed package to `advance`
so you may look into **examples/advance**.

> To mark generated structures as a private (`-P`) is completely optional, however, it's common case that if we already aggregated
> our events to the one structure then we probably don't want to expose events objects to outer world except only through event bus.

Generated event bus will looks something like

```go
type Events struct {
    UserCreated    eventUserCreated
    UserRemoved    eventUserRemoved
    UserSubscribed eventUserSubscribed
    UserLeaved     eventUserLeaved
}
``` 

Finally you can embed event bus to you struct like this: 
```go
package advance

type App struct {
    Events Events
    //...
}
```

### Mirroring and integration

Event-based approach for complex systems most of the time means integration with other external, legacy or just other
components. Common case for enterprise solution is to use integration bus (IBM MQ, RabbitMQ, ....) as a transport for events.

For such cases you may use mirroring (`-m`) as well as global sink (`-m`). Both ways let you consume events in unified 
way without caring about types.

Those approaches very similar to each other, however, mirroring (`-m`) is a bit faster but supports only one sink and 
global sink (`-s`) that just subscribes to all events and has no limits for the number of listeners.

So for the example above generator will create:

**mirroring** (`-m`)

```go
func EventsWithMirror(mirror func(eventName string, payload interface{})) *Events 
```

**global sink** (`-s`)

```go
func (bus *Events) Sink(sink func(eventName string, payload interface{})) *Events
```

Function parameters are self-explainable, but:

* `eventName` - name of event (`UserCreated`, `UserRemoved`,...)
* `payload` - original event object (not reference, a value)


#### From mirror

All described above were about consuming events made by our generated events bus. However, you may want also
transparently integrate external system used as source of events and propagate them to the local instance. For example,
you may want use notification from a message broker (RabbitMQ, IBM MQ, HTTP REST,...) as internal events.

```
+----------------+                +-----------+  <--- emit  --- +-----------+
| exteral system | --- event ---> | event bus |                 | component |
+----------------+    (as emit)   +-----------+  --- listen --> +-----------+
```   

In terms of `events-generator` such approach called `FromMirror` and it's available ony together with `EventBus`.

Generated code could be a bit tricky, however, to generate `FromMirror` handlers just add `-f` flag to the generator.
 It will produce (for example above) methods for events:

**universal emitter**

`func (ev *Events) Emit(eventName string, payload interface{})`

Emits event by name. Payload should event type (reference or value). Silently drops invalid parameters:
unknown event, incorrect payload.

**universal payload fabric**

`func (ev *Events) Payload(eventName string) interface{}`

Creates payload value (reference) by event name or returns nil.


Both of this methods require case-sensitive event name, however, by flag `-i` it can be switched to case-insensitive mode.