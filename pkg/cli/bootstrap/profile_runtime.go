package bootstrap

import (
	"context"

	gepprofiles "github.com/go-go-golems/geppetto/pkg/engineprofiles"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
)

type ResolvedCLIProfileRuntime struct {
	ProfileSelection     *ResolvedCLIProfileSelection
	ProfileRegistryChain *ResolvedProfileRegistryChain
	ConfigFiles          []string
	Close                func()
}

func ResolveCLIProfileRuntime(
	ctx context.Context,
	cfg AppBootstrapConfig,
	parsed *values.Values,
) (*ResolvedCLIProfileRuntime, error) {
	selection, err := ResolveCLIProfileSelection(cfg, parsed)
	if err != nil {
		return nil, err
	}

	registryChain, err := ResolveProfileRegistryChain(ctx, selection.ProfileSettings)
	if err != nil {
		return nil, err
	}

	ret := &ResolvedCLIProfileRuntime{
		ProfileSelection:     selection,
		ProfileRegistryChain: registryChain,
		ConfigFiles:          append([]string(nil), selection.ConfigFiles...),
	}
	if registryChain != nil {
		ret.Close = registryChain.Close
	}
	return ret, nil
}

func (r *ResolvedCLIProfileRuntime) Registry() gepprofiles.Registry {
	if r == nil || r.ProfileRegistryChain == nil {
		return nil
	}
	return r.ProfileRegistryChain.Registry
}

func (r *ResolvedCLIProfileRuntime) Reader() gepprofiles.RegistryReader {
	if r == nil || r.ProfileRegistryChain == nil {
		return nil
	}
	return r.ProfileRegistryChain.Reader
}
