// Package config loads and exposes runtime configuration.
package config

import (
	"strings"

	cxlist "github.com/arcgolabs/collectionx/list"
	cxset "github.com/arcgolabs/collectionx/set"
)

// FallbackOn defines the condition under which a fallback file is served.
type FallbackOn string

// SourceBackend defines the asset source backend type.
type SourceBackend string

const (
	// SourceBackendLocal serves assets from a local filesystem root.
	SourceBackendLocal SourceBackend = "local"
)

var supportedSourceBackends = cxset.NewOrderedSet[SourceBackend](SourceBackendLocal)

const (
	// FallbackOnNotFound indicates that the fallback target
	// will be served when the requested asset is not found
	// in the asset registry (i.e. lookup miss).
	FallbackOnNotFound FallbackOn = "not_found"

	// FallbackOnForbidden indicates that the fallback target
	// will be served when the requested asset exists but is
	// not accessible due to permission or policy restrictions.
	FallbackOnForbidden FallbackOn = "forbidden"
)

// Assets defines the static asset serving configuration.
// It represents a single mount point that maps an HTTP URL path
// prefix to a filesystem root directory.
//
// The design assumes:
//   - one process serves one asset mount
//   - no dynamic routing or multi-location resolution
//   - all assets are scanned and registered at startup
type Assets struct {
	// Path is the HTTP request path prefix used to match incoming requests.
	//
	// It must start with '/' and represents the virtual root of static assets.
	// For example:
	//   "/"      → serve assets from the site root
	//   "/static" → serve assets under /static/*
	//
	// This path is matched before any filesystem lookup occurs.
	Path string `koanf:"path" validate:"required,startswith=/"`

	// Backend selects the source backend implementation.
	//
	// Supported values:
	//   - "local": scan assets from a local filesystem root
	//
	// Unknown values are rejected during source initialization.
	Backend SourceBackend `koanf:"backend" validate:"required,oneof=local"`

	// Root is the filesystem directory used as the source of static assets.
	//
	// All files under this directory will be scanned at startup
	// and registered into the in-memory asset registry.
	//
	// This path should point to an existing directory and is
	// typically resolved to an absolute path during initialization.
	Root string `koanf:"root" validate:"required"`

	// Entry is the default entry file name used for directory requests.
	//
	// When a request resolves to a directory path, the server will
	// attempt to serve the file specified by Entry within that directory.
	//
	// Typical values include:
	//   - "index.html"
	//
	// Entry must be a relative file name and must not start with '/'.
	Entry string `koanf:"entry" validate:"required,spack_relative_path"`

	// Fallback defines the fallback serving behavior when a request
	// cannot be resolved normally.
	//
	// The fallback target must already exist in the asset registry
	// and is resolved as a virtual path (not a filesystem path).
	//
	// If no fallback behavior is desired, this field should be left
	// empty in the configuration.
	Fallback Fallback `koanf:"fallback" validate:"required"`
}

func NormalizeSourceBackend(raw SourceBackend) SourceBackend {
	normalized := SourceBackend(strings.ToLower(strings.TrimSpace(string(raw))))
	switch normalized {
	case "", SourceBackendLocal:
		return SourceBackendLocal
	default:
		return normalized
	}
}

func SupportedSourceBackends() *cxlist.List[SourceBackend] {
	return cxlist.NewList[SourceBackend](supportedSourceBackends.Values()...)
}

func SupportedSourceBackendNames() *cxlist.List[string] {
	return cxlist.MapList[SourceBackend, string](SupportedSourceBackends(), func(_ int, backend SourceBackend) string {
		return string(backend)
	})
}

func IsSupportedSourceBackend(backend SourceBackend) bool {
	return supportedSourceBackends.Contains(backend)
}

// NormalizedBackend returns the effective source backend name.
func (a Assets) NormalizedBackend() SourceBackend {
	return NormalizeSourceBackend(a.Backend)
}

// Fallback defines the rules for serving a fallback asset
// when a request cannot be fulfilled under certain conditions.
type Fallback struct {
	// On specifies the condition that triggers the fallback.
	//
	// Supported values:
	//   - "not_found"  : triggered when asset lookup fails
	//   - "forbidden"  : triggered when asset access is denied
	On FallbackOn `koanf:"on" validate:"omitempty,oneof=not_found forbidden"`

	// Target is the virtual asset path to be served as fallback.
	//
	// This path is resolved against the asset registry and must
	// exist at startup time. It typically points to an entry file
	// such as "/index.html".
	//
	// The target is not re-scanned or dynamically resolved at runtime.
	Target string `koanf:"target" validate:"required_with=On,omitempty,spack_relative_path"`
}
