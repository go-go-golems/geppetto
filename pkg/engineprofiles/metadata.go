package engineprofiles

import "time"

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

// TouchRegistryMetadata updates version/provenance fields for a registry mutation.
func TouchRegistryMetadata(meta *RegistryMetadata, opts SaveOptions, now int64) {
	if meta == nil {
		return
	}
	if now <= 0 {
		now = nowMillis()
	}
	if meta.CreatedAtMs == 0 {
		meta.CreatedAtMs = now
	}
	if meta.CreatedBy == "" && opts.Actor != "" {
		meta.CreatedBy = opts.Actor
	}
	if opts.Source != "" {
		meta.Source = opts.Source
	}
	meta.UpdatedAtMs = now
	if opts.Actor != "" {
		meta.UpdatedBy = opts.Actor
	}
	meta.Version++
}

// TouchEngineProfileMetadata updates version/provenance fields for a profile mutation.
func TouchEngineProfileMetadata(meta *EngineProfileMetadata, opts SaveOptions, now int64) {
	if meta == nil {
		return
	}
	if now <= 0 {
		now = nowMillis()
	}
	if meta.CreatedAtMs == 0 {
		meta.CreatedAtMs = now
	}
	if meta.CreatedBy == "" && opts.Actor != "" {
		meta.CreatedBy = opts.Actor
	}
	if opts.Source != "" {
		meta.Source = opts.Source
	}
	meta.UpdatedAtMs = now
	if opts.Actor != "" {
		meta.UpdatedBy = opts.Actor
	}
	meta.Version++
}
