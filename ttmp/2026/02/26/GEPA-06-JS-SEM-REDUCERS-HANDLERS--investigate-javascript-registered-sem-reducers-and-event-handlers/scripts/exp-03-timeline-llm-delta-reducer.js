onSem("llm.delta", function(ev) {
  if (!ev || !ev.id) {
    return;
  }
});

registerSemReducer("llm.delta", function(ev) {
  return {
    consume: false,
    upserts: [
      {
        id: ev.id + "-delta-projection",
        kind: "llm.delta.projection",
        props: {
          cumulative: ev.data && ev.data.cumulative,
          delta: ev.data && ev.data.delta
        }
      }
    ]
  };
});
