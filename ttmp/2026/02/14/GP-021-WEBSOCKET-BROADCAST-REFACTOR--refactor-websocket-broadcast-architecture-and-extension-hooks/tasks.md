# Tasks

## TODO

- [ ] Introduce `ConversationWSBroker` abstraction and transport adapter over `ConnectionPool`
- [ ] Route existing SEM stream fanout through broker publish API
- [ ] Route timeline upsert emission through broker publish API
- [ ] Add connection subscription model (`ws_profile`, `channels`) parsed at `/ws` connect
- [ ] Add channel classification rules for outgoing frames (`sem`, `timeline`, `control`, future debug channels)
- [ ] Add broker-level filtering by subscription profile/channel
- [ ] Add router option for backend websocket emitter factory registration
- [ ] Add default compatibility profile matching current behavior
- [ ] Add unit and integration tests for profile-filtered fanout and compatibility
- [ ] Add observability counters/log fields for publish/deliver/drop paths
- [ ] Design and gate optional `debug.turn_snapshot` channel as follow-up implementation
- [ ] Update developer docs for websocket protocol and extension contracts
