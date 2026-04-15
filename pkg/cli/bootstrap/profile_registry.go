package bootstrap

import (
	"context"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
)

type ResolvedProfileRegistryChain struct {
	Registry              gepprofiles.Registry
	Reader                gepprofiles.RegistryReader
	DefaultRegistrySlug   gepprofiles.RegistrySlug
	DefaultProfileResolve gepprofiles.ResolveInput
	Close                 func()
}

func ResolveProfileRegistryChain(ctx context.Context, selection ProfileSettings) (*ResolvedProfileRegistryChain, error) {
	if len(selection.ProfileRegistries) == 0 {
		if selection.Profile != "" {
			return nil, &gepprofiles.ValidationError{
				Field:  "profile-settings.profile-registries",
				Reason: "must be configured when profile-settings.profile is set",
			}
		}
		return &ResolvedProfileRegistryChain{}, nil
	}

	specs, err := gepprofiles.ParseRegistrySourceSpecs(selection.ProfileRegistries)
	if err != nil {
		return nil, err
	}
	chain, err := gepprofiles.NewChainedRegistryFromSourceSpecs(ctx, specs)
	if err != nil {
		return nil, err
	}

	defaultResolve := gepprofiles.ResolveInput{}
	if selection.Profile != "" {
		profileSlug, err := gepprofiles.ParseEngineProfileSlug(selection.Profile)
		if err != nil {
			_ = chain.Close()
			return nil, err
		}
		defaultResolve.EngineProfileSlug = profileSlug
	}

	return &ResolvedProfileRegistryChain{
		Registry:              chain,
		Reader:                chain,
		DefaultRegistrySlug:   chain.DefaultRegistrySlug(),
		DefaultProfileResolve: defaultResolve,
		Close: func() {
			_ = chain.Close()
		},
	}, nil
}
