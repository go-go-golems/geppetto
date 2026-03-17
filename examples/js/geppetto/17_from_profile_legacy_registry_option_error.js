const gp = require("geppetto");

let threw = false;
try {
  gp.engines.fromProfile("assistant");
} catch (e) {
  threw = /engines\.fromprofile has been removed/i.test(String(e));
}

assert(threw, "engines.fromProfile should throw hard-cut removal error");
console.log("engines.fromProfile removal error: PASS");
