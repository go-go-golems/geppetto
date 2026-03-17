package scopedjs

import (
	"fmt"
	"strings"

	gojengine "github.com/go-go-golems/go-go-goja/engine"
	ggjmodules "github.com/go-go-golems/go-go-goja/modules"
)

func (b *Builder) AddModule(name string, register ModuleRegistrar, doc ModuleDoc) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return fmt.Errorf("module name is required")
	}
	if register == nil {
		return fmt.Errorf("module %q register function is nil", trimmedName)
	}
	doc.Name = firstNonEmpty(doc.Name, trimmedName)
	doc.Exports = NormalizeNonEmptyStrings(doc.Exports)
	b.modules = append(b.modules, moduleEntry{
		name:     trimmedName,
		register: register,
		doc:      doc,
	})
	b.manifest.Modules = append(b.manifest.Modules, doc)
	return nil
}

func (b *Builder) AddNativeModule(mod ggjmodules.NativeModule) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	if mod == nil {
		return fmt.Errorf("native module is nil")
	}
	name := strings.TrimSpace(mod.Name())
	if name == "" {
		return fmt.Errorf("native module name is empty")
	}
	b.nativeModules = append(b.nativeModules, mod)
	b.manifest.Modules = append(b.manifest.Modules, ModuleDoc{
		Name:        name,
		Description: strings.TrimSpace(mod.Doc()),
	})
	return nil
}

func (b *Builder) AddGlobal(name string, bind GlobalBinding, doc GlobalDoc) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return fmt.Errorf("global name is required")
	}
	if bind == nil {
		return fmt.Errorf("global %q binding is nil", trimmedName)
	}
	doc.Name = firstNonEmpty(doc.Name, trimmedName)
	b.globals = append(b.globals, globalEntry{
		name: trimmedName,
		bind: bind,
		doc:  doc,
	})
	b.manifest.Globals = append(b.manifest.Globals, doc)
	return nil
}

func (b *Builder) AddInitializer(init gojengine.RuntimeInitializer) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	if init == nil {
		return fmt.Errorf("runtime initializer is nil")
	}
	b.initializers = append(b.initializers, init)
	return nil
}

func (b *Builder) AddBootstrapSource(name string, source string) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return fmt.Errorf("bootstrap source name is required")
	}
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("bootstrap source %q is empty", trimmedName)
	}
	b.bootstrapEntries = append(b.bootstrapEntries, bootstrapEntry{
		name:   trimmedName,
		source: source,
	})
	b.manifest.BootstrapFiles = append(b.manifest.BootstrapFiles, trimmedName)
	return nil
}

func (b *Builder) AddBootstrapFile(path string) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Errorf("bootstrap file path is required")
	}
	b.bootstrapEntries = append(b.bootstrapEntries, bootstrapEntry{
		name:     trimmedPath,
		filePath: trimmedPath,
	})
	b.manifest.BootstrapFiles = append(b.manifest.BootstrapFiles, trimmedPath)
	return nil
}

func (b *Builder) AddHelper(name string, signature string, description string) error {
	if b == nil {
		return fmt.Errorf("builder is nil")
	}
	trimmedName := strings.TrimSpace(name)
	trimmedSig := strings.TrimSpace(signature)
	if trimmedName == "" && trimmedSig == "" {
		return fmt.Errorf("helper name or signature is required")
	}
	b.manifest.Helpers = append(b.manifest.Helpers, HelperDoc{
		Name:        trimmedName,
		Signature:   trimmedSig,
		Description: strings.TrimSpace(description),
	})
	return nil
}

func (b *Builder) Manifest() EnvironmentManifest {
	if b == nil {
		return EnvironmentManifest{}
	}
	return cloneManifest(b.manifest)
}
