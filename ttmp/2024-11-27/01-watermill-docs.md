=== BEGIN: docs/articles/index.txt ===
On this page

Articles

   On this page

   You can find more in-depth tips on Watermill in these articles:
     * Distributed Transactions in Go: Read Before You Try
     * Live website updates with Go, SSE, and htmx
     * Using MySQL as a Pub/Sub
     * Creating local Go dev environment with Docker and live code
       reloading

   Help us improve this page
   Prev
   Troubleshooting
   Next
   Awesome Watermill

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/articles/index.txt ===

=== BEGIN: docs/awesome/index.txt ===
On this page

     * Examples
     * Pub/Subs
     * Logging
     * Observability
     * Other

Awesome Watermill

   On this page
     * Examples
     * Pub/Subs
     * Logging
     * Observability
     * Other

   Below is a list of libraries that are not maintained by Three Dots
   Labs, but you may find them useful.

   Please note we can’t provide support or guarantee they work correctly.
   Do your own research.

   If you know another library or are an author of one, please add it to
   the list .

Examples#

     * https://github.com/minghsu0107/golang-taipei-watermill-example
     * https://github.com/minghsu0107/Kafka-PubSub
     * https://github.com/pperaltaisern/go-example-financing

Pub/Subs#

     * AMQP 1.0 https://github.com/kahowell/watermill-amqp10
     * Apache Pulsar https://github.com/AlexCuse/watermill-pulsar
     * Apache RocketMQ https://github.com/yflau/watermill-rocketmq
     * CockroachDB https://github.com/cockroachdb/watermill-crdb
     * Ensign https://github.com/rotationalio/watermill-ensign
     * GoogleCloud Pub/Sub HTTP Push
       https://github.com/dentech-floss/watermill-googlecloud-http
     * MongoDB https://github.com/cunyat/watermill-mongodb
     * MQTT https://github.com/perfect13/watermill-mqtt
     * NSQ https://github.com/chennqqi/watermill-nsq
     * Redis Zset https://github.com/stong1994/watermill-rediszset
     * SQLite https://github.com/davidroman0O/watermill-comfymill

   If you want to find out how to implement your own Pub/Sub adapter,
   check out Implementing custom Pub/Sub .

Logging#

     * logrus
          + https://github.com/ma-hartma/watermill-logrus-adapter
          + https://github.com/UNIwise/walrus
     * logur https://github.com/logur/integration-watermill
     * zap
          + https://github.com/garsue/watermillzap
          + https://github.com/pperaltaisern/watermillzap
     * zerolog
          + https://github.com/alexdrl/zerowater
          + https://github.com/bogatyr285/watermillzlog
          + https://github.com/vsvp21/zerolog-watermill-adapter

Observability#

     * OpenCensus
          + https://github.com/czeslavo/watermill-opencensus
          + https://github.com/sagikazarmark/ocwatermill
     * OpenTelemetry
          + https://github.com/voi-oss/watermill-opentelemetry
          + https://github.com/dentech-floss/watermill-opentelemetry-go-ex
            tra
          + AMQP https://github.com/hpcslag/otel-watermill-amqp
          + GoChannel
            https://github.com/hpcslag/watermill-otel-tracable-gochannel

Other#

     * https://github.com/asyncapi/go-watermill-template
     * https://github.com/goph/watermillx
     * https://github.com/voi-oss/protoc-gen-event

   Help us improve this page
   Prev
   Articles

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/awesome/index.txt ===

=== BEGIN: docs/cqrs/index.txt ===
On this page

     * CQRS
          + Building blocks
     * Usage
          + Example domain
          + Sending a command
          + Command handler
          + Event handler
          + Event Handler groups
          + Generic handlers
          + Building a read model with the event handler
          + Wiring it up
          + What’s next?

CQRS Component

   On this page
     * CQRS
          + Building blocks
     * Usage
          + Example domain
          + Sending a command
          + Command handler
          + Event handler
          + Event Handler groups
          + Generic handlers
          + Building a read model with the event handler
          + Wiring it up
          + What’s next?

CQRS#

     CQRS means “Command-query responsibility segregation”. We segregate
     the responsibility between commands (write requests) and queries
     (read requests). The write requests and the read requests are
     handled by different objects.

     That’s it. We can further split up the data storage, having separate
     read and write stores. Once that happens, there may be many read
     stores, optimized for handling different types of queries or
     spanning many bounded contexts. Though separate read/write stores
     are often discussed in relation with CQRS, this is not CQRS itself.
     CQRS is just the first split of commands and queries.

     Source: www.cqrs.nu FAQ

   CQRS Schema

   The cqrs component provides some useful abstractions built on top of
   Pub/Sub and Router that help to implement the CQRS pattern.

   You don’t need to implement the entire CQRS. It’s very common to use
   just the event part of this component to build event-driven
   applications.

Building blocks#

Event#

   The event represents something that already took place. Events are
   immutable.

Event Bus#

