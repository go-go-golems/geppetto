const gp = require("geppetto");

let profilesErr = false;
try {
  gp.profiles.listRegistries();
} catch (e) {
  profilesErr = /configured profile registry/i.test(String(e));
}
assert(profilesErr, "profiles namespace should require configured profile registry");

let fromProfileErr = false;
try {
  gp.engines.fromProfile("assistant");
} catch (e) {
  fromProfileErr = /configured profile registry/i.test(String(e));
}
assert(fromProfileErr, "engines.fromProfile should require configured profile registry");

console.log("missing profile registry errors: PASS");
