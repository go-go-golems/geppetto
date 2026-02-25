const plugins = require("geppetto/plugins");

const report = {
  keys: Object.keys(plugins || {}).sort(),
  apiVersion: plugins && plugins.EXTRACTOR_PLUGIN_API_VERSION,
};

console.log(JSON.stringify(report, null, 2));
