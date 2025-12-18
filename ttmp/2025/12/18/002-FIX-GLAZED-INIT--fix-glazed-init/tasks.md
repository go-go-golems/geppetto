# Tasks

## Completed

- [x] Replace `clay.InitViper` with `clay.InitGlazed` in all example files
- [x] Replace `logging.InitLoggerFromViper()` with `logging.InitLoggerFromCobra(cmd)` in all example files
- [x] Replace `AddNoPublisherHandler` with `AddConsumerHandler` in event-router.go
- [x] Replace `middlewares.GatherFlagsFromViper` with `LoadParametersFromFiles` and `UpdateFromEnv` in layers.go

## Files Modified

1. `cmd/examples/claude-tools/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
2. `cmd/examples/generic-tool-calling/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
3. `cmd/examples/simple-inference/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
4. `cmd/examples/middleware-inference/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
5. `cmd/examples/openai-tools/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
6. `cmd/examples/simple-streaming-inference/main.go` - Updated InitViper → InitGlazed, InitLoggerFromViper → InitLoggerFromCobra
7. `cmd/llm-runner/main.go` - Updated InitViper → InitGlazed
8. `pkg/events/event-router.go` - Updated AddNoPublisherHandler → AddConsumerHandler
9. `pkg/layers/layers.go` - Replaced GatherFlagsFromViper with LoadParametersFromFiles + UpdateFromEnv using ResolveAppConfigPath

## Migration Details

Following the glaze migration guide (`glaze help migrating-from-viper-to-config-files`):

1. **Config File Loading**: Replaced `GatherFlagsFromViper()` with `LoadParametersFromFile()` using `ResolveAppConfigPath()` for automatic config discovery
2. **Environment Variables**: Added `UpdateFromEnv("PINOCCHIO")` middleware for environment variable support
3. **Logging Initialization**: Moved from `InitLoggerFromViper()` to `InitLoggerFromCobra(cmd)` in PersistentPreRunE handlers
4. **Clay Initialization**: Replaced `clay.InitViper()` with `clay.InitGlazed()` for proper Glazed setup

## Verification

All deprecated Viper-related lint errors (SA1019) have been resolved. Verified with:
```bash
golangci-lint run --disable-all --enable=staticcheck | grep -E "(SA1019|clay.InitViper|GatherFlagsFromViper|AddNoPublisherHandler|InitLoggerFromViper)"
# Result: No deprecated Viper issues found!
```
