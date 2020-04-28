# Tools to manipulate with structs

Installation: `go get -v github.com/reddec/struct-view/cmd/...`

* [Events generator](#events-generator)
* [Cache generator](#cache-generator)
* [Timed cache](#timed-cache)
* [Binary encoding](#binary-gen)
* [JSON array enum generator](#json-enum-array-gen)
* [Ring buffer generator](#ring-buffer-generator)
* struct-view

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
  -e, --emitter=     Create emitter factory [$EMITTER]
  -l, --listener=    Create method to subscribe for all events (default: SubscribeAll) [$LISTENER]
  -H, --hint=        Give a hint about events (eventName -> struct name) [$HINT]
  -c, --context      Add context to events [$CONTEXT]

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

You may use option `ref` (like `event:"EventName,ref"`) to use payload by reference.

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

### Multiple packages as a source

Sometimes you need to create events for types that you should not modify (add comments) or for types from different
packages. 

`events-gen` supports as positional arguments multiple source directories that needs to be scanned. Also it is possible
to give generator a "hint": expected event name and type. In this case generator will create events object for types that
matches hints as well as marked by comments.

As an addition to the example above let's imagine other package called `transactions` located in `../transactions` directory
with types `UserTX` and `BankTX` that we want to use as our events `UserTxCreated` and `BankTxCreated`. So we need to modify
generator command (used example from basic to reduce number flags, however, you are not restricted in that) as: 

`events-gen -p basic -o events.go -H UserTxCreate:UserTX -H BankTxCreated:BankTX . ../transactions` 

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

### Emitter

It might be useful to use emitter in an external code or already existent code, however, use instance of a `EventBus`
could be not an ideal decision due to increase of code coupling.

For that reason you may create additional emitter by flag `-e <EmitterFunc>` that will generate additional method `EmitterFunc`
in a `EventBus` and additional structure that aggregates all `Emit()` methods in one place. By using this approach you
may require an interface in your code instead of exact implementation.

So for the basic example, described above, `-e Emitter` the implementation of this interface will be generated 
(note: interface by itself will not be generated due to a best-practice "accept interface, return structure"):


```go
type Sample interface {
    UserCreated(payload User)
    UserRemoved(payload User)
    UserSubscribed(payload Subscription)
    UserLeaved(payload Subscription)
}
```


### Listener

To subscribe on all events exists method `SubscribeAll`, however, name of the method could be overloaded by 
`-l <listener method>` flag. If method name is empty, method will not be generated.

### Context

To add `context` argument for all events, add flag `-c` 

## Cache generator

Generates multi-level cache for key-value data with a separate synchronization unit per value.

```
Usage:
  cache-gen [OPTIONS]

Application Options:
  -p, --package=      Package name (can be override by output dir) (default: cache) [$PACKAGE]
  -o, --output=       Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -k, --key-type=     Key type [$KEY_TYPE]
  -v, --value-type=   Value type [$VALUE_TYPE]
  -i, --key-import=   Import for key type [$KEY_IMPORT]
  -I, --value-import= Import for value type [$VALUE_IMPORT]
  -t, --type-name=    Typename for cache (default: Manager) [$TYPE_NAME]

Help Options:
  -h, --help          Show this help message
```

**Common use-case**: you have a service that contains user profiles (identified by ID) and you want to cache results
to prevent multiple requests for the same user. However, you expect that your code could be used from multiple
threads/gorountines so parallel requests to different users should not block each-other, but code should make only
one request per unique user id.

Example:

```
cache-gen -p users -k int64 -v *UserProfile -I github.com/MyCompany/MyTypes -o user_cached.go
```

* `-p users` sets package name to `users`
* `-k int64` sets key type (user id) to int64
* `-v *UserProfile` sets value type (user profile) as a ref to user-defined type
* `-I github.com/MyCompany/MyType`  sets import for user-defined package for value type (`UserProfile`)
* `-o user_cached.go` sets output to file named `user_cached.go`

Type name (`-t`) not set so default name will be used (`Manager`).

Result (functions body omitted):


```go
package cache

import (
	"context"
	mytypes "github.com/MyCompany/MyTypes"
	"sync"
)

type UpdaterManager interface {
	Update(ctx context.Context, key int64) (*mytypes.UserProfile, error)
}
type UpdaterManagerFunc func(ctx context.Context, key int64) (*mytypes.UserProfile, error)

func (fn UpdaterManagerFunc) Update(ctx context.Context, key int64) (*mytypes.UserProfile, error) {} // body omitted

func NewManagerFunc(updateFunc UpdaterManagerFunc) *Manager {} // body omitted

func NewManager(updater UpdaterManager) *Manager {} // body omitted

type Manager struct {
    // fields omitted
}

func (mgr *Manager) Find(key int64) *cacheManager {}
func (mgr *Manager) FindOrCreate(key int64) *cacheManager {}
func (mgr *Manager) Get(ctx context.Context, key int64) (*mytypes.UserProfile, error) {}
func (mgr *Manager) Set(key int64, value *mytypes.UserProfile) {}
func (mgr *Manager) Purge(key int64) {}
func (mgr *Manager) PurgeAll() {}
func (mgr *Manager) Snapshot() map[int64]*mytypes.UserProfile {}

type cacheManager struct {
    // fields omitted    
}

func (cache *cacheManager) Valid() bool {}
func (cache *cacheManager) Invalidate() {}
func (cache *cacheManager) Key() int64 {}
func (cache *cacheManager) Get() *mytypes.UserProfile {}
func (cache *cacheManager) Ensure(ctx context.Context) (*mytypes.UserProfile, error) {}
func (cache *cacheManager) Set(value *mytypes.UserProfile) {}
func (cache *cacheManager) Update(ctx context.Context, force bool) error {}
```

See full example in `examples/cache`


## Timed cache

Generate simple cache with expiration time

```
Usage:
  timed-cache [OPTIONS]

Application Options:
  -p, --package=      Package name (can be override by output dir) (default: cache) [$PACKAGE]
  -o, --output=       Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -v, --value-type=   Value type [$VALUE_TYPE]
  -I, --value-import= Import for value type [$VALUE_IMPORT]
  -t, --type-name=    Typename for cache (default: Manager) [$TYPE_NAME]
  -a, --array         Is value should be an array [$ARRAY]

Help Options:
  -h, --help          Show this help message

```

## Binary gen

Generate very simple static binary marshal/unmarshal for struct. Unknown fields are ignored. Goal is support same
encoding/decoding with C/C++ structures with same encoding layout (aka: result should be decodable on Big endian machines
like `(struct *my_type)(buffer)`).

Currently supported types:

* `uint8`, `uint16`, `uint32`, `uint64`

```
Usage:
  binary-gen [OPTIONS]

Application Options:
  -o, --output=    Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -t, --type-name= TypeName for generator (default: Manager) [$TYPE_NAME]

Help Options:
  -h, --help       Show this help message

```

see [examples/binarygen](examples/binarygen) directory

## JSON-enum array gen

Generates type alias, validator for values (according to entered values) and custom JSON Unmarshal with embedded checks.


```
Usage:
  json-enum-gen [OPTIONS] [Values...]

Application Options:
  -p, --package=     Package name (can be override by output dir) (default: enum) [$PACKAGE]
  -o, --output=      Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -t, --type-name=   Enum name (default: Manager) [$TYPE_NAME]
  -s, --source-type= Source type name (default: string) [$SOURCE_TYPE]

Help Options:
  -h, --help         Show this help message

```


For example you need to check incoming request to play some channel (ex: podcast). Request message will look like

```json
{
  "channel" : "channel name",
  "speed": 1 
}
```

You want to allow your customers choose only specific values for speed: 0.5, 1, 1.5, 2, 2.5

Common solution is to create enum, defined possible values, generate JSON wrapper. But **it will bring mess to your code**
with meaningless values like `Channel0.5 = 0.5; Channel1 = 1` and so on.

This command will let you keep clean code as much as possible:

```
json-enum-gen -s int -o speed.go -t Speed 0.5 1 1.5 2 2.5
``` 

will generate code:


```go
// Code generated by json-enum-gen. DO NOT EDIT.
//go:generate json-enum-gen -s int -t Speed 0.5 1 1.5 2 2.5
package enum

import (
        "encoding/json"
        "errors"
)

type Speed int

func (v Speed) Get() int {
        return int(v)
}
func (v Speed) IsValid() bool {
        switch int(v) {
        case 0.5, 1, 1.5, 2, 2.5:
                return true
        default:
                return false
        }
}
func (v *Speed) UnmarshalJSON(data []byte) error {
        var parsed int
        err := json.Unmarshal(data, &parsed)
        if err != nil {
                return err
        }
        typed := Speed(parsed)
        if !typed.IsValid() {
                return errors.New("Invalid value for type Speed. Possible options are: 0.5, 1, 1.5, 2, 2.5")
        }
        *v = typed
        return nil
}

```

## Sync map gen

Generates Java-like thread-safe map

Generated code will implement next interface:

```go
package sample

type UpdaterFunc func(key KeyType) (ValueType, error)

type SyncItem interface {
    Valid() bool
    Invalidate() 
    Key() KeyType
    Get() ValueType
    Set(value ValueType)
    Ensure(updater UpdaterFunc) (ValueType, error)
}

type SyncMap interface{
    Find(key KeyType) SyncItem
    FindOrCreate(key KeyType) SyncItem
    Get(key KeyType, construct UpdaterFunc) (ValueType, error)
    Set(key KeyType, value ValueType)
    Purge(key KeyType)
    PurgeAll()
    Snapshot() map[KeyType]ValueType
} 
```

Usage

```
Usage:
  syncmap-gen [OPTIONS]

Application Options:
  -p, --package=      Package name (can be override by output dir) (default: cache) [$PACKAGE]
  -o, --output=       Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -k, --key-type=     Key type [$KEY_TYPE]
  -v, --value-type=   Value type [$VALUE_TYPE]
  -i, --key-import=   Import for key type [$KEY_IMPORT]
  -I, --value-import= Import for value type [$VALUE_IMPORT]
  -t, --type-name=    TypeName for cache (default: Manager) [$TYPE_NAME]

Help Options:
  -h, --help          Show this help message
```

### Params gen


Scans struct methods and generate wrappers for all parameters for all exported methods

Example:

```go

type App struct {}

func (app *App) Sum(a, b, c int) int {return a + b +c}
```

invoke

`params-gen -t App -o params.go`

will generate


```go
type SumParams struct {
	A int `form:"a" json:"a" xml:"a" yaml:"a"`
	B int `form:"b" json:"b" xml:"b" yaml:"b"`
	C int `form:"c" json:"c" xml:"c" yaml:"c"`
}

func (sp *SumParams) Invoke(app *App) int {
	return app.Sum(sp.A, sp.B, sp.C)
}
```



Usage

```
Usage:
  params-gen [OPTIONS]

Application Options:
  -p, --package=   Package name (can be override by output dir) [$PACKAGE]
  -o, --output=    Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
      --dir=       Directory to scan (default: .) [$DIR]
  -t, --type-name= TypeName for cache (default: Manager) [$TYPE_NAME]

Help Options:
  -h, --help       Show this help message

```


## Ring buffer generator

Classical fixed-buffer circle (ring) container, where new data overwrites old one. [Wikipedia](https://en.wikipedia.org/wiki/Circular_buffer).

Complexity

|          |       |
|----------|-------|
| Add      |  O(1) |
| Get      |  O(1) |
| Remove   |  N/A  |
| Copy     |  O(N) |

**Example:**

Generate ring buffer for `int` type
 
`ring-buffer-gen -p abc -t int --name IntBuffer`

will produce (implementation details omitted):

```go
package abc

// New instance of ring buffer
func NewIntBuffer(size uint) *IntBuffer {}

// Wrap pre-allocated buffer to ring buffer
func WrapIntBuffer(buffer []int) *IntBuffer {}

// Ring buffer for type int
type IntBuffer struct {
	seq  uint64
	data []int
}

// Add new element to the ring buffer. If buffer is full, the oldest element will be overwritten
func (rb *IntBuffer) Add(value int) {}

// Get element by index. Negative index is counting from end
func (rb *IntBuffer) Get(index int) (ans int) {}

// Length of used elements. Always in range between zero and maximum capacity
func (rb *IntBuffer) Len() int {}

// Clone of ring buffer with shallow copy of underlying buffer
func (rb *IntBuffer) Clone() *IntBuffer {}

// Flatten copy of underlying buffer. Data is always ordered in an insertion order
func (rb *IntBuffer) Flatten() []int {}

```

**Usage**

```
Usage:
  ring-buffer-gen [OPTIONS]

Application Options:
  -p, --package=      Package name (can be override by output dir) (default: enum) [$PACKAGE]
  -o, --output=       Generated output destination (- means STDOUT) (default: -) [$OUTPUT]
  -t, --type-name=    Type name to wrap [$TYPE_NAME]
      --name=         Result structure name (default: RingBuffer) [$NAME]
      --synchronized  Make collection be synchronized [$SYNCHRONIZED]
  -i, --import=       Import for type [$IMPORT]

Help Options:
  -h, --help          Show this help message
```