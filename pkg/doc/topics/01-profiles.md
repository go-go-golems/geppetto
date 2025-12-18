---
Title: Using profiles in Pinocchio
Slug: profiles
Short: |
  Configure and use different profiles in Pinocchio to override layer parameters.
Topics:
- configuration
Commands:
- pinocchio
Flags:
- profile
IsTopLevel: true
ShowPerDefault: true
SectionType: GeneralTopic
---

# Profile Configuration in Pinocchio

Pinocchio allows users to override layer parameters using profiles. This
functionality makes it easy to switch between different configurations (for example,
different api keys or url endpoints).

For example, this allows us to use the openai api-type with different urls in order to use ollama or anyscale, for
example.

## Configuring Profiles

Profiles are defined in a YAML configuration file, typically located at `~/.config/pinocchio/profiles.yaml` on Linux
systems (or the equivalent path on macOS). Each profile specifies a set of parameters that can be used to override the
default settings.

Here's an example `profiles.yaml` file:

```yaml
mixtral:
  openai-chat:
    openai-base-url: https://api.endpoints.anyscale.com/v1
    openai-api-key: XXX
  ai-chat:
    ai-engine: mistralai/Mixtral-8x7B-Instruct-v0.1
    ai-api-type: openai

mistral:
  openai-chat:
    openai-base-url: https://api.endpoints.anyscale.com/v1
    openai-api-key: XXX
  ai-chat:
    ai-engine: mistralai/Mistral-7B-Instruct-v0.1
    ai-api-type: openai

zephir:
  openai-chat:
    openai-base-url: https://api.endpoints.anyscale.com/v1
    openai-api-key: XXX
  ai-chat:
    ai-engine: HuggingFaceH4/zephyr-7b-beta
    ai-api-type: openai
```

## Selecting a Profile

To select a profile for use, you can:

- set environment variables
- pass flags (`--profile`, `--profile-file`)
- set the values in the Pinocchio config file (`config.yaml`)

Pinocchio’s precedence (highest → lowest) is:

**flags > env > config > profiles > defaults**

### Using the Environment Variable

```bash
export PINOCCHIO_PROFILE=mistral
export PINOCCHIO_PROFILE_FILE=~/.config/pinocchio/profiles.yaml  # optional override
pinocchio [command]
```

### Using the Command Line Flag

```bash
pinocchio --profile mistral [command]
```

### Setting in `config.yaml`

Pinocchio loads config files from the first existing path in this search order:

1. `$XDG_CONFIG_HOME/pinocchio/config.yaml` (via `os.UserConfigDir()`, commonly `~/.config/pinocchio/config.yaml`)
2. `~/.pinocchio/config.yaml`
3. `/etc/pinocchio/config.yaml`

To set profile selection via config, you must set values under the `profile-settings` layer:

```yaml
profile-settings:
  profile: mistral
  # profile-file: /etc/pinocchio/profiles.yaml  # optional override
```

After setting the desired profile, Pinocchio will use the parameters defined within that profile for all operations.

## Debugging (recommended)

Use `--print-parsed-parameters` to see exactly where each parameter value came from (defaults vs profiles vs config vs env vs flags).

