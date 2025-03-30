What I want to do as part of this refactor:

- rename publishermanager as EventBroadcaster. It is actually of no relevance to the user of geppetto, it's internal to steps

The goal is to refactor handling streaming events from the chat step in a more typed fashion. 

Here's how we could introduce it to @client.go:

- [x] helper function to register a chateventhandler/callback instead of having to register a topic and a watermill router by myself
  - [x] helper function (later maybe a method on EventRouter)
     - [x] RegisterChatHandler(step, id, chatEventHandler)
       - [x] step.AddPublishedTopic(chat-%s, id)
       - [x] router AddHandler(chat-%s, id, func message() { switch message type and then: dispatchToHandler })
     - [x] chatEventHandler: an interface that handles  the chat messages HandleXMessage etc...
       - [x] concrete implementation for chatClient in our case

- RegisterChatHandler could maybe be called func (c *ChatClient) Start(ctx)  err { 
       err = c.registerChatHandler(ctx)
      RunHandlers
   }
  - registerChatHandler shoudl be able to return an error, so that Start also handles errors

- move the callback function in registerHandler out to something reusable

- move all that chatEventHandler and callback and registering to the router entirely to @event-router.go so that all we do is  call
eventRouter.RegisterChatEventHandler(chatClient)