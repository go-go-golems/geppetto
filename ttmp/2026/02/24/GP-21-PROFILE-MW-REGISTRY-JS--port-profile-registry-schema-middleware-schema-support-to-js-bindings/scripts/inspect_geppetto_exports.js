const gp = require("geppetto");

function sortedKeys(obj) {
  return Object.keys(obj || {}).sort();
}

const report = {
  topLevel: sortedKeys(gp),
  engines: sortedKeys(gp.engines),
  middlewares: sortedKeys(gp.middlewares),
  tools: sortedKeys(gp.tools),
  hasProfilesNamespace: Object.prototype.hasOwnProperty.call(gp, "profiles"),
  hasSchemasNamespace: Object.prototype.hasOwnProperty.call(gp, "schemas"),
  hasPluginsModuleTopLevel: Object.prototype.hasOwnProperty.call(gp, "plugins"),
};

console.log(JSON.stringify(report, null, 2));
