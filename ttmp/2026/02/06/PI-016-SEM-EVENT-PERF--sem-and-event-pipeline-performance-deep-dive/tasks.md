# Tasks

## Baseline and instrumentation

- [ ] Add pipeline performance metrics (consume latency, projector latency, buffer trims, queue depth)
- [ ] Create reproducible benchmark/load harness for SEM/event path
- [ ] Produce baseline performance report and store in ticket

## Optimization work

- [ ] Implement sem buffer ring-buffer replacement (remove overflow copy-trim churn)
- [ ] Optimize SEM envelope cursor enrichment path to reduce marshal/unmarshal overhead
- [ ] Prototype decoupled projector persistence worker with ordering guarantees

## Validation and rollout

- [ ] Run before/after benchmarks and document deltas
- [ ] Add ordering/consistency regression tests for optimized pipeline
- [ ] Add rollout plan with feature flags and fallback strategy
