# Tasks

## TODO

- [x] Confirm module package layout and JS entrypoint (`require("geppetto")`) responsibilities
- [x] Define JS turn/block canonical object shapes (required vs optional fields)
- [x] Implement decode algorithm spec (JS -> Go `turns.Turn`) with fallback rules
- [x] Implement encode algorithm spec (Go `turns.Turn` -> JS) with stable field ordering policy
- [x] Bind decode/encode key lookups to generated key mappers in `pkg/turns/keys_gen.go`
- [x] Bind block kind conversion to generated kind mapper in `pkg/turns/block_kind_gen.go`
- [x] Document unknown key/kind handling (strict required keys, tolerant optional passthrough)
- [x] Add JS contract test script for turn/block mapper invariants under ticket `scripts/`
- [x] Run JS mapper script and capture output in diary
- [x] Update changelog with completed setup+turn mapping baseline
