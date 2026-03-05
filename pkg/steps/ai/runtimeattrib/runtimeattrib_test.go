package runtimeattrib

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
)

func TestAttachToExtraProfileVersionNormalization(t *testing.T) {
	tests := []struct {
		name     string
		runtime  map[string]any
		want     uint64
		wantSeen bool
	}{
		{
			name:     "dotted int64",
			runtime:  map[string]any{"profile.version": int64(3)},
			want:     3,
			wantSeen: true,
		},
		{
			name:     "dotted int",
			runtime:  map[string]any{"profile.version": 4},
			want:     4,
			wantSeen: true,
		},
		{
			name:     "dotted float64 integer value",
			runtime:  map[string]any{"profile.version": 5.0},
			want:     5,
			wantSeen: true,
		},
		{
			name:     "underscored int still supported",
			runtime:  map[string]any{"profile_version": int(6)},
			want:     6,
			wantSeen: true,
		},
		{
			name:     "dotted takes precedence over underscored",
			runtime:  map[string]any{"profile.version": int64(7), "profile_version": int64(8)},
			want:     7,
			wantSeen: true,
		},
		{
			name:     "non-integer float ignored",
			runtime:  map[string]any{"profile.version": 1.5},
			wantSeen: false,
		},
		{
			name:     "zero ignored",
			runtime:  map[string]any{"profile.version": 0},
			wantSeen: false,
		},
		{
			name:     "negative ignored",
			runtime:  map[string]any{"profile.version": -1},
			wantSeen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := &turns.Turn{}
			if err := turns.KeyTurnMetaRuntime.Set(&turn.Metadata, tt.runtime); err != nil {
				t.Fatalf("set runtime metadata: %v", err)
			}

			extra := map[string]any{}
			AddRuntimeAttributionToExtra(extra, turn)

			got, ok := extra["profile.version"]
			if ok != tt.wantSeen {
				t.Fatalf("profile.version presence mismatch: got %v want %v (value=%#v)", ok, tt.wantSeen, got)
			}
			if !tt.wantSeen {
				return
			}

			gotU64, ok := got.(uint64)
			if !ok {
				t.Fatalf("profile.version type mismatch: got %T want uint64", got)
			}
			if gotU64 != tt.want {
				t.Fatalf("profile.version mismatch: got %d want %d", gotU64, tt.want)
			}
		})
	}
}
