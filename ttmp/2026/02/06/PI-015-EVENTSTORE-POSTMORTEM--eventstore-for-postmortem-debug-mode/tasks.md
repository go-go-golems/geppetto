# Tasks

## Discovery and contract

- [ ] Confirm required postmortem workflows and query filters with debug UI stakeholders
- [ ] Finalize EventStore schema and retention policy defaults

## Backend implementation

- [ ] Implement `EventStore` interface and SQLite backend
- [ ] Add async ingestion queue and batch writer from SEM envelope stream
- [ ] Add history/export/retention endpoints under `/debug/events/history/*`

## Validation and operations

- [ ] Add integration tests for persistence/restart and ordered retrieval
- [ ] Add load tests for ingestion throughput and backpressure behavior
- [ ] Add metrics and operational notes (queue depth, dropped events, DB growth)
