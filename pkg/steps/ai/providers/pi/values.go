package pi

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// Pi's configured values (models.json apiKey and headers, auth.json keys):
//
//	"!cmd"            run cmd through the shell and use its trimmed stdout
//	"$NAME", "${NAME}" the environment variable (the credential's env map first)
//	"$$", "$!"        a literal "$" or "!"
//	anything else     a literal
//
// This mirrors pi-coding-agent/dist/core/resolve-config-value.js.

var envNamePrefix = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*`)

// commandCache keeps command results for the process lifetime, as Pi does.
var commandCache sync.Map

func (a *AgentDir) resolveValue(ctx context.Context, config string, env map[string]string) (string, error) {
	if strings.HasPrefix(config, "!") {
		if !a.opts.allowCommands {
			return "", errors.New("a command value needs pi.WithCommandValues()")
		}
		if v, ok := commandCache.Load(config); ok {
			return v.(string), nil
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, "sh", "-c", config[1:]).Output() // #nosec G204 -- the user's own Pi config
		if err != nil {
			return "", errors.New("the configured command failed") // never echo the command: it may embed a secret
		}
		v := strings.TrimSpace(string(out))
		if v == "" {
			return "", errors.New("the configured command printed nothing")
		}
		commandCache.Store(config, v)
		return v, nil
	}
	var b strings.Builder
	for i := 0; i < len(config); {
		c := config[i]
		if c != '$' {
			b.WriteByte(c)
			i++
			continue
		}
		if i+1 < len(config) && (config[i+1] == '$' || config[i+1] == '!') {
			b.WriteByte(config[i+1])
			i += 2
			continue
		}
		var name string
		next := i + 1
		if i+1 < len(config) && config[i+1] == '{' {
			end := strings.IndexByte(config[i+2:], '}')
			if end >= 0 && envNamePrefix.MatchString(config[i+2:i+2+end]) && len(envNamePrefix.FindString(config[i+2:i+2+end])) == end {
				name, next = config[i+2:i+2+end], i+3+end
			}
		} else if m := envNamePrefix.FindString(config[i+1:]); m != "" {
			name, next = m, i+1+len(m)
		}
		if name == "" {
			b.WriteByte('$')
			i++
			continue
		}
		v, ok := env[name]
		if !ok || v == "" {
			v, ok = a.opts.env(name)
		}
		if !ok || v == "" {
			return "", errors.Errorf("environment variable %s is not set", name)
		}
		b.WriteString(v)
		i = next
	}
	return b.String(), nil
}
