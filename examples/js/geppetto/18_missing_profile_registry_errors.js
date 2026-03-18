const gp = require("geppetto");

let profilesErr = false;
try {
  gp.profiles.listRegistries();
} catch (e) {
  profilesErr = /configured profile registry/i.test(String(e));
}
assert(profilesErr, "profiles namespace should require configured profile registry");

console.log("missing profile registry errors: PASS");
