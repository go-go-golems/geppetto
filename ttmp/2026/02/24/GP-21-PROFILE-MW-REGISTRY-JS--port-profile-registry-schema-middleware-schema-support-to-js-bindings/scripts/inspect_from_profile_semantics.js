const gp = require("geppetto");

const explicit = gp.engines.fromProfile("explicit-model", {
  profile: "opts-model",
  apiType: "openai",
  apiKey: "test-openai-key",
});

const optsProfile = gp.engines.fromProfile(undefined, {
  profile: "opts-model",
  apiType: "openai",
  apiKey: "test-openai-key",
});

const envProfile = gp.engines.fromProfile(undefined, {
  apiType: "openai",
  apiKey: "test-openai-key",
});

const fromConfig = gp.engines.fromConfig({
  apiType: "openai",
  model: "gpt-4o-mini",
  apiKey: "test-openai-key",
});

console.log({
  explicitName: explicit.name,
  optsName: optsProfile.name,
  envName: envProfile.name,
  fromConfigName: fromConfig.name,
});
