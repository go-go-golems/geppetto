#!/usr/bin/env node

/**
 * Prototype for GEPA-06:
 * - Demonstrates current single-handler-per-type semantics (overwrite risk)
 * - Demonstrates a composable reducer/handler model with wildcard support
 */

function createCurrentRegistry() {
  const handlers = new Map();
  return {
    registerSem(type, handler) {
      handlers.set(type, handler);
    },
    handleSem(event, ctx) {
      const handler = handlers.get(event.type);
      if (!handler) return;
      handler(event, ctx);
    },
  };
}

function createProposedRuntime() {
  const reducersByType = new Map();
  const handlersByType = new Map();
  const wildcardHandlers = [];

  function pushReducer(type, reducer) {
    const list = reducersByType.get(type) || [];
    list.push(reducer);
    reducersByType.set(type, list);
  }

  function pushHandler(type, handler) {
    const list = handlersByType.get(type) || [];
    list.push(handler);
    handlersByType.set(type, list);
  }

  return {
    registerReducer(type, reducer) {
      pushReducer(type, reducer);
    },
    registerHandler(type, handler) {
      if (type === '*') {
        wildcardHandlers.push(handler);
        return;
      }
      pushHandler(type, handler);
    },
    dispatch(event, state) {
      const reducers = reducersByType.get(event.type) || [];
      const nextState = reducers.reduce((acc, reducer) => reducer(acc, event), state);
      const handlers = (handlersByType.get(event.type) || []).concat(wildcardHandlers);
      for (const handler of handlers) {
        handler(event, nextState);
      }
      return nextState;
    },
  };
}

function runCurrentModelDemo() {
  const registry = createCurrentRegistry();
  const ctx = { stream: [] };

  registry.registerSem('llm.delta', (event, context) => {
    context.stream.push(`default:${event.data.cumulative}`);
  });

  registry.registerSem('llm.delta', (event, context) => {
    context.stream.push(`custom:${event.data.cumulative}`);
  });

  registry.handleSem({ type: 'llm.delta', data: { cumulative: 'hello' } }, ctx);
  return ctx.stream;
}

function runProposedModelDemo() {
  const runtime = createProposedRuntime();

  runtime.registerReducer('llm.delta', (state, event) => ({
    ...state,
    tokenCount: state.tokenCount + 1,
    cumulative: event.data.cumulative,
  }));

  runtime.registerReducer('llm.delta', (state) => ({
    ...state,
    lastUpdatedBy: 'llm.delta.reducer',
  }));

  runtime.registerHandler('llm.delta', (event, state) => {
    console.log(`handler(llm.delta): cumulative=${state.cumulative} tokens=${state.tokenCount}`);
  });

  runtime.registerHandler('*', (event, state) => {
    console.log(`handler(*): type=${event.type} state.lastUpdatedBy=${state.lastUpdatedBy || 'n/a'}`);
  });

  let state = { tokenCount: 0, cumulative: '' };
  state = runtime.dispatch({ type: 'llm.delta', data: { cumulative: 'h' } }, state);
  state = runtime.dispatch({ type: 'llm.delta', data: { cumulative: 'he' } }, state);
  state = runtime.dispatch({ type: 'tool.result', data: { ok: true } }, state);
  return state;
}

console.log('--- current single-handler map demo ---');
const currentResult = runCurrentModelDemo();
console.log(JSON.stringify(currentResult));

console.log('--- proposed composable reducer+handler demo ---');
const finalState = runProposedModelDemo();
console.log(JSON.stringify(finalState));