// ...
// EventBus transports events to event handlers.
type EventBus struct {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_bus.go
// ...
type EventBusConfig struct {
        // GeneratePublishTopic is used to generate topic name for publishing ev
ent.
        GeneratePublishTopic GenerateEventPublishTopicFn

        // OnPublish is called before sending the event.
        // The *message.Message can be modified.
        //
        // This option is not required.
        OnPublish OnEventSendFn

        // Marshaler is used to marshal and unmarshal events.
        // It is required.
        Marshaler CommandEventMarshaler

        // Logger instance used to log.
        // If not provided, watermill.NopLogger is used.
        Logger watermill.LoggerAdapter
}

func (c *EventBusConfig) setDefaults() {
        if c.Logger == nil {
                c.Logger = watermill.NopLogger{}
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_bus.go

Event Processor#

// ...
// EventProcessor determines which EventHandler should handle event received fro
m event bus.
type EventProcessor struct {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_processor.go
// ...
type EventProcessorConfig struct {
        // GenerateSubscribeTopic is used to generate topic for subscribing to e
vents.
        // If event processor is using handler groups, GenerateSubscribeTopic is
 used instead.
        GenerateSubscribeTopic EventProcessorGenerateSubscribeTopicFn

        // SubscriberConstructor is used to create subscriber for EventHandler.
        //
        // This function is called for every EventHandler instance.
        // If you want to re-use one subscriber for multiple handlers, use Group
EventProcessor instead.
        SubscriberConstructor EventProcessorSubscriberConstructorFn

        // OnHandle is called before handling event.
        // OnHandle works in a similar way to middlewares: you can inject additi
onal logic before and after handling a event.
        //
        // Because of that, you need to explicitly call params.Handler.Handle()
to handle the event.
        //
        //  func(params EventProcessorOnHandleParams) (err error) {
        //      // logic before handle
        //      //  (...)
        //
        //      err := params.Handler.Handle(params.Message.Context(), params.Ev
ent)
        //
        //      // logic after handle
        //      //  (...)
        //
        //      return err
        //  }
        //
        // This option is not required.
        OnHandle EventProcessorOnHandleFn

        // AckOnUnknownEvent is used to decide if message should be acked if eve
nt has no handler defined.
        AckOnUnknownEvent bool

        // Marshaler is used to marshal and unmarshal events.
        // It is required.
        Marshaler CommandEventMarshaler

        // Logger instance used to log.
        // If not provided, watermill.NopLogger is used.
        Logger watermill.LoggerAdapter

        // disableRouterAutoAddHandlers is used to keep backwards compatibility.
        // it is set when EventProcessor is created by NewEventProcessor.
        // Deprecated: please migrate to NewEventProcessorWithConfig.
        disableRouterAutoAddHandlers bool
}

func (c *EventProcessorConfig) setDefaults() {
        if c.Logger == nil {
                c.Logger = watermill.NopLogger{}
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_processor.go

Event Group Processor#

// ...
// EventGroupProcessor determines which EventHandler should handle event receive
d from event bus.
// Compared to EventProcessor, EventGroupProcessor allows to have multiple handl
ers that share the same subscriber instance.
type EventGroupProcessor struct {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_processor_grou
   p.go
// ...
type EventGroupProcessorConfig struct {
        // GenerateSubscribeTopic is used to generate topic for subscribing to e
vents for handler groups.
        // This option is required for EventProcessor if handler groups are used
.
        GenerateSubscribeTopic EventGroupProcessorGenerateSubscribeTopicFn

        // SubscriberConstructor is used to create subscriber for GroupEventHand
ler.
        // This function is called for every events group once - thanks to that
it's possible to have one subscription per group.
        // It's useful, when we are processing events from one stream and we wan
t to do it in order.
        SubscriberConstructor EventGroupProcessorSubscriberConstructorFn

        // OnHandle is called before handling event.
        // OnHandle works in a similar way to middlewares: you can inject additi
onal logic before and after handling a event.
        //
        // Because of that, you need to explicitly call params.Handler.Handle()
to handle the event.
        //
        //  func(params EventGroupProcessorOnHandleParams) (err error) {
        //      // logic before handle
        //      //  (...)
        //
        //      err := params.Handler.Handle(params.Message.Context(), params.Ev
ent)
        //
        //      // logic after handle
        //      //  (...)
        //
        //      return err
        //  }
        //
        // This option is not required.
        OnHandle EventGroupProcessorOnHandleFn

        // AckOnUnknownEvent is used to decide if message should be acked if eve
nt has no handler defined.
        AckOnUnknownEvent bool

        // Marshaler is used to marshal and unmarshal events.
        // It is required.
        Marshaler CommandEventMarshaler

        // Logger instance used to log.
        // If not provided, watermill.NopLogger is used.
        Logger watermill.LoggerAdapter
}

func (c *EventGroupProcessorConfig) setDefaults() {
        if c.Logger == nil {
                c.Logger = watermill.NopLogger{}
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_processor_grou
   p.go

   Learn more in Event Group Processor .

Event Handler#

// ...
// EventHandler receives events defined by NewEvent and handles them with its Ha
ndle method.
// If using DDD, CommandHandler may modify and persist the aggregate.
// It can also invoke a process manager, a saga or just build a read model.
//
// In contrast to CommandHandler, every Event can have multiple EventHandlers.
//
// One instance of EventHandler is used during handling messages.
// When multiple events are delivered at the same time, Handle method can be exe
cuted multiple times at the same time.
// Because of that, Handle method needs to be thread safe!
type EventHandler interface {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_handler.go

Command#

   The command is a simple data structure, representing the request for
   executing some operation.

Command Bus#

// ...
// CommandBus transports commands to command handlers.
type CommandBus struct {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_bus.go
// ...
type CommandBusConfig struct {
        // GeneratePublishTopic is used to generate topic for publishing command
.
        GeneratePublishTopic CommandBusGeneratePublishTopicFn

        // OnSend is called before publishing the command.
        // The *message.Message can be modified.
        //
        // This option is not required.
        OnSend CommandBusOnSendFn

        // Marshaler is used to marshal and unmarshal commands.
        // It is required.
        Marshaler CommandEventMarshaler

        // Logger instance used to log.
        // If not provided, watermill.NopLogger is used.
        Logger watermill.LoggerAdapter
}

func (c *CommandBusConfig) setDefaults() {
        if c.Logger == nil {
                c.Logger = watermill.NopLogger{}
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_bus.go

Command Processor#

// ...
// CommandProcessorSubscriberConstructorFn creates subscriber for CommandHandler
.
// It allows you to create a separate customized Subscriber for every command ha
ndler.
type CommandProcessorSubscriberConstructorFn func(CommandProcessorSubscriberCons
tructorParams) (message.Subscriber, error)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_processor.go
// ...
type CommandProcessorConfig struct {
        // GenerateSubscribeTopic is used to generate topic for subscribing comm
and.
        GenerateSubscribeTopic CommandProcessorGenerateSubscribeTopicFn

        // SubscriberConstructor is used to create subscriber for CommandHandler
.
        SubscriberConstructor CommandProcessorSubscriberConstructorFn

        // OnHandle is called before handling command.
        // OnHandle works in a similar way to middlewares: you can inject additi
onal logic before and after handling a command.
        //
        // Because of that, you need to explicitly call params.Handler.Handle()
to handle the command.
        //  func(params CommandProcessorOnHandleParams) (err error) {
        //      // logic before handle
        //      //  (...)
        //
        //      err := params.Handler.Handle(params.Message.Context(), params.Co
mmand)
        //
        //      // logic after handle
        //      //  (...)
        //
        //      return err
        //  }
        //
        // This option is not required.
        OnHandle CommandProcessorOnHandleFn

        // Marshaler is used to marshal and unmarshal commands.
        // It is required.
        Marshaler CommandEventMarshaler

        // Logger instance used to log.
        // If not provided, watermill.NopLogger is used.
        Logger watermill.LoggerAdapter

        // If true, CommandProcessor will ack messages even if CommandHandler re
turns an error.
        // If RequestReplyBackend is not null and sending reply fails, the messa
ge will be nack-ed anyway.
        //
        // Warning: It's not recommended to use this option when you are using r
equestreply component
        // (requestreply.NewCommandHandler or requestreply.NewCommandHandlerWith
Result), as it may ack the
        // command when sending reply failed.
        //
        // When you are using requestreply, you should use requestreply.PubSubBa
ckendConfig.AckCommandErrors.
        AckCommandHandlingErrors bool

        // disableRouterAutoAddHandlers is used to keep backwards compatibility.
        // it is set when CommandProcessor is created by NewCommandProcessor.
        // Deprecated: please migrate to NewCommandProcessorWithConfig.
        disableRouterAutoAddHandlers bool
}

func (c *CommandProcessorConfig) setDefaults() {
        if c.Logger == nil {
                c.Logger = watermill.NopLogger{}
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_processor.go

Command Handler#

// ...
// CommandHandler receives a command defined by NewCommand and handles it with t
he Handle method.
// If using DDD, CommandHandler may modify and persist the aggregate.
//
// In contrast to EventHandler, every Command must have only one CommandHandler.
//
// One instance of CommandHandler is used during handling messages.
// When multiple commands are delivered at the same time, Handle method can be e
xecuted multiple times at the same time.
// Because of that, Handle method needs to be thread safe!
type CommandHandler interface {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_handler.go

Command and Event Marshaler#

// ...
// CommandEventMarshaler marshals Commands and Events to Watermill's messages an
d vice versa.
// Payload of the command needs to be marshaled to []bytes.
type CommandEventMarshaler interface {
        // Marshal marshals Command or Event to Watermill's message.
        Marshal(v interface{}) (*message.Message, error)

        // Unmarshal unmarshals watermill's message to v Command or Event.
        Unmarshal(msg *message.Message, v interface{}) (err error)

        // Name returns the name of Command or Event.
        // Name is used to determine, that received command or event is event wh
ich we want to handle.
        Name(v interface{}) string

        // NameFromMessage returns the name of Command or Event from Watermill's
 message (generated by Marshal).
        //
        // When we have Command or Event marshaled to Watermill's message,
        // we should use NameFromMessage instead of Name to avoid unnecessary un
marshaling.
        NameFromMessage(msg *message.Message) string
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/marshaler.go

Usage#

Example domain#

   As an example, we will use a simple domain, that is responsible for
   handing room booking in a hotel.

   We will use Event Storming notation to show the model of this domain.

   Legend:
     * blue post-its are commands
     * orange post-its are events
     * green post-its are read models, asynchronously generated from
       events
     * violet post-its are policies, which are triggered by events and
       produce commands
     * pink post its are hot-spots; we mark places where problems often
       occur

   CQRS Event Storming

   The domain is simple:
     * A Guest is able to book a room.
     * Whenever a room is booked, we order a beer for the guest (because
       we love our guests).
          + We know that sometimes there are not enough beers.
     * We generate a financial report based on the bookings.

Sending a command#

   For the beginning, we need to simulate the guest’s action.
// ...
                bookRoomCmd := &BookRoom{
                        RoomId:    fmt.Sprintf("%d", i),
                        GuestName: "John",
                        StartDate: startDate,
                        EndDate:   endDate,
                }
                if err := commandBus.Send(context.Background(), bookRoomCmd); er
r != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

Command handler#

   BookRoomHandler will handle our command.
// ...
// BookRoomHandler is a command handler, which handles BookRoom command and emit
s RoomBooked.
//
// In CQRS, one command must be handled by only one handler.
// When another handler with this command is added to command processor, error w
ill be retuerned.
type BookRoomHandler struct {
        eventBus *cqrs.EventBus
}

func (b BookRoomHandler) Handle(ctx context.Context, cmd *BookRoom) error {
        // some random price, in production you probably will calculate in wiser
 way
        price := (rand.Int63n(40) + 1) * 10

        log.Printf(
                "Booked %s for %s from %s to %s",
                cmd.RoomId,
                cmd.GuestName,
                time.Unix(cmd.StartDate.Seconds, int64(cmd.StartDate.Nanos)),
                time.Unix(cmd.EndDate.Seconds, int64(cmd.EndDate.Nanos)),
        )

        // RoomBooked will be handled by OrderBeerOnRoomBooked event handler,
        // in future RoomBooked may be handled by multiple event handler
        if err := b.eventBus.Publish(ctx, &RoomBooked{
                ReservationId: watermill.NewUUID(),
                RoomId:        cmd.RoomId,
                GuestName:     cmd.GuestName,
                Price:         price,
                StartDate:     cmd.StartDate,
                EndDate:       cmd.EndDate,
        }); err != nil {
                return err
        }

        return nil
}

// OrderBeerOnRoomBooked is a event handler, which handles RoomBooked event and
emits OrderBeer command.
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

Event handler#

   As mentioned before, we want to order a beer every time when a room is
   booked (“Whenever a Room is booked” post-it). We do it by using the
   OrderBeer command.
// ...
// OrderBeerOnRoomBooked is a event handler, which handles RoomBooked event and
emits OrderBeer command.
type OrderBeerOnRoomBooked struct {
        commandBus *cqrs.CommandBus
}

func (o OrderBeerOnRoomBooked) Handle(ctx context.Context, event *RoomBooked) er
ror {
        orderBeerCmd := &OrderBeer{
                RoomId: event.RoomId,
                Count:  rand.Int63n(10) + 1,
        }

        return o.commandBus.Send(ctx, orderBeerCmd)
}

// OrderBeerHandler is a command handler, which handles OrderBeer command and em
its BeerOrdered.
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

   OrderBeerHandler is very similar to BookRoomHandler. The only
   difference is, that it sometimes returns an error when there are not
   enough beers, which causes redelivery of the command. You can find the
   entire implementation in the example source code .

Event Handler groups#

   By default, each event handler has a separate subscriber instance. It
   works fine, if just one event type is sent to the topic.

   In the scenario, when we have multiple event types on one topic, you
   have two options:
    1. You can set EventConfig.AckOnUnknownEvent to true - it will
       acknowledge all events that are not handled by handler,
    2. You can use Event Handler groups mechanism.

   To use event groups, you need to set GenerateHandlerGroupSubscribeTopic
   and GroupSubscriberConstructor options in EventConfig .

   After that, you can use AddHandlersGroup on EventProcessor .
// ...
        err = eventProcessor.AddHandlersGroup(
                "events",
                cqrs.NewGroupEventHandler(OrderBeerOnRoomBooked{commandBus}.Hand
le),
                cqrs.NewGroupEventHandler(NewBookingsFinancialReport().Handle),
                cqrs.NewGroupEventHandler(func(ctx context.Context, event *BeerO
rdered) error {
                        logger.Info("Beer ordered", watermill.LogFields{
                                "room_id": event.RoomId,
                        })
                        return nil
                }),
        )
        if err != nil {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

   Both GenerateHandlerGroupSubscribeTopic and GroupSubscriberConstructor
   receives information about group name in function arguments.

Generic handlers#

   Since Watermill v1.3 it’s possible to use generic handlers for commands
   and events. It’s useful when you have a lot of commands/events and you
   don’t want to create a handler for each of them.
// ...
                cqrs.NewGroupEventHandler(OrderBeerOnRoomBooked{commandBus}.Hand
le),
                cqrs.NewGroupEventHandler(NewBookingsFinancialReport().Handle),
                cqrs.NewGroupEventHandler(func(ctx context.Context, event *BeerO
rdered) error {
                        logger.Info("Beer ordered", watermill.LogFields{
                                "room_id": event.RoomId,
                        })
                        return nil
                }),
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

   Under the hood, it creates EventHandler or CommandHandler
   implementation. It’s available for all kind of handlers.
// ...
// NewCommandHandler creates a new CommandHandler implementation based on provid
ed function
// and command type inferred from function argument.
func NewCommandHandler[Command any](
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/command_handler.go
// ...
// NewEventHandler creates a new EventHandler implementation based on provided f
unction
// and event type inferred from function argument.
func NewEventHandler[T any](
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_handler.go
// ...
// NewGroupEventHandler creates a new GroupEventHandler implementation based on
provided function
// and event type inferred from function argument.
func NewGroupEventHandler[T any](handleFunc func(ctx context.Context, event *T)
error) GroupEventHandler {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/components/cqrs/event_handler.go

Building a read model with the event handler#

// ...
// BookingsFinancialReport is a read model, which calculates how much money we m
ay earn from bookings.
// Like OrderBeerOnRoomBooked, it listens for RoomBooked event.
//
// This implementation is just writing to the memory. In production, you will pr
obably will use some persistent storage.
type BookingsFinancialReport struct {
        handledBookings map[string]struct{}
        totalCharge     int64
        lock            sync.Mutex
}

func NewBookingsFinancialReport() *BookingsFinancialReport {
        return &BookingsFinancialReport{handledBookings: map[string]struct{}{}}
}

func (b *BookingsFinancialReport) Handle(ctx context.Context, event *RoomBooked)
 error {
        // Handle may be called concurrently, so it need to be thread safe.
        b.lock.Lock()
        defer b.lock.Unlock()

        // When we are using Pub/Sub which doesn't provide exactly-once delivery
 semantics, we need to deduplicate messages.
        // GoChannel Pub/Sub provides exactly-once delivery,
        // but let's make this example ready for other Pub/Sub implementations.
        if _, ok := b.handledBookings[event.ReservationId]; ok {
                return nil
        }
        b.handledBookings[event.ReservationId] = struct{}{}

        b.totalCharge += event.Price

        fmt.Printf(">>> Already booked rooms for $%d\n", b.totalCharge)
        return nil
}

var amqpAddress = "amqp://guest:guest@rabbitmq:5672/"

func main() {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

Wiring it up#

   We have all the blocks to build our CQRS application.

   We will use the AMQP (RabbitMQ) as our message broker: AMQP .

   Under the hood, CQRS is using Watermill’s message router. If you are
   not familiar with it and want to learn how it works, you should check
   Getting Started guide . It will also show you how to use some standard
   messaging patterns, like metrics, poison queue, throttling, correlation
   and other tools used by every message-driven application. Those come
   built-in with Watermill.

   Let’s go back to the CQRS. As you already know, CQRS is built from
   multiple components, like Command or Event buses, handlers, processors,
   etc.
// ...
func main() {
        logger := watermill.NewStdLogger(false, false)
        cqrsMarshaler := cqrs.ProtobufMarshaler{}

        // You can use any Pub/Sub implementation from here: https://watermill.i
o/pubsubs/
        // Detailed RabbitMQ implementation: https://watermill.io/pubsubs/amqp/
        // Commands will be send to queue, because they need to be consumed once
.
        commandsAMQPConfig := amqp.NewDurableQueueConfig(amqpAddress)
        commandsPublisher, err := amqp.NewPublisher(commandsAMQPConfig, logger)
        if err != nil {
                panic(err)
        }
        commandsSubscriber, err := amqp.NewSubscriber(commandsAMQPConfig, logger
)
        if err != nil {
                panic(err)
        }

        // Events will be published to PubSub configured Rabbit, because they ma
y be consumed by multiple consumers.
        // (in that case BookingsFinancialReport and OrderBeerOnRoomBooked).
        eventsPublisher, err := amqp.NewPublisher(amqp.NewDurablePubSubConfig(am
qpAddress, nil), logger)
        if err != nil {
                panic(err)
        }

        // CQRS is built on messages router. Detailed documentation: https://wat
ermill.io/docs/messages-router/
        router, err := message.NewRouter(message.RouterConfig{}, logger)
        if err != nil {
                panic(err)
        }

        // Simple middleware which will recover panics from event or command han
dlers.
        // More about router middlewares you can find in the documentation:
        // https://watermill.io/docs/messages-router/#middleware
        //
        // List of available middlewares you can find in message/router/middlewa
re.
        router.AddMiddleware(middleware.Recoverer)

        commandBus, err := cqrs.NewCommandBusWithConfig(commandsPublisher, cqrs.
CommandBusConfig{
                GeneratePublishTopic: func(params cqrs.CommandBusGeneratePublish
TopicParams) (string, error) {
                        // we are using queue RabbitMQ config, so we need to hav
e topic per command type
                        return params.CommandName, nil
                },
                OnSend: func(params cqrs.CommandBusOnSendParams) error {
                        logger.Info("Sending command", watermill.LogFields{
                                "command_name": params.CommandName,
                        })

                        params.Message.Metadata.Set("sent_at", time.Now().String
())

                        return nil
                },
                Marshaler: cqrsMarshaler,
                Logger:    logger,
        })
        if err != nil {
                panic(err)
        }

        commandProcessor, err := cqrs.NewCommandProcessorWithConfig(
                router,
                cqrs.CommandProcessorConfig{
                        GenerateSubscribeTopic: func(params cqrs.CommandProcesso
rGenerateSubscribeTopicParams) (string, error) {
                                // we are using queue RabbitMQ config, so we nee
d to have topic per command type
                                return params.CommandName, nil
                        },
                        SubscriberConstructor: func(params cqrs.CommandProcessor
SubscriberConstructorParams) (message.Subscriber, error) {
                                // we can reuse subscriber, because all commands
 have separated topics
                                return commandsSubscriber, nil
                        },
                        OnHandle: func(params cqrs.CommandProcessorOnHandleParam
s) error {
                                start := time.Now()

                                err := params.Handler.Handle(params.Message.Cont
ext(), params.Command)

                                logger.Info("Command handled", watermill.LogFiel
ds{
                                        "command_name": params.CommandName,
                                        "duration":     time.Since(start),
                                        "err":          err,
                                })

                                return err
                        },
                        Marshaler: cqrsMarshaler,
                        Logger:    logger,
                },
        )
        if err != nil {
                panic(err)
        }

        eventBus, err := cqrs.NewEventBusWithConfig(eventsPublisher, cqrs.EventB
usConfig{
                GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopic
Params) (string, error) {
                        // because we are using PubSub RabbitMQ config, we can u
se one topic for all events
                        return "events", nil

                        // we can also use topic per event type
                        // return params.EventName, nil
                },

                OnPublish: func(params cqrs.OnEventSendParams) error {
                        logger.Info("Publishing event", watermill.LogFields{
                                "event_name": params.EventName,
                        })

                        params.Message.Metadata.Set("published_at", time.Now().S
tring())

                        return nil
                },

                Marshaler: cqrsMarshaler,
                Logger:    logger,
        })
        if err != nil {
                panic(err)
        }

        eventProcessor, err := cqrs.NewEventGroupProcessorWithConfig(
                router,
                cqrs.EventGroupProcessorConfig{
                        GenerateSubscribeTopic: func(params cqrs.EventGroupProce
ssorGenerateSubscribeTopicParams) (string, error) {
                                return "events", nil
                        },
                        SubscriberConstructor: func(params cqrs.EventGroupProces
sorSubscriberConstructorParams) (message.Subscriber, error) {
                                config := amqp.NewDurablePubSubConfig(
                                        amqpAddress,
                                        amqp.GenerateQueueNameTopicNameWithSuffi
x(params.EventGroupName),
                                )

                                return amqp.NewSubscriber(config, logger)
                        },

                        OnHandle: func(params cqrs.EventGroupProcessorOnHandlePa
rams) error {
                                start := time.Now()

                                err := params.Handler.Handle(params.Message.Cont
ext(), params.Event)

                                logger.Info("Event handled", watermill.LogFields
{
                                        "event_name": params.EventName,
                                        "duration":   time.Since(start),
                                        "err":        err,
                                })

                                return err
                        },

                        Marshaler: cqrsMarshaler,
                        Logger:    logger,
                },
        )
        if err != nil {
                panic(err)
        }

        err = commandProcessor.AddHandlers(
                cqrs.NewCommandHandler("BookRoomHandler", BookRoomHandler{eventB
us}.Handle),
                cqrs.NewCommandHandler("OrderBeerHandler", OrderBeerHandler{even
tBus}.Handle),
        )
        if err != nil {
                panic(err)
        }

        err = eventProcessor.AddHandlersGroup(
                "events",
                cqrs.NewGroupEventHandler(OrderBeerOnRoomBooked{commandBus}.Hand
le),
                cqrs.NewGroupEventHandler(NewBookingsFinancialReport().Handle),
                cqrs.NewGroupEventHandler(func(ctx context.Context, event *BeerO
rdered) error {
                        logger.Info("Beer ordered", watermill.LogFields{
                                "room_id": event.RoomId,
                        })
                        return nil
                }),
        )
        if err != nil {
                panic(err)
        }

        // publish BookRoom commands every second to simulate incoming traffic
        go publishCommands(commandBus)

        // processors are based on router, so they will work when router will st
art
        if err := router.Run(context.Background()); err != nil {
                panic(err)
        }
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/5-cqrs-protobuf/main
   .go

   And that’s all. We have a working CQRS application.

What’s next?#

   As mentioned before, if you are not familiar with Watermill, we highly
   recommend reading Getting Started guide .
   Help us improve this page
   Prev
   Middleware
   Next
   Troubleshooting

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/cqrs/index.txt ===

=== BEGIN: docs/getting-started/index.txt ===
On this page

     * What is Watermill?
     * Why use Watermill?
     * Install
     * One-Minute Background
     * Publisher & Subscriber
          + Subscribing for Messages
          + Creating Messages
          + Publishing Messages
     * Router
          + Router configuration
          + Handlers
     * Logging
     * What’s next?
     * Examples
     * Support

Getting started

   On this page
     * What is Watermill?
     * Why use Watermill?
     * Install
     * One-Minute Background
     * Publisher & Subscriber
          + Subscribing for Messages
          + Creating Messages
          + Publishing Messages
     * Router
          + Router configuration
          + Handlers
     * Logging
     * What’s next?
     * Examples
     * Support

What is Watermill?#

   Watermill is a Go library for working with message streams. You can use
   it to build event-driven systems with popular Pub/Sub implementations
   like Kafka or RabbitMQ, as well as HTTP or Postgres if that fits your
   use case. It comes with a set of Pub/Sub implementations and can be
   easily extended.

   Watermill also ships with standard middlewares like instrumentation,
   poison queue, throttling, correlation, and other tools used by every
   message-driven application.

Why use Watermill?#

   When using microservices, synchronous communication is not always the
   right choice. Asynchronous methods became a new standard way to
   communicate.

   While there are many tools and libraries for synchronous communication,
   like HTTP, correctly setting up a message-oriented project can be
   challenging. There are many different message queues and streaming
   systems, each with different features, client libraries, and APIs.

   Watermill aims to be the standard messaging library for Go, hiding all
   that complexity behind an API that is easy to understand. It provides
   all you need to build an application based on events or other
   asynchronous patterns.

   Watermill is NOT a framework. It’s a lightweight library that’s easy to
   plug in or remove from your project.

Install#

go get -u github.com/ThreeDotsLabs/watermill

One-Minute Background#

   The idea behind event-driven applications is always the same: listen to
   and react to incoming messages. Watermill supports this behavior for
   multiple publishers and subscribers .

   The core part of Watermill is the Message . It is what http.Request is
   for the net/http package. Most Watermill features work with this
   struct.

   Watermill provides a few APIs for working with messages. They build on
   top of each other, each step providing a higher-level API:
     * At the bottom, the Publisher and Subscriber interfaces. It’s the
       “raw” way of working with messages. You get full control, but also
       need to handle everything yourself.
     * The Router is similar to HTTP routers you probably know. It
       introduces message handlers.
     * The CQRS component adds generic handlers without needing to marshal
       and unmarshal messages yourself.

   Watermill components pyramid

Publisher & Subscriber#

   Most Pub/Sub libraries come with complex features. For Watermill, it’s
   enough to implement two interfaces to start working with them: the
   Publisher and Subscriber.
type Publisher interface {
        Publish(topic string, messages ...*Message) error
        Close() error
}

type Subscriber interface {
        Subscribe(ctx context.Context, topic string) (<-chan *Message, error)
        Close() error
}

Subscribing for Messages#

   Subscribe expects a topic name and returns a channel of incoming
   messages. What topic exactly means depends on the Pub/Sub
   implementation. Usually, it needs to match the topic name used by the
   publisher.
messages, err := subscriber.Subscribe(ctx, "example.topic")
if err != nil {
        panic(err)
}

for msg := range messages {
        fmt.Printf("received message: %s, payload: %s\n", msg.UUID, string(msg.P
ayload))
        msg.Ack()
}

   See detailed examples below for supported PubSubs.
   (BUTTON) Go Channel (BUTTON) Kafka (BUTTON) NATS Streaming (BUTTON)
   Google Cloud Pub/Sub (BUTTON) RabbitMQ (AMQP) (BUTTON) SQL (BUTTON) AWS
   SQS (BUTTON) AWS SNS
// ...
package main

import (
        "context"
        "fmt"
        "time"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill/message"
        "github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

func main() {
        pubSub := gochannel.NewGoChannel(
                gochannel.Config{},
                watermill.NewStdLogger(false, false),
        )

        messages, err := pubSub.Subscribe(context.Background(), "example.topic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/go-channel/main.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                fmt.Printf("received message: %s, payload: %s\n", msg.UUID, stri
ng(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/go-channel/main.go
   Running in Docker

   The easiest way to run Watermill locally with Kafka is by using Docker.
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    depends_on:
      - kafka
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  zookeeper:
    image: confluentinc/cp-zookeeper:7.3.1
    restart: unless-stopped
    logging:
      driver: none
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181

  kafka:
    image: confluentinc/cp-kafka:7.3.1
    restart: unless-stopped
    depends_on:
      - zookeeper
    logging:
      driver: none
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"

   Full source: _examples/pubsubs/kafka/docker-compose.yml

   The source should go to main.go.

   To run, execute the docker-compose up command.

   A more detailed explanation of how it works (and how to add live code
   reload) can be found in the Go Docker dev environment article .
// ...
package main

import (
        "context"
        "log"
        "time"

        "github.com/IBM/sarama"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        saramaSubscriberConfig := kafka.DefaultSaramaSubscriberConfig()
        // equivalent of auto.offset.reset: earliest
        saramaSubscriberConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

        subscriber, err := kafka.NewSubscriber(
                kafka.SubscriberConfig{
                        Brokers:               []string{"kafka:9092"},
                        Unmarshaler:           kafka.DefaultMarshaler{},
                        OverwriteSaramaConfig: saramaSubscriberConfig,
                        ConsumerGroup:         "test_consumer_group",
                },
                watermill.NewStdLogger(false, false),
        )
        if err != nil {
                panic(err)
        }

        messages, err := subscriber.Subscribe(context.Background(), "example.top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/kafka/main.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/kafka/main.go
   Running in Docker

   The easiest way to run Watermill locally with NATS is using Docker.
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    depends_on:
      - nats-streaming
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  nats-streaming:
    image: nats-streaming:0.11.2
    restart: unless-stopped

   Full source: _examples/pubsubs/nats-streaming/docker-compose.yml

   The source should go to main.go.

   To run, execute the docker-compose up command.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        "log"
        "time"

        stan "github.com/nats-io/stan.go"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-nats/pkg/nats"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        subscriber, err := nats.NewStreamingSubscriber(
                nats.StreamingSubscriberConfig{
                        ClusterID:        "test-cluster",
                        ClientID:         "example-subscriber",
                        QueueGroup:       "example",
                        DurableName:      "my-durable",
                        SubscribersCount: 4, // how many goroutines should consu
me messages
                        CloseTimeout:     time.Minute,
                        AckWaitTimeout:   time.Second * 30,
                        StanOptions: []stan.Option{
                                stan.NatsURL("nats://nats-streaming:4222"),
                        },
                        Unmarshaler: nats.GobMarshaler{},
                },
                watermill.NewStdLogger(false, false),
        )
        if err != nil {
                panic(err)
        }

        messages, err := subscriber.Subscribe(context.Background(), "example.top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/nats-streaming/mai
   n.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/nats-streaming/mai
   n.go
   Running in Docker

   You can run the Google Cloud Pub/Sub emulator locally for development.
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    depends_on:
      - googlecloud
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    environment:
      # use local emulator instead of google cloud engine
      PUBSUB_EMULATOR_HOST: "googlecloud:8085"
    working_dir: /app
    command: go run main.go

  googlecloud:
    image: google/cloud-sdk:414.0.0
    entrypoint: gcloud --quiet beta emulators pubsub start --host-port=0.0.0.0:8
085 --verbosity=debug --log-http
    restart: unless-stopped

   Full source: _examples/pubsubs/googlecloud/docker-compose.yml

   The source should go to main.go.

   To run, execute docker-compose up.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        "log"
        "time"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        logger := watermill.NewStdLogger(false, false)
        subscriber, err := googlecloud.NewSubscriber(
                googlecloud.SubscriberConfig{
                        // custom function to generate Subscription Name,
                        // there are also predefined TopicSubscriptionName and T
opicSubscriptionNameWithSuffix available.
                        GenerateSubscriptionName: func(topic string) string {
                                return "test-sub_" + topic
                        },
                        ProjectID: "test-project",
                },
                logger,
        )
        if err != nil {
                panic(err)
        }

        // Subscribe will create the subscription. Only messages that are sent a
fter the subscription is created may be received.
        messages, err := subscriber.Subscribe(context.Background(), "example.top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/googlecloud/main.g
   o
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/googlecloud/main.g
   o
   Running in Docker
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    depends_on:
      - rabbitmq
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  rabbitmq:
    image: rabbitmq:3.7
    restart: unless-stopped

   Full source: _examples/pubsubs/amqp/docker-compose.yml

   The source should go to main.go.

   To run, execute docker-compose up.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        "log"
        "time"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
        "github.com/ThreeDotsLabs/watermill/message"
)

var amqpURI = "amqp://guest:guest@rabbitmq:5672/"

func main() {
        amqpConfig := amqp.NewDurableQueueConfig(amqpURI)

        subscriber, err := amqp.NewSubscriber(
                // This config is based on this example: https://www.rabbitmq.co
m/tutorials/tutorial-two-go.html
                // It works as a simple queue.
                //
                // If you want to implement a Pub/Sub style service instead, che
ck
                // https://watermill.io/pubsubs/amqp/#amqp-consumer-groups
                amqpConfig,
                watermill.NewStdLogger(false, false),
        )
        if err != nil {
                panic(err)
        }

        messages, err := subscriber.Subscribe(context.Background(), "example.top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/amqp/main.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/amqp/main.go
   Running in Docker
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    depends_on:
      - mysql
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  mysql:
    image: mysql:8.0
    restart: unless-stopped
    ports:
      - 3306:3306
    environment:
      MYSQL_DATABASE: watermill
      MYSQL_ALLOW_EMPTY_PASSWORD: "yes"

   Full source: _examples/pubsubs/sql/docker-compose.yml

   The source should go to main.go.

   To run, execute docker-compose up.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        stdSQL "database/sql"
        "log"
        "time"

        driver "github.com/go-sql-driver/mysql"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        db := createDB()
        logger := watermill.NewStdLogger(false, false)

        subscriber, err := sql.NewSubscriber(
                db,
                sql.SubscriberConfig{
                        SchemaAdapter:    sql.DefaultMySQLSchema{},
                        OffsetsAdapter:   sql.DefaultMySQLOffsetsAdapter{},
                        InitializeSchema: true,
                },
                logger,
        )
        if err != nil {
                panic(err)
        }

        messages, err := subscriber.Subscribe(context.Background(), "example_top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/sql/main.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/sql/main.go
   Running in Docker
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  localstack:
    image: localstack/localstack:latest
    environment:
      - SERVICES=sqs,sns
      - AWS_DEFAULT_REGION=us-east-1
      - EDGE_PORT=4566
    ports:
      - "4566-4597:4566-4597"
    healthcheck:
      test: awslocal sqs list-queues && awslocal sns list-topics
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 30s

   Full source: _examples/pubsubs/aws-sqs/docker-compose.yml

   The source should go to main.go.

   To run, execute docker-compose up.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        "log"
        "net/url"
        "time"

        "github.com/aws/aws-sdk-go-v2/aws"
        amazonsqs "github.com/aws/aws-sdk-go-v2/service/sqs"
        transport "github.com/aws/smithy-go/endpoints"
        "github.com/samber/lo"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-aws/sqs"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        logger := watermill.NewStdLogger(false, false)

        sqsOpts := []func(*amazonsqs.Options){
                amazonsqs.WithEndpointResolverV2(sqs.OverrideEndpointResolver{
                        Endpoint: transport.Endpoint{
                                URI: *lo.Must(url.Parse("http://localstack:4566"
)),
                        },
                }),
        }

        subscriberConfig := sqs.SubscriberConfig{
                AWSConfig: aws.Config{
                        Credentials: aws.AnonymousCredentials{},
                },
                OptFns: sqsOpts,
        }

        subscriber, err := sqs.NewSubscriber(subscriberConfig, logger)
        if err != nil {
                panic(err)
        }

        messages, err := subscriber.Subscribe(context.Background(), "example-top
ic")
        if err != nil {
                panic(err)
        }

        go process(messages)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sqs/main.go
// ...
func process(messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("received message: %s, payload: %s", msg.UUID, string
(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sqs/main.go
   Running in Docker
services:
  server:
    image: golang:1.23
    restart: unless-stopped
    volumes:
      - .:/app
      - $GOPATH/pkg/mod:/go/pkg/mod
    working_dir: /app
    command: go run main.go

  localstack:
    image: localstack/localstack:latest
    environment:
      - SERVICES=sqs,sns
      - AWS_DEFAULT_REGION=us-east-1
      - EDGE_PORT=4566
    ports:
      - "4566-4597:4566-4597"
    healthcheck:
      test: awslocal sqs list-queues && awslocal sns list-topics
      interval: 5s
      timeout: 5s
      retries: 5
      start_period: 30s

   Full source: _examples/pubsubs/aws-sns/docker-compose.yml

   The source should go to main.go.

   To run, execute docker-compose up.

   A more detailed explanation of how it is working (and how to add live
   code reload) can be found in Go Docker dev environment article .
// ...
package main

import (
        "context"
        "fmt"
        "log"
        "net/url"
        "time"

        "github.com/aws/aws-sdk-go-v2/aws"
        amazonsns "github.com/aws/aws-sdk-go-v2/service/sns"
        amazonsqs "github.com/aws/aws-sdk-go-v2/service/sqs"
        transport "github.com/aws/smithy-go/endpoints"
        "github.com/samber/lo"

        "github.com/ThreeDotsLabs/watermill"
        "github.com/ThreeDotsLabs/watermill-aws/sns"
        "github.com/ThreeDotsLabs/watermill-aws/sqs"
        "github.com/ThreeDotsLabs/watermill/message"
)

func main() {
        logger := watermill.NewStdLogger(false, false)

        snsOpts := []func(*amazonsns.Options){
                amazonsns.WithEndpointResolverV2(sns.OverrideEndpointResolver{
                        Endpoint: transport.Endpoint{
                                URI: *lo.Must(url.Parse("http://localstack:4566"
)),
                        },
                }),
        }

        sqsOpts := []func(*amazonsqs.Options){
                amazonsqs.WithEndpointResolverV2(sqs.OverrideEndpointResolver{
                        Endpoint: transport.Endpoint{
                                URI: *lo.Must(url.Parse("http://localstack:4566"
)),
                        },
                }),
        }

        topicResolver, err := sns.NewGenerateArnTopicResolver("000000000000", "u
s-east-1")
        if err != nil {
                panic(err)
        }

        newSubscriber := func(name string) (message.Subscriber, error) {
                subscriberConfig := sns.SubscriberConfig{
                        AWSConfig: aws.Config{
                                Credentials: aws.AnonymousCredentials{},
                        },
                        OptFns:        snsOpts,
                        TopicResolver: topicResolver,
                        GenerateSqsQueueName: func(ctx context.Context, snsTopic
 sns.TopicArn) (string, error) {
                                topic, err := sns.ExtractTopicNameFromTopicArn(s
nsTopic)
                                if err != nil {
                                        return "", err
                                }

                                return fmt.Sprintf("%v-%v", topic, name), nil
                        },
                }

                sqsSubscriberConfig := sqs.SubscriberConfig{
                        AWSConfig: aws.Config{
                                Credentials: aws.AnonymousCredentials{},
                        },
                        OptFns: sqsOpts,
                }

                return sns.NewSubscriber(subscriberConfig, sqsSubscriberConfig,
logger)
        }

        subA, err := newSubscriber("subA")
        if err != nil {
                panic(err)
        }

        subB, err := newSubscriber("subB")
        if err != nil {
                panic(err)
        }

        messagesA, err := subA.Subscribe(context.Background(), "example-topic")
        if err != nil {
                panic(err)
        }

        messagesB, err := subB.Subscribe(context.Background(), "example-topic")
        if err != nil {
                panic(err)
        }

        go process("A", messagesA)
        go process("B", messagesB)
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sns/main.go
// ...
func process(prefix string, messages <-chan *message.Message) {
        for msg := range messages {
                log.Printf("%v received message: %s, payload: %s", prefix, msg.U
UID, string(msg.Payload))

                // we need to Acknowledge that we received and processed the mes
sage,
                // otherwise, it will be resent over and over again.
                msg.Ack()
        }
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sns/main.go

Creating Messages#

   Watermill doesn’t enforce any message format. NewMessage expects a
   slice of bytes as the payload. You can use strings, JSON, protobuf,
   Avro, gob, or anything else that serializes to []byte.

   The message UUID is optional but recommended for debugging.
msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, world!"))

Publishing Messages#

   Publish expects a topic and one or more Messages to be published.
err := publisher.Publish("example.topic", msg)
if err != nil {
    panic(err)
}

   (BUTTON) Go Channel (BUTTON) Kafka (BUTTON) NATS Streaming (BUTTON)
   Google Cloud Pub/Sub (BUTTON) RabbitMQ (AMQP) (BUTTON) SQL (BUTTON) AWS
   SQS (BUTTON) AWS SNS
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example.topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/go-channel/main.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example.topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/kafka/main.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example.topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/nats-streaming/mai
   n.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example.topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/googlecloud/main.g
   o
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example.topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/amqp/main.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte(`{"message
": "Hello, world!"}`))

                if err := publisher.Publish("example_topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/sql/main.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example-topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sqs/main.go
// ...
                msg := message.NewMessage(watermill.NewUUID(), []byte("Hello, wo
rld!"))

                if err := publisher.Publish("example-topic", msg); err != nil {
                        panic(err)
                }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/pubsubs/aws-sns/main.go

Router#

   Publishers and subscribers are the low-level parts of Watermill. For
   most cases, you want to use a high-level API: Router component.

Router configuration#

   Start with configuring the router and adding plugins and middlewares.

   A middleware is a function executed for each incoming message. You can
   use one of the existing ones for things like correlation, metrics,
   poison queue, retrying, throttling, etc. . You can also create your
   own.
// ...
        router, err := message.NewRouter(message.RouterConfig{}, logger)
        if err != nil {
                panic(err)
        }

        // SignalsHandler will gracefully shutdown Router when SIGTERM is receiv
ed.
        // You can also close the router by just calling `r.Close()`.
        router.AddPlugin(plugin.SignalsHandler)

        // Router level middleware are executed for every message sent to the ro
uter
        router.AddMiddleware(
                // CorrelationID will copy the correlation id from the incoming
message's metadata to the produced messages
                middleware.CorrelationID,

                // The handler function is retried if it returns an error.
                // After MaxRetries, the message is Nacked and it's up to the Pu
bSub to resend it.
                middleware.Retry{
                        MaxRetries:      3,
                        InitialInterval: time.Millisecond * 100,
                        Logger:          logger,
                }.Middleware,

                // Recoverer handles panics from handlers.
                // In this case, it passes them as errors to the Retry middlewar
e.
                middleware.Recoverer,
        )
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

Handlers#

   Set up handlers that the router uses. Each handler independently
   handles incoming messages.

   A handler listens to messages from the given subscriber and topic. Any
   messages returned from the handler function will be published to the
   given publisher and topic.
// ...
        // AddHandler returns a handler which can be used to add handler level m
iddleware
        // or to stop handler.
        handler := router.AddHandler(
                "struct_handler",          // handler name, must be unique
                "incoming_messages_topic", // topic from which we will read even
ts
                pubSub,
                "outgoing_messages_topic", // topic to which we will publish eve
nts
                pubSub,
                structHandler{}.Handler,
        )
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

   Note: the example above uses one pubSub argument for both the
   subscriber and publisher. It’s because we use the GoChannel
   implementation, which is a simple in-memory Pub/Sub.

   Alternatively, if you don’t plan to publish messages from within the
   handler, you can use the simpler AddNoPublisherHandler method.
// ...
        router.AddNoPublisherHandler(
                "print_incoming_messages",
                "incoming_messages_topic",
                pubSub,
                printMessages,
        )
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

   You can use two types of handler functions:
    1. a function func(msg *message.Message) ([]*message.Message, error)
    2. a struct method func (c structHandler) Handler(msg
       *message.Message) ([]*message.Message, error)

   Use the first one if your handler is a function without any
   dependencies. The second option is useful when your handler requires
   dependencies such as a database handle or a logger.
// ...
func printMessages(msg *message.Message) error {
        fmt.Printf(
                "\n> Received message: %s\n> %s\n> metadata: %v\n\n",
                msg.UUID, string(msg.Payload), msg.Metadata,
        )
        return nil
}

type structHandler struct {
        // we can add some dependencies here
}

func (s structHandler) Handler(msg *message.Message) ([]*message.Message, error)
 {
        log.Println("structHandler received message", msg.UUID)

        msg = message.NewMessage(watermill.NewUUID(), []byte("message produced b
y structHandler"))
        return message.Messages{msg}, nil
}

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

   Finally, run the router.
// ...
        if err := router.Run(ctx); err != nil {
                panic(err)
        }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

   The complete example’s source can be found at
   /_examples/basic/3-router/main.go .

Logging#

   To see Watermill’s logs, pass any logger that implements the
   LoggerAdapter . For experimental development, you can use NewStdLogger.

   Watermill provides ready-to-use slog adapter. You can create it with
   watermill.NewSlogLogger . You can also map Watermill’s log levels to
   slog levels with watermill.NewSlogLoggerWithLevelMapping .

What’s next?#

   For more details, see documentation topics .

   See the CQRS component for another high-level API.

Examples#

   Check out the examples that will show you how to start using Watermill.

   The recommended entry point is Your first Watermill application . It
   contains the entire environment in docker-compose.yml, including Go and
   Kafka, which you can run with one command.

   After that, you can see the Realtime feed example. It uses more
   middlewares and contains two handlers.

   For a different subscriber implementation (HTTP), see the
   receiving-webhooks example. It is a straightforward application that
   saves webhooks to Kafka.

   You can find the complete list of examples in the README .

Support#

   If anything is not clear, feel free to use any of our support channels
   ; we will be glad to help.
   Help us improve this page
   Next
   Message

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/getting-started/index.txt ===

=== BEGIN: docs/index.txt ===

=== END: docs/index.txt ===

=== BEGIN: docs/message/index.txt ===
On this page

     * Ack
          +
     * Nack
          +
     * Context

Message

   On this page
     * Ack
          +
     * Nack
          +
     * Context

   Message is one of core parts of Watermill. Messages are emitted by
   Publishers and received by Subscribers . When a message is processed,
   you should send an Ack() or a Nack() when the processing failed.

   Acks and Nacks are processed by Subscribers (in default
   implementations, the subscribers are waiting for an Ack or a Nack).
// ...
type Message struct {
        // UUID is a unique identifier of the message.
        //
        // It is only used by Watermill for debugging.
        // UUID can be empty.
        UUID string

        // Metadata contains the message metadata.
        //
        // Can be used to store data which doesn't require unmarshalling the ent
ire payload.
        // It is something similar to HTTP request's headers.
        //
        // Metadata is marshaled and will be saved to the PubSub.
        Metadata Metadata

        // Payload is the message's payload.
        Payload Payload

        // ack is closed when acknowledge is received.
        ack chan struct{}
        // noACk is closed when negative acknowledge is received.
        noAck chan struct{}

        ackMutex    sync.Mutex
        ackSentType ackType

        ctx context.Context
}

// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/message.go

Ack#

Sending Ack#

// ...
// Ack sends message's acknowledgement.
//
// Ack is not blocking.
// Ack is idempotent.
// False is returned, if Nack is already sent.
func (m *Message) Ack() bool {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/message.go

Nack#

// ...
// Nack sends message's negative acknowledgement.
//
// Nack is not blocking.
// Nack is idempotent.
// False is returned, if Ack is already sent.
func (m *Message) Nack() bool {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/message.go

Receiving Ack/Nack#

// ...
        select {
        case <-msg.Acked():
                log.Print("ack received")
        case <-msg.Nacked():
                log.Print("nack received")
        }
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/docs/content/docs/message/receiving-
   ack.go

Context#

   Message contains the standard library context, just like an HTTP
   request.
// ...
// Context returns the message's context. To change the context, use
// SetContext.
//
// The returned context is always non-nil; it defaults to the
// background context.
func (m *Message) Context() context.Context {
        if m.ctx != nil {
                return m.ctx
        }
        return context.Background()
}

// SetContext sets provided context to the message.
func (m *Message) SetContext(ctx context.Context) {
        m.ctx = ctx
}
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/message.go
   Help us improve this page
   Prev
   Getting started
   Next
   Pub/Sub

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/message/index.txt ===

=== BEGIN: docs/messages-router/index.txt ===
On this page

     * Configuration
     * Handler
     * No publisher handler
     * Ack
     * Producing messages
     * Running the Router
          + Ensuring that the Router is running
          + Closing the Router
     * Adding handler after the router has started
     * Stopping running handler
     * Execution models
     * Middleware
     * Plugin
     * Context

Router

   On this page
     * Configuration
     * Handler
     * No publisher handler
     * Ack
     * Producing messages
     * Running the Router
          + Ensuring that the Router is running
          + Closing the Router
     * Adding handler after the router has started
     * Stopping running handler
     * Execution models
     * Middleware
     * Plugin
     * Context

   Publishers and Subscribers are rather low-level parts of Watermill. In
   production use, you’d usually want to use a high-level interface and
   features like correlation, metrics, poison queue, retrying, throttling,
   etc. .

   You also might not want to send an Ack when processing was successful.
   Sometimes, you’d like to send a message after processing of another
   message finishes.

   To handle these requirements, there is a component named Router.
   Watermill Router

Configuration#

// ...
type RouterConfig struct {
        // CloseTimeout determines how long router should work for handlers when
 closing.
        CloseTimeout time.Duration
}

func (c *RouterConfig) setDefaults() {
        if c.CloseTimeout == 0 {
                c.CloseTimeout = time.Second * 30
        }
}

// Validate returns Router configuration error, if any.
func (c RouterConfig) Validate() error {
        return nil
}
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Handler#

   At the beginning you need to implement HandlerFunc:
// ...
// HandlerFunc is function called when message is received.
//
// msg.Ack() is called automatically when HandlerFunc doesn't return error.
// When HandlerFunc returns error, msg.Nack() is called.
// When msg.Ack() was called in handler and HandlerFunc returns error,
// msg.Nack() will be not sent because Ack was already sent.
//
// HandlerFunc's are executed parallel when multiple messages was received
// (because msg.Ack() was sent in HandlerFunc or Subscriber supports multiple co
nsumers).
type HandlerFunc func(msg *Message) ([]*Message, error)

// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   Next, you have to add a new handler with Router.AddHandler:
// ...
// AddHandler adds a new handler.
//
// handlerName must be unique. For now, it is used only for debugging.
//
// subscribeTopic is a topic from which handler will receive messages.
//
// publishTopic is a topic to which router will produce messages returned by han
dlerFunc.
// When handler needs to publish to multiple topics,
// it is recommended to just inject Publisher to Handler or implement middleware
// which will catch messages and publish to topic based on metadata for example.
//
// If handler is added while router is already running, you need to explicitly c
all RunHandlers().
func (r *Router) AddHandler(
        handlerName string,
        subscribeTopic string,
        subscriber Subscriber,
        publishTopic string,
        publisher Publisher,
        handlerFunc HandlerFunc,
) *Handler {
        r.logger.Info("Adding handler", watermill.LogFields{
                "handler_name": handlerName,
                "topic":        subscribeTopic,
        })

        r.handlersLock.Lock()
        defer r.handlersLock.Unlock()

        if _, ok := r.handlers[handlerName]; ok {
                panic(DuplicateHandlerNameError{handlerName})
        }

        publisherName, subscriberName := internal.StructName(publisher), interna
l.StructName(subscriber)

        newHandler := &handler{
                name:   handlerName,
                logger: r.logger,

                subscriber:     subscriber,
                subscribeTopic: subscribeTopic,
                subscriberName: subscriberName,

                publisher:     publisher,
                publishTopic:  publishTopic,
                publisherName: publisherName,

                handlerFunc: handlerFunc,

                runningHandlersWg:     r.runningHandlersWg,
                runningHandlersWgLock: r.runningHandlersWgLock,

                messagesCh:     nil,
                routersCloseCh: r.closingInProgressCh,

                startedCh: make(chan struct{}),
        }

        r.handlersWg.Add(1)
        r.handlers[handlerName] = newHandler

        select {
        case r.handlerAdded <- struct{}{}:
        default:
                // closeWhenAllHandlersStopped is not always waiting for handler
Added
        }

        return &Handler{
                router:  r,
                handler: newHandler,
        }
}

// AddNoPublisherHandler adds a new handler.
// This handler cannot return messages.
// When message is returned it will occur an error and Nack will be sent.
//
// handlerName must be unique. For now, it is used only for debugging.
//
// subscribeTopic is a topic from which handler will receive messages.
//
// subscriber is Subscriber from which messages will be consumed.
//
// If handler is added while router is already running, you need to explicitly c
all RunHandlers().
func (r *Router) AddNoPublisherHandler(
        handlerName string,
        subscribeTopic string,
        subscriber Subscriber,
        handlerFunc NoPublishHandlerFunc,
) *Handler {
        handlerFuncAdapter := func(msg *Message) ([]*Message, error) {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   See an example usage from Getting Started :
// ...
        // AddHandler returns a handler which can be used to add handler level m
iddleware
        // or to stop handler.
        handler := router.AddHandler(
                "struct_handler",          // handler name, must be unique
                "incoming_messages_topic", // topic from which we will read even
ts
                pubSub,
                "outgoing_messages_topic", // topic to which we will publish eve
nts
                pubSub,
                structHandler{}.Handler,
        )

        // Handler level middleware is only executed for a specific handler
        // Such middleware can be added the same way the router level ones
        handler.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
                return func(message *message.Message) ([]*message.Message, error
) {
                        log.Println("executing handler specific middleware for "
, message.UUID)

                        return h(message)
                }
        })

// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

No publisher handler#

   Not every handler will produce new messages. You can add this kind of
   handler by using Router.AddNoPublisherHandler:
// ...
// AddNoPublisherHandler adds a new handler.
// This handler cannot return messages.
// When message is returned it will occur an error and Nack will be sent.
//
// handlerName must be unique. For now, it is used only for debugging.
//
// subscribeTopic is a topic from which handler will receive messages.
//
// subscriber is Subscriber from which messages will be consumed.
//
// If handler is added while router is already running, you need to explicitly c
all RunHandlers().
func (r *Router) AddNoPublisherHandler(
        handlerName string,
        subscribeTopic string,
        subscriber Subscriber,
        handlerFunc NoPublishHandlerFunc,
) *Handler {
        handlerFuncAdapter := func(msg *Message) ([]*Message, error) {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Ack#

   By default, msg.Ack() is called when HanderFunc doesn’t return an
   error. If an error is returned, msg.Nack() will be called. Because of
   this, you don’t have to call msg.Ack() or msg.Nack() after a message is
   processed (you can if you want, of course).

Producing messages#

   When returning multiple messages from a handler, be aware that most
   Publisher implementations don’t support atomic publishing of messages .
   It may end up producing only some of messages and sending msg.Nack() if
   the broker or the storage are not available.

   If it is an issue, consider publishing just one message with each
   handler.

Running the Router#

   To run the Router, you need to call Run().
// ...
// Run runs all plugins and handlers and starts subscribing to provided topics.
// This call is blocking while the router is running.
//
// When all handlers have stopped (for example, because subscriptions were close
d), the router will also stop.
//
// To stop Run() you should call Close() on the router.
//
// ctx will be propagated to all subscribers.
//
// When all handlers are stopped (for example: because of closed connection), Ru
n() will be also stopped.
func (r *Router) Run(ctx context.Context) (err error) {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Ensuring that the Router is running#

   It can be useful to know if the router is running. You can use the
   Running() method for this.
// ...
// Running is closed when router is running.
// In other words: you can wait till router is running using
//
//      fmt.Println("Starting router")
//      go r.Run(ctx)
//      <- r.Running()
//      fmt.Println("Router is running")
//
// Warning: for historical reasons, this channel is not aware of router closing
- the channel will be closed if the router has been running and closed.
func (r *Router) Running() chan struct{} {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   You can also use IsRunning function, that returns bool:
// ...
// IsRunning returns true when router is running.
//
// Warning: for historical reasons, this method is not aware of router closing.
// If you want to know if the router was closed, use IsClosed.
func (r *Router) IsRunning() bool {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Closing the Router#

   To close the Router, you need to call Close().
// ...
// Close gracefully closes the router with a timeout provided in the configurati
on.
func (r *Router) Close() error {
        r.closedLock.Lock()
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   Close() will close all publishers and subscribers, and wait for all
   handlers to finish.

   Close() will wait for a timeout configured in
   RouterConfig.CloseTimeout. If the timeout is reached, Close() will
   return an error.

Adding handler after the router has started#

   You can add a new handler while the router is already running. To do
   that, you need to call AddNoPublisherHandler or AddHandler and call
   RunHandlers.
// ...
// RunHandlers runs all handlers that were added after Run().
// RunHandlers is idempotent, so can be called multiple times safely.
func (r *Router) RunHandlers(ctx context.Context) error {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Stopping running handler#

   It is possible to stop just one running handler by calling Stop().

   Please keep in mind, that router will be closed when there are no
   running handlers.
// ...
// Stop stops the handler.
// Stop is asynchronous.
// You can check if handler was stopped with Stopped() function.
func (h *Handler) Stop() {
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Execution models#

   Subscribers can consume either one message at a time or multiple
   messages in parallel.
     * Single stream of messages is the simplest approach and it means
       that until a msg.Ack() is called, the subscriber will not receive
       any new messages.
     * Multiple message streams are supported only by some subscribers. By
       subscribing to multiple topic partitions at once, several messages
       can be consumed in parallel, even previous messages that were not
       acked (for example, the Kafka subscriber works like this). Router
       handles this model by running concurrent HandlerFuncs, one for each
       partition.

   See the chosen Pub/Sub documentation for supported execution models.

Middleware#

// ...
// HandlerMiddleware allows us to write something like decorators to HandlerFunc
.
// It can execute something before handler (for example: modify consumed message
)
// or after (modify produced messages, ack/nack on consumed message, handle erro
rs, logging, etc.).
//
// It can be attached to the router by using `AddMiddleware` method.
//
// Example:
//
//      func ExampleMiddleware(h message.HandlerFunc) message.HandlerFunc {
//              return func(message *message.Message) ([]*message.Message, error
) {
//                      fmt.Println("executed before handler")
//                      producedMessages, err := h(message)
//                      fmt.Println("executed after handler")
//
//                      return producedMessages, err
//              }
//      }
type HandlerMiddleware func(h HandlerFunc) HandlerFunc

// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   A full list of standard middleware can be found in Middleware .

Plugin#

// ...
// RouterPlugin is function which is executed on Router start.
type RouterPlugin func(*Router) error

// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

   A full list of standard plugins can be found in message/router/plugin .

Context#

   Each message received by handler holds some useful values in the
   context:
// ...
// HandlerNameFromCtx returns the name of the message handler in the router that
 consumed the message.
func HandlerNameFromCtx(ctx context.Context) string {
        return valFromCtx(ctx, handlerNameKey)
}

// PublisherNameFromCtx returns the name of the message publisher type that publ
ished the message in the router.
// For example, for Kafka it will be `kafka.Publisher`.
func PublisherNameFromCtx(ctx context.Context) string {
        return valFromCtx(ctx, publisherNameKey)
}

// SubscriberNameFromCtx returns the name of the message subscriber type that su
bscribed to the message in the router.
// For example, for Kafka it will be `kafka.Subscriber`.
func SubscriberNameFromCtx(ctx context.Context) string {
        return valFromCtx(ctx, subscriberNameKey)
}

// SubscribeTopicFromCtx returns the topic from which message was received in th
e router.
func SubscribeTopicFromCtx(ctx context.Context) string {
        return valFromCtx(ctx, subscribeTopicKey)
}

// PublishTopicFromCtx returns the topic to which message will be published by t
he router.
func PublishTopicFromCtx(ctx context.Context) string {
        return valFromCtx(ctx, publishTopicKey)
}
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/message/router_context.go
   Help us improve this page
   Prev
   Pub/Sub
   Next
   Middleware

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/messages-router/index.txt ===

=== BEGIN: docs/middlewares/index.txt ===
On this page

     * Introduction
     * Usage
     * Available middleware
          + Instant Ack
          + Deduplicator
          + Ignore Errors
          + Circuit Breaker
          + Correlation
          + Retry
          + Throttle
          + Poison
          + Timeout
          + Delay On Error
          + Randomfail
          + Recoverer
          + Duplicator

Middleware

   On this page
     * Introduction
     * Usage
     * Available middleware
          + Instant Ack
          + Deduplicator
          + Ignore Errors
          + Circuit Breaker
          + Correlation
          + Retry
          + Throttle
          + Poison
          + Timeout
          + Delay On Error
          + Randomfail
          + Recoverer
          + Duplicator

Introduction#

   Middleware wrap handlers with functionality that is important, but not
   relevant for the primary handler’s logic. Examples include retrying the
   handler after an error was returned, or recovering from panic in the
   handler and capturing the stacktrace.

   Middleware wrap the handler function like this:
// ...
// HandlerMiddleware allows us to write something like decorators to HandlerFunc
.
// It can execute something before handler (for example: modify consumed message
)
// or after (modify produced messages, ack/nack on consumed message, handle erro
rs, logging, etc.).
//
// It can be attached to the router by using `AddMiddleware` method.
//
// Example:
//
//      func ExampleMiddleware(h message.HandlerFunc) message.HandlerFunc {
//              return func(message *message.Message) ([]*message.Message, error
) {
//                      fmt.Println("executed before handler")
//                      producedMessages, err := h(message)
//                      fmt.Println("executed after handler")
//
//                      return producedMessages, err
//              }
//      }
type HandlerMiddleware func(h HandlerFunc) HandlerFunc
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/router.go

Usage#

   Middleware can be executed for all as well as for a specific handler in
   a router. When middleware is added directly to a router it will be
   executed for all of handlers provided for a router. If a middleware
   should be executed only for a specific handler, it needs to be added to
   handler in the router.

   Example usage is shown below:
// ...
        router, err := message.NewRouter(message.RouterConfig{}, logger)
        if err != nil {
                panic(err)
        }

        // SignalsHandler will gracefully shutdown Router when SIGTERM is receiv
ed.
        // You can also close the router by just calling `r.Close()`.
        router.AddPlugin(plugin.SignalsHandler)

        // Router level middleware are executed for every message sent to the ro
uter
        router.AddMiddleware(
                // CorrelationID will copy the correlation id from the incoming
message's metadata to the produced messages
                middleware.CorrelationID,

                // The handler function is retried if it returns an error.
                // After MaxRetries, the message is Nacked and it's up to the Pu
bSub to resend it.
                middleware.Retry{
                        MaxRetries:      3,
                        InitialInterval: time.Millisecond * 100,
                        Logger:          logger,
                }.Middleware,

                // Recoverer handles panics from handlers.
                // In this case, it passes them as errors to the Retry middlewar
e.
                middleware.Recoverer,
        )

        // For simplicity, we are using the gochannel Pub/Sub here,
        // You can replace it with any Pub/Sub implementation, it will work the
same.
        pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)

        // Producing some incoming messages in background
        go publishMessages(pubSub)

        // AddHandler returns a handler which can be used to add handler level m
iddleware
        // or to stop handler.
        handler := router.AddHandler(
                "struct_handler",          // handler name, must be unique
                "incoming_messages_topic", // topic from which we will read even
ts
                pubSub,
                "outgoing_messages_topic", // topic to which we will publish eve
nts
                pubSub,
                structHandler{}.Handler,
        )

        // Handler level middleware is only executed for a specific handler
        // Such middleware can be added the same way the router level ones
        handler.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
                return func(message *message.Message) ([]*message.Message, error
) {
                        log.Println("executing handler specific middleware for "
, message.UUID)

                        return h(message)
                }
        })

        // just for debug, we are printing all messages received on `incoming_me
ssages_topic`
        router.AddNoPublisherHandler(
                "print_incoming_messages",
                "incoming_messages_topic",
                pubSub,
                printMessages,
        )

        // just for debug, we are printing all events sent to `outgoing_messages
_topic`
        router.AddNoPublisherHandler(
                "print_outgoing_messages",
                "outgoing_messages_topic",
                pubSub,
                printMessages,
        )

        // Now that all handlers are registered, we're running the Router.
        // Run is blocking while the router is running.
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/_examples/basic/3-router/main.go

Available middleware#

   Below are the middleware provided by Watermill and ready to use. You
   can also easily implement your own. For example, if you’d like to store
   every received message in some kind of log, it’s the best way to do it.

Instant Ack#

// InstantAck makes the handler instantly acknowledge the incoming message, rega
rdless of any errors.
// It may be used to gain throughput, but at a cost:
// If you had exactly-once delivery, you may expect at-least-once instead.
// If you had ordered messages, the ordering might be broken.
func InstantAck(h message.HandlerFunc) message.HandlerFunc {
        return func(message *message.Message) ([]*message.Message, error) {
                message.Ack()
                return h(message)
        }
}

Deduplicator#

// Deduplicator drops similar messages if they are present
// in a [ExpiringKeyRepository]. The similarity is determined
// by a [MessageHasher]. Time out is applied to repository
// operations using [context.WithTimeout].
//
// Call [Deduplicator.Middleware] for a new middleware
// or [Deduplicator.Decorator] for a [message.PublisherDecorator].
//
// KeyFactory defaults to [NewMessageHasherAdler32] with read
// limit  set to [math.MaxInt64] for fast tagging.
// Use [NewMessageHasherSHA256] for minimal collisions.
//
// Repository defaults to [NewMapExpiringKeyRepository] with one
// minute retention window. This default setting is performant
// but **does not support distributed operations**. If you
// implement a [ExpiringKeyRepository] backed by Redis,
// please submit a pull request.
//
// Timeout defaults to one minute. If lower than
// five milliseconds, it is set to five milliseconds.
//
// [ExpiringKeyRepository] must expire values
// in a certain time window. If there is no expiration, only one
// unique message will be ever delivered as long as the repository
// keeps its state.
type Deduplicator struct {
        KeyFactory MessageHasher
        Repository ExpiringKeyRepository
        Timeout    time.Duration
}

// IsDuplicate returns true if the message hash tag calculated
// using a [MessageHasher] was seen in deduplication time window.
func (d *Deduplicator) IsDuplicate(m *message.Message) (bool, error) {
        key, err := d.KeyFactory(m)
        if err != nil {
                return false, err
        }
        ctx, cancel := context.WithTimeout(m.Context(), d.Timeout)
        defer cancel()
        return d.Repository.IsDuplicate(ctx, key)
}

// Middleware returns the [message.HandlerMiddleware]
// that drops similar messages in a given time window.
func (d *Deduplicator) Middleware(h message.HandlerFunc) message.HandlerFunc {
        d = applyDefaultsToDeduplicator(d)
        return func(msg *message.Message) ([]*message.Message, error) {
                isDuplicate, err := d.IsDuplicate(msg)
                if err != nil {
                        return nil, err
                }
                if isDuplicate {
                        return nil, nil
                }
                return h(msg)
        }
}

// NewMapExpiringKeyRepository returns a memory store
// backed by a regular hash map protected by
// a [sync.Mutex]. The state **cannot be shared or synchronized
// between instances** by design for performance.
//
// If you need to drop duplicate messages by orchestration,
// implement [ExpiringKeyRepository] interface backed by Redis
// or similar.
//
// Window specifies the minimum duration of how long the
// duplicate tags are remembered for. Real duration can
// extend up to 50% longer because it depends on the
// clean up cycle.
func NewMapExpiringKeyRepository(window time.Duration) (ExpiringKeyRepository, e
rror) {
        if window < time.Millisecond {
                return nil, errors.New("deduplication window of less than a mill
isecond is impractical")
        }

        kr := &mapExpiringKeyRepository{
                window: window,
                mu:     &sync.Mutex{},
                tags:   make(map[string]time.Time),
        }
        go kr.cleanOutLoop(context.Background(), time.NewTicker(window/2))
        return kr, nil
}

// Len returns the number of known tags that have not been
// cleaned out yet.
func (kr *mapExpiringKeyRepository) Len() (count int) {
        kr.mu.Lock()
        count = len(kr.tags)
        kr.mu.Unlock()
        return
}

// NewMessageHasherAdler32 generates message hashes using a fast
// Adler-32 checksum of the [message.Message] body. Read
// limit specifies how many bytes of the message are
// used for calculating the hash.
//
// Lower limit improves performance but results in more false
// positives. Read limit must be greater than
// [MessageHasherReadLimitMinimum].
func NewMessageHasherAdler32(readLimit int64) MessageHasher {
        if readLimit < MessageHasherReadLimitMinimum {
                readLimit = MessageHasherReadLimitMinimum
        }
        return func(m *message.Message) (string, error) {
                h := adler32.New()
                _, err := io.CopyN(h, bytes.NewReader(m.Payload), readLimit)
                if err != nil && err != io.EOF {
                        return "", err
                }
                return string(h.Sum(nil)), nil
        }
}

// NewMessageHasherSHA256 generates message hashes using a slower
// but more resilient hashing of the [message.Message] body. Read
// limit specifies how many bytes of the message are
// used for calculating the hash.
//
// Lower limit improves performance but results in more false
// positives. Read limit must be greater than
// [MessageHasherReadLimitMinimum].
func NewMessageHasherSHA256(readLimit int64) MessageHasher {
        if readLimit < MessageHasherReadLimitMinimum {
                readLimit = MessageHasherReadLimitMinimum
        }

        return func(m *message.Message) (string, error) {
                h := sha256.New()
                _, err := io.CopyN(h, bytes.NewReader(m.Payload), readLimit)
                if err != nil && err != io.EOF {
                        return "", err
                }
                return string(h.Sum(nil)), nil
        }
}

// NewMessageHasherFromMetadataField looks for a hash value
// inside message metadata instead of calculating a new one.
// Useful if a [MessageHasher] was applied in a previous
// [message.HandlerFunc].
func NewMessageHasherFromMetadataField(field string) MessageHasher {
        return func(m *message.Message) (string, error) {
                fromMetadata, ok := m.Metadata[field]
                if ok {
                        return fromMetadata, nil
                }
                return "", fmt.Errorf("cannot recover hash value from metadata o
f message #%s: field %q is absent", m.UUID, field)
        }
}

// PublisherDecorator returns a decorator that
// acknowledges and drops every [message.Message] that
// was recognized by a [Deduplicator].
//
// The returned decorator provides the same functionality
// to a [message.Publisher] as [Deduplicator.Middleware]
// to a [message.Router].
func (d *Deduplicator) PublisherDecorator() message.PublisherDecorator {
        return func(pub message.Publisher) (message.Publisher, error) {
                if pub == nil {
                        return nil, errors.New("cannot decorate a <nil> publishe
r")
                }

                return &deduplicatingPublisherDecorator{
                        Publisher:    pub,
                        deduplicator: applyDefaultsToDeduplicator(d),
                }, nil
        }
}

Ignore Errors#

// IgnoreErrors provides a middleware that makes the handler ignore some explici
tly whitelisted errors.
type IgnoreErrors struct {
        ignoredErrors map[string]struct{}
}

// NewIgnoreErrors creates a new IgnoreErrors middleware.
func NewIgnoreErrors(errs []error) IgnoreErrors {
        errsMap := make(map[string]struct{}, len(errs))

        for _, err := range errs {
                errsMap[err.Error()] = struct{}{}
        }

        return IgnoreErrors{errsMap}
}

// Middleware returns the IgnoreErrors middleware.
func (i IgnoreErrors) Middleware(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
                events, err := h(msg)
                if err != nil {
                        if _, ok := i.ignoredErrors[errors.Cause(err).Error()];
ok {
                                return events, nil
                        }

                        return events, err
                }

                return events, nil
        }
}

Circuit Breaker#

// CircuitBreaker is a middleware that wraps the handler in a circuit breaker.
// Based on the configuration, the circuit breaker will fail fast if the handler
 keeps returning errors.
// This is useful for preventing cascading failures.
type CircuitBreaker struct {
        cb *gobreaker.CircuitBreaker
}

// NewCircuitBreaker returns a new CircuitBreaker middleware.
// Refer to the gobreaker documentation for the available settings.
func NewCircuitBreaker(settings gobreaker.Settings) CircuitBreaker {
        return CircuitBreaker{
                cb: gobreaker.NewCircuitBreaker(settings),
        }
}

// Middleware returns the CircuitBreaker middleware.
func (c CircuitBreaker) Middleware(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
                out, err := c.cb.Execute(func() (interface{}, error) {
                        return h(msg)
                })

                var result []*message.Message
                if out != nil {
                        result = out.([]*message.Message)
                }

                return result, err
        }
}

Correlation#

// SetCorrelationID sets a correlation ID for the message.
//
// SetCorrelationID should be called when the message enters the system.
// When message is produced in a request (for example HTTP),
// message correlation ID should be the same as the request's correlation ID.
func SetCorrelationID(id string, msg *message.Message) {
        if MessageCorrelationID(msg) != "" {
                return
        }

        msg.Metadata.Set(CorrelationIDMetadataKey, id)
}

// MessageCorrelationID returns correlation ID from the message.
func MessageCorrelationID(message *message.Message) string {
        return message.Metadata.Get(CorrelationIDMetadataKey)
}

// CorrelationID adds correlation ID to all messages produced by the handler.
// ID is based on ID from message received by handler.
//
// To make CorrelationID working correctly, SetCorrelationID must be called to f
irst message entering the system.
func CorrelationID(h message.HandlerFunc) message.HandlerFunc {
        return func(message *message.Message) ([]*message.Message, error) {
                producedMessages, err := h(message)

                correlationID := MessageCorrelationID(message)
                for _, msg := range producedMessages {
                        SetCorrelationID(correlationID, msg)
                }

                return producedMessages, err
        }
}

Retry#

// Retry provides a middleware that retries the handler if errors are returned.
// The retry behaviour is configurable, with exponential backoff and maximum ela
psed time.
type Retry struct {
        // MaxRetries is maximum number of times a retry will be attempted.
        MaxRetries int

        // InitialInterval is the first interval between retries. Subsequent int
ervals will be scaled by Multiplier.
        InitialInterval time.Duration
        // MaxInterval sets the limit for the exponential backoff of retries. Th
e interval will not be increased beyond MaxInterval.
        MaxInterval time.Duration
        // Multiplier is the factor by which the waiting interval will be multip
lied between retries.
        Multiplier float64
        // MaxElapsedTime sets the time limit of how long retries will be attemp
ted. Disabled if 0.
        MaxElapsedTime time.Duration
        // RandomizationFactor randomizes the spread of the backoff times within
 the interval of:
        // [currentInterval * (1 - randomization_factor), currentInterval * (1 +
 randomization_factor)].
        RandomizationFactor float64

        // OnRetryHook is an optional function that will be executed on each ret
ry attempt.
        // The number of the current retry is passed as retryNum,
        OnRetryHook func(retryNum int, delay time.Duration)

        Logger watermill.LoggerAdapter
}

// Middleware returns the Retry middleware.
func (r Retry) Middleware(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
                producedMessages, err := h(msg)
                if err == nil {
                        return producedMessages, nil
                }

                expBackoff := backoff.NewExponentialBackOff()
                expBackoff.InitialInterval = r.InitialInterval
                expBackoff.MaxInterval = r.MaxInterval
                expBackoff.Multiplier = r.Multiplier
                expBackoff.MaxElapsedTime = r.MaxElapsedTime
                expBackoff.RandomizationFactor = r.RandomizationFactor

                ctx := msg.Context()
                if r.MaxElapsedTime > 0 {
                        var cancel func()
                        ctx, cancel = context.WithTimeout(ctx, r.MaxElapsedTime)
                        defer cancel()
                }

                retryNum := 1
                expBackoff.Reset()
        retryLoop:
                for {
                        waitTime := expBackoff.NextBackOff()
                        select {
                        case <-ctx.Done():
                                return producedMessages, err
                        case <-time.After(waitTime):
                                // go on
                        }

                        producedMessages, err = h(msg)
                        if err == nil {
                                return producedMessages, nil
                        }

                        if r.Logger != nil {
                                r.Logger.Error("Error occurred, retrying", err,
watermill.LogFields{
                                        "retry_no":     retryNum,
                                        "max_retries":  r.MaxRetries,
                                        "wait_time":    waitTime,
                                        "elapsed_time": expBackoff.GetElapsedTim
e(),
                                })
                        }
                        if r.OnRetryHook != nil {
                                r.OnRetryHook(retryNum, waitTime)
                        }

                        retryNum++
                        if retryNum > r.MaxRetries {
                                break retryLoop
                        }
                }

                return nil, err
        }
}

Throttle#

// Throttle provides a middleware that limits the amount of messages processed p
er unit of time.
// This may be done e.g. to prevent excessive load caused by running a handler o
n a long queue of unprocessed messages.
type Throttle struct {
        ticker *time.Ticker
}

// NewThrottle creates a new Throttle middleware.
// Example duration and count: NewThrottle(10, time.Second) for 10 messages per
second
func NewThrottle(count int64, duration time.Duration) *Throttle {
        return &Throttle{
                ticker: time.NewTicker(duration / time.Duration(count)),
        }
}

// Middleware returns the Throttle middleware.
func (t Throttle) Middleware(h message.HandlerFunc) message.HandlerFunc {
        return func(message *message.Message) ([]*message.Message, error) {
                // throttle is shared by multiple handlers, which will wait for
their "tick"
                <-t.ticker.C

                return h(message)
        }
}

Poison#

// PoisonQueue provides a middleware that salvages unprocessable messages and pu
blished them on a separate topic.
// The main middleware chain then continues on, business as usual.
func PoisonQueue(pub message.Publisher, topic string) (message.HandlerMiddleware
, error) {
        if topic == "" {
                return nil, ErrInvalidPoisonQueueTopic
        }

        pq := poisonQueue{
                topic: topic,
                pub:   pub,
                shouldGoToPoisonQueue: func(err error) bool {
                        return true
                },
        }

        return pq.Middleware, nil
}

// PoisonQueueWithFilter is just like PoisonQueue, but accepts a function that d
ecides which errors qualify for the poison queue.
func PoisonQueueWithFilter(pub message.Publisher, topic string, shouldGoToPoison
Queue func(err error) bool) (message.HandlerMiddleware, error) {
        if topic == "" {
                return nil, ErrInvalidPoisonQueueTopic
        }

        pq := poisonQueue{
                topic: topic,
                pub:   pub,

                shouldGoToPoisonQueue: shouldGoToPoisonQueue,
        }

        return pq.Middleware, nil
}

Timeout#

// Timeout makes the handler cancel the incoming message's context after a speci
fied time.
// Any timeout-sensitive functionality of the handler should listen on msg.Conte
xt().Done() to know when to fail.
func Timeout(timeout time.Duration) func(message.HandlerFunc) message.HandlerFun
c {
        return func(h message.HandlerFunc) message.HandlerFunc {
                return func(msg *message.Message) ([]*message.Message, error) {
                        ctx, cancel := context.WithTimeout(msg.Context(), timeou
t)
                        defer func() {
                                cancel()
                        }()

                        msg.SetContext(ctx)
                        return h(msg)
                }
        }
}

Delay On Error#

// DelayOnError is a middleware that adds the delay metadata to the message if a
n error occurs.
//
// IMPORTANT: The delay metadata doesn't cause delays with all Pub/Subs! Using i
t won't have any effect on Pub/Subs that don't support it.
// See the list of supported Pub/Subs in the documentation: https://watermill.io
/advanced/delayed-messages/
type DelayOnError struct {
        // InitialInterval is the first interval between retries. Subsequent int
ervals will be scaled by Multiplier.
        InitialInterval time.Duration
        // MaxInterval sets the limit for the exponential backoff of retries. Th
e interval will not be increased beyond MaxInterval.
        MaxInterval time.Duration
        // Multiplier is the factor by which the waiting interval will be multip
lied between retries.
        Multiplier float64
}

Randomfail#

// RandomFail makes the handler fail with an error based on random chance. Error
 probability should be in the range (0,1).
func RandomFail(errorProbability float32) message.HandlerMiddleware {
        return func(h message.HandlerFunc) message.HandlerFunc {
                return func(message *message.Message) ([]*message.Message, error
) {
                        if shouldFail(errorProbability) {
                                return nil, errors.New("random fail occurred")
                        }

                        return h(message)
                }
        }
}

// RandomPanic makes the handler panic based on random chance. Panic probability
 should be in the range (0,1).
func RandomPanic(panicProbability float32) message.HandlerMiddleware {
        return func(h message.HandlerFunc) message.HandlerFunc {
                return func(message *message.Message) ([]*message.Message, error
) {
                        if shouldFail(panicProbability) {
                                panic("random panic occurred")
                        }

                        return h(message)
                }
        }
}

Recoverer#

// RecoveredPanicError holds the recovered panic's error along with the stacktra
ce.
type RecoveredPanicError struct {
        V          interface{}
        Stacktrace string
}

// Recoverer recovers from any panic in the handler and appends RecoveredPanicEr
ror with the stacktrace
// to any error returned from the handler.
func Recoverer(h message.HandlerFunc) message.HandlerFunc {
        return func(event *message.Message) (events []*message.Message, err erro
r) {
                panicked := true

                defer func() {
                        if r := recover(); r != nil || panicked {
                                err = errors.WithStack(RecoveredPanicError{V: r,
 Stacktrace: string(debug.Stack())})
                        }
                }()

                events, err = h(event)
                panicked = false
                return events, err
        }
}

Duplicator#

// Duplicator is processing messages twice, to ensure that the endpoint is idemp
otent.
func Duplicator(h message.HandlerFunc) message.HandlerFunc {
        return func(msg *message.Message) ([]*message.Message, error) {
                firstProducedMessages, firstErr := h(msg)
                if firstErr != nil {
                        return nil, firstErr
                }

                secondProducedMessages, secondErr := h(msg)
                if secondErr != nil {
                        return nil, secondErr
                }

                return append(firstProducedMessages, secondProducedMessages...),
 nil
        }
}

   Help us improve this page
   Prev
   Router
   Next
   CQRS Component

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/middlewares/index.txt ===

=== BEGIN: docs/pub-sub/index.txt ===
On this page

     * Publisher
          + Publishing multiple messages
          + Async publish
     * Subscriber
          + Ack/Nack mechanism
     * At-least-once delivery
     * Universal tests
     * Built-in implementations
     * Implementing custom Pub/Sub

Pub/Sub

   On this page
     * Publisher
          + Publishing multiple messages
          + Async publish
     * Subscriber
          + Ack/Nack mechanism
     * At-least-once delivery
     * Universal tests
     * Built-in implementations
     * Implementing custom Pub/Sub

Publisher#

// ...
type Publisher interface {
        // Publish publishes provided messages to the given topic.
        //
        // Publish can be synchronous or asynchronous - it depends on the implem
entation.
        //
        // Most publisher implementations don't support atomic publishing of mes
sages.
        // This means that if publishing one of the messages fails, the next mes
sages will not be published.
        //
        // Publish does not work with a single Context.
        // Use the Context() method of each message instead.
        //
        // Publish must be thread safe.
        Publish(topic string, messages ...*Message) error
        // Close should flush unsent messages if publisher is async.
        Close() error
}
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/pubsub.go

Publishing multiple messages#

   Most publishers implementations don’t support atomic publishing of
   messages. This means that if publishing one of the messages fails, the
   next messages won’t be published.

Async publish#

   Publish can be synchronous or asynchronous - it depends on the
   implementation.

Close()#

   Close should flush unsent messages if the publisher is asynchronous. It
   is important to not forget to close the subscriber. Otherwise you may
   lose some of the messages.

Subscriber#

// ...
type Subscriber interface {
        // Subscribe returns an output channel with messages from the provided t
opic.
        // The channel is closed after Close() is called on the subscriber.
        //
        // To receive the next message, `Ack()` must be called on the received m
essage.
        // If message processing fails and the message should be redelivered `Na
ck()` should be called instead.
        //
        // When the provided ctx is canceled, the subscriber closes the subscrip
tion and the output channel.
        // The provided ctx is passed to all produced messages.
        // When Nack or Ack is called on the message, the context of the message
 is canceled.
        Subscribe(ctx context.Context, topic string) (<-chan *Message, error)
        // Close closes all subscriptions with their output channels and flushes
 offsets etc. when needed.
        Close() error
}
// ...

   Full source: github.com/ThreeDotsLabs/watermill/message/pubsub.go

Ack/Nack mechanism#

   It is the Subscriber’s responsibility to handle an Ack and a Nack from
   a message. A proper implementation should wait for an Ack or a Nack
   before consuming the next message.

   Important Subscriber’s implementation notice: Ack/offset to message’s
   storage/broker must be sent after Ack from Watermill’s message.
   Otherwise there is a chance to lose messages if the process dies before
   the messages have been processed.

Close()#

   Close closes all subscriptions with their output channels and flushes
   offsets, etc. when needed.

At-least-once delivery#

   Watermill is built with at-least-once delivery semantics. That means
   when some error occurs when processing a message and an Ack cannot be
   sent, the message will be redelivered.

   You need to keep it in mind and build your application to be idempotent
   or implement a deduplication mechanism.

   Unfortunately, it’s not possible to create an universal middleware for
   deduplication, so we encourage you to build your own.

Universal tests#

   Every Pub/Sub is similar in most aspects. To avoid implementing
   separate tests for every Pub/Sub, we’ve created a test suite which
   should be passed by any Pub/Sub implementation.

   These tests can be found in pubsub/tests/test_pubsub.go.

Built-in implementations#

   To check available Pub/Sub implementations, see Supported Pub/Subs .

Implementing custom Pub/Sub#

   See Implementing custom Pub/Sub for instructions on how to introduce
   support for a new Pub/Sub.

   We will also be thankful for submitting pull requests with the new
   Pub/Sub implementations.

   You can also request a new Pub/Sub implementation by submitting a new
   issue .
   Help us improve this page
   Prev
   Message
   Next
   Router

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/pub-sub/index.txt ===

=== BEGIN: docs/troubleshooting/index.txt ===
On this page

     * Logging
     * Debugging Pub/Sub tests
          + Running a single test
          + grep is your friend
     * I have a deadlock

Troubleshooting

   On this page
     * Logging
     * Debugging Pub/Sub tests
          + Running a single test
          + grep is your friend
     * I have a deadlock

Logging#

   In most cases, you will find the answer to your problem in the logs.
   Watermill offers a significant amount of logs on different severity
   levels.

   If you are using StdLoggerAdapter, just change debug, and trace options
   to true:
logger := watermill.NewStdLogger(true, true)

Debugging Pub/Sub tests#

Running a single test#

make up
go test -v ./... -run TestPublishSubscribe/TestContinueAfterSubscribeClose

grep is your friend#

   Each executed test case has a unique UUID. It’s used in the topic’s
   name. Thanks to that, you can easily grep the output of the test. It
   gives you detailed information about the test execution.
> go test -v ./... > test.out

> less test.out

// ...

--- PASS: TestPublishSubscribe (0.00s)
    --- PASS: TestPublishSubscribe/TestPublishSubscribe (2.38s)
        --- PASS: TestPublishSubscribe/TestPublishSubscribe/81eeb56c-3336-4eb9-a
0ac-13abda6f38ff (2.38s)

cat test.out | grep 81eeb56c-3336-4eb9-a0ac-13abda6f38ff | less

[watermill] 2020/08/18 14:51:46.283366 subscriber.go:300:       level=TRACE msg=
"Msg acked" message_uuid=5c920330-5075-4870-8d86-9013771eee78 provider=google_cl
oud_pubsub subscription_name=topic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff topic=to
pic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff
[watermill] 2020/08/18 14:51:46.283405 subscriber.go:300:       level=TRACE msg=
"Msg acked" message_uuid=46e04a08-994e-4c04-afff-7fd42fd67f95 provider=google_cl
oud_pubsub subscription_name=topic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff topic=to
pic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff
2020/08/18 14:51:46 all messages (100/100) received in bulk read after 110.04155
ms of 45s (test ID: 81eeb56c-3336-4eb9-a0ac-13abda6f38ff)
[watermill] 2020/08/18 14:51:46.284569 subscriber.go:186:       level=DEBUG msg=
"Closing message consumer" provider=google_cloud_pubsub subscription_name=topic_
81eeb56c-3336-4eb9-a0ac-13abda6f38ff topic=topic_81eeb56c-3336-4eb9-a0ac-13abda6
f38ff
[watermill] 2020/08/18 14:51:46.284828 subscriber.go:300:       level=TRACE msg=
"Msg acked" message_uuid=2f409208-d4d2-46f6-b6b9-afb1aea0e59f provider=google_cl
oud_pubsub subscription_name=topic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff topic=to
pic_81eeb56c-3336-4eb9-a0ac-13abda6f38ff
        --- PASS: TestPublishSubscribe/TestPublishSubscribe/81eeb56c-3336-4eb9-a
0ac-13abda6f38ff (2.38s)

I have a deadlock#

   When running locally, you can send a SIGQUIT to the running process:
     * CTRL + \ on Linux
     * kill -s SIGQUIT [pid] on other UNIX systems

   This will kill the process and print all goroutines along with lines on
   which they have stopped.
SIGQUIT: quit
PC=0x45e7c3 m=0 sigcode=128

goroutine 1 [runnable]:
github.com/ThreeDotsLabs/watermill/pubsub/gochannel.(*GoChannel).sendMessage(0xc
000024100, 0x7c5250, 0xd, 0xc000872d70, 0x0, 0x0)
        /home/example/go/src/github.com/ThreeDotsLabs/watermill/pubsub/gochannel
/pubsub.go:83 +0x36a
github.com/ThreeDotsLabs/watermill/pubsub/gochannel.(*GoChannel).Publish(0xc0000
24100, 0x7c5250, 0xd, 0xc000098530, 0x1, 0x1, 0x0, 0x0)
        /home/example/go/src/github.com/ThreeDotsLabs/watermill/pubsub/gochannel
/pubsub.go:53 +0x6d
main.publishMessages(0x7fdf7a317000, 0xc000024100)
        /home/example/go/src/github.com/ThreeDotsLabs/watermill/docs/src-link/_e
xamples/pubsubs/go-channel/main.go:43 +0x1ec
main.main()
        /home/example/go/src/github.com/ThreeDotsLabs/watermill/docs/src-link/_e
xamples/pubsubs/go-channel/main.go:36 +0x20a

// ...

   When running in production and you don’t want to kill the entire
   process, a better idea is to use pprof .

   You can visit http://localhost:6060/debug/pprof/goroutine?debug=1 on
   your local machine to see all goroutines status.
goroutine profile: total 5
1 @ 0x41024c 0x6a8311 0x6a9bcb 0x6a948d 0x7028bc 0x70260a 0x42f187 0x45c971
#       0x6a8310        github.com/ThreeDotsLabs/watermill.LogFields.Add+0xd0
                                        /home/example/go/src/github.com/ThreeDot
sLabs/watermill/log.go:15
#       0x6a9bca        github.com/ThreeDotsLabs/watermill/pubsub/gochannel.(*Go
Channel).sendMessage+0x6fa      /home/example/go/src/github.com/ThreeDotsLabs/wa
termill/pubsub/gochannel/pubsub.go:75
#       0x6a948c        github.com/ThreeDotsLabs/watermill/pubsub/gochannel.(*Go
Channel).Publish+0x6c           /home/example/go/src/github.com/ThreeDotsLabs/wa
termill/pubsub/gochannel/pubsub.go:53
#       0x7028bb        main.publishMessages+0x1eb
                                        /home/example/go/src/github.com/ThreeDot
sLabs/watermill/docs/src-link/_examples/pubsubs/go-channel/main.go:43
#       0x702609        main.main+0x209
                                        /home/example/go/src/github.com/ThreeDot
sLabs/watermill/docs/src-link/_examples/pubsubs/go-channel/main.go:36
#       0x42f186        runtime.main+0x206
                                        /usr/lib/go/src/runtime/proc.go:201

// ...

   Help us improve this page
   Prev
   CQRS Component
   Next
   Articles

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: docs/troubleshooting/index.txt ===

=== BEGIN: pubsubs/gochannel/index.txt ===
On this page

     *
          + Characteristics
          + Configuration
          + Publishing
          + Subscribing
          + Marshaler

Go Channel

   On this page
     *
          + Characteristics
          + Configuration
          + Publishing
          + Subscribing
          + Marshaler

// ...
// GoChannel is the simplest Pub/Sub implementation.
// It is based on Golang's channels which are sent within the process.
//
// GoChannel has no global state,
// that means that you need to use the same instance for Publishing and Subscrib
ing!
//
// When GoChannel is persistent, messages order is not guaranteed.
type GoChannel struct {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/pubsub/gochannel/pubsub.go

   You can find a fully functional example with Go Channels in the
   Watermill examples .

Characteristics#

         Feature       Implements Note
   ConsumerGroups      no
   ExactlyOnceDelivery yes
   GuaranteedOrder     yes
   Persistent          no

Configuration#

   You can inject configuration via the constructor.
// ...
func NewGoChannel(config Config, logger watermill.LoggerAdapter) *GoChannel {
        if logger == nil {
                logger = watermill.NopLogger{}
        }

        return &GoChannel{
                config: config,

                subscribers:            make(map[string][]*subscriber),
                subscribersByTopicLock: sync.Map{},
                logger: logger.With(watermill.LogFields{
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/pubsub/gochannel/pubsub.go

Publishing#

// ...
// Publish in GoChannel is NOT blocking until all consumers consume.
// Messages will be send in background.
//
// Messages may be persisted or not, depending of persistent attribute.
func (g *GoChannel) Publish(topic string, messages ...*message.Message) error {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/pubsub/gochannel/pubsub.go

Subscribing#

// ...
// Subscribe returns channel to which all published messages are sent.
// Messages are not persisted. If there are no subscribers and message is produc
ed it will be gone.
//
// There are no consumer groups support etc. Every consumer will receive every p
roduced message.
func (g *GoChannel) Subscribe(ctx context.Context, topic string) (<-chan *messag
e.Message, error) {
// ...

   Full source:
   github.com/ThreeDotsLabs/watermill/pubsub/gochannel/pubsub.go

Marshaler#

   No marshaling is needed when sending messages within the process.
   Help us improve this page
   Prev
   Firestore Pub/Sub
   Next
   Google Cloud Pub/Sub

   [event-driven-banner.png]
   Check our online hands-on training

   Three Dots Labs Three Dots Labs

   © Three Dots Labs 2014 — 2024

   Watermill is open-source software and is not backed by venture capital.
   We are an independent, bootstrapped company.

=== END: pubsubs/gochannel/index.txt ===


