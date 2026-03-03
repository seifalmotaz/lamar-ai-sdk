# API Stability Guarantees

This document defines the stability guarantees for the Lamar SDK.

---

## Versioning

Lamar follows [Semantic Versioning 2.0.0](https://semver.org/).

### Version Numbers

- **v0.x.x**: Pre-release. APIs may change without notice.
- **v1.x.x**: Stable release with backward compatibility guarantees.
- **v2.x.x**: Breaking changes allowed with migration guide.

### Version Components

- **MAJOR (X.0.0)**: Breaking changes
- **MINOR (0.X.0)**: New features, backward compatible
- **PATCH (0.0.X)**: Bug fixes, backward compatible

---

## Breaking Changes

A change is considered **breaking** if:

1. Removing or renaming an exported type, function, method, or constant
2. Changing a function signature (parameters or return types)
3. Adding a required parameter to a function
4. Changing an interface definition
5. Removing methods from an interface
6. Changing the JSON serialization format of exported types
7. Changing default behavior that existing code relies on
8. Renaming exported struct fields
9. Changing the order of struct fields (affects JSON encoding)
10. Changing error types or codes for existing scenarios

### Non-Breaking Changes

The following changes are **not** breaking:

1. Adding new exported types, functions, methods, or constants
2. Adding new optional parameters via functional options
3. Adding new fields to structs (with zero value defaults)
4. Adding new methods to an interface (with documentation that implementations should embed)
5. Adding new error codes
6. Improving performance
7. Fixing bugs (behavior that was clearly incorrect)
8. Adding new providers
9. Changing internal (unexported) implementation details

---

## Deprecation Policy

### Deprecation Process

1. **Announcement**: Feature marked as deprecated with `// Deprecated:` comment
2. **Documentation**: Migration guide provided in comments
3. **Migration Period**: At least one major version with both old and new APIs
4. **Removal**: In next major version after migration period

### Deprecation Timeline

| Release Type | Minimum Migration Period |
|--------------|-------------------------|
| v0.x to v1.0 | No migration period required |
| v1.x to v2.0 | 6 months minimum |
| v2.x to v3.0 | 6 months minimum |

### Deprecation Example

```go
// GenerateSchema generates a response matching the provided schema.
//
// Deprecated: Use GenerateObject[T] for type-safe schema generation.
// This method will be removed in v2.0.0.
//
// Migration:
//   Before: GenerateSchema(ctx, model, prompt, schema)
//   After:  GenerateObject[T](ctx, model, prompt)
func GenerateSchema(ctx context.Context, model Generator, prompt string, schema any) (*GenerateResult, error) {
    // Legacy implementation
    return generateWithSchema(ctx, model, prompt, schema)
}

// GenerateObject generates structured output matching a schema inferred from T.
func GenerateObject[T any](ctx context.Context, model Generator, prompt string, opts ...Option) (*ObjectResult[T], error) {
    // New implementation
}
```

---

## Stability Tiers

### Tier 1: Stable

- Production-ready and battle-tested
- Full backward compatibility guarantees
- Semantic versioning enforced
- Comprehensive documentation
- Full test coverage

**Tier 1 Packages**:
- `lamar` (root package)
- `lamar/provider`
- `lamar/generate`
- `lamar/stream`
- `lamar/embed`
- `lamar/tool`
- `lamar/providers/openai`

### Tier 2: Beta

- Feature-complete, may have rough edges
- Minor changes possible between minor versions
- May change based on user feedback
- Good documentation
- Tested but not production-hardened

**Tier 2 Packages**:
- `lamar/providers/anthropic`
- `lamar/providers/google`
- `lamar/middleware`

### Tier 3: Alpha

- Experimental features
- No stability guarantees
- May be removed without notice
- Limited documentation

**Tier 3 Packages**:
- `lamar/providers/azure`
- `lamar/providers/amazon-bedrock`
- New experimental providers

### Tier 4: Internal

- Implementation details
- No guarantees whatsoever
- May change at any time
- Not part of the public API

**Tier 4 Packages**:
- `lamar/internal/*`
- All files in `internal/` directories

---

## Version Support

### Supported Versions

| Version | Status | Support Ends | Notes |
|---------|--------|--------------|-------|
| v0.x | Pre-release | v1.0 release | Breaking changes allowed |
| v1.x | Stable | 1 year after v2.0 release | Full support |
| v2.x | Future | TBD | Planning |

### Security Patches

Security fixes are backported to:
- Current stable version (v1.x)
- Previous stable version for 6 months after next major release

---

## Compatibility Promises

### Interface Stability

All interfaces in stable packages maintain:
- Existing methods will never be removed or changed
- New methods may be added (implementations should embed interfaces)

```go
// Correct: Embed interface for forward compatibility
type MyModel struct{}

func (m *MyModel) Provider() string { return "my-provider" }
func (m *MyModel) ModelID() string  { return "my-model" }
func (m *MyModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
    // implementation
}
func (m *MyModel) Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error) {
    // implementation
}

// Compile-time verification
var _ provider.LanguageModel = (*MyModel)(nil)
```

### Function Signature Stability

Public functions will never change signatures in v1.x:

```go
// This signature is stable for v1.x
func Generate(ctx context.Context, model Generator, prompt string, opts ...Option) (*Result, error)

// New options can be added without breaking changes
func NewFeature(opts ...Option) Option {
    return func(c *Config) {
        c.newFeature = true
    }
}
```

### Type Compatibility

Types in stable packages:
- Struct fields will not be removed or renamed
- New fields may be added (with zero value defaults)
- JSON serialization format is stable

---

## Experimental Features

Features marked as "experimental" are exempt from stability guarantees:

```go
// Experimental: This API may change in future versions.
func ExperimentalFeature() { ... }
```

Experimental features:
- May change between minor versions
- May be removed without deprecation period
- Should be used with caution in production

---

## Release Process

### Pre-release Checklist

For each release:

- [ ] All tests pass (unit, integration, contract)
- [ ] No breaking changes in minor/patch releases
- [ ] CHANGELOG.md updated
- [ ] Documentation updated
- [ ] Migration guide provided for deprecations
- [ ] Version tag created

### Release Cadence

| Release Type | Cadence |
|--------------|---------|
| Patch | As needed for bug fixes |
| Minor | Monthly, or as features are ready |
| Major | Annually, or when breaking changes needed |

---

## Breaking Change Process

When a breaking change is necessary:

### 1. Proposal Phase

- Create GitHub issue with "breaking-change" label
- Document reason for breaking change
- Propose API replacement
- Gather community feedback (minimum 2 weeks)

### 2. Implementation Phase

- Implement new API alongside deprecated API
- Add `// Deprecated:` comments with migration guide
- Update documentation
- Add compile-time deprecation warnings where possible

### 3. Release Phase

- Release with deprecated API (minor version bump)
- Announce deprecation in release notes
- Update migration guide

### 4. Removal Phase

- Wait minimum migration period (6 months)
- Remove deprecated API in next major version
- Provide automated migration tools if possible

---

## Migration Guides

Migration guides are maintained for all breaking changes:

### Location

```
docs/migrations/
├── v0-to-v1.md
├── v1-to-v2.md
└── ...
```

### Content

Each migration guide includes:

1. **Overview**: Summary of changes
2. **Breaking Changes**: Detailed list with before/after examples
3. **Automatic Migration**: Commands for automated migration
4. **Manual Steps**: Required manual changes
5. **Deprecations**: Features deprecated in this version
6. **Timeline**: When deprecated features will be removed

### v0 to v1 Migration Example

```markdown
# Migrating from v0.x to v1.0

## Overview

v1.0 introduces stable APIs with backward compatibility guarantees.

## Breaking Changes

### ContentPart replaced with polymorphic Content

**Before (v0.x)**:
```go
part := provider.ContentPart{
    Type: "text",
    Text: "Hello",
}
```

**After (v1.0)**:
```go
part := provider.Text("Hello")
// or
part := provider.TextContent{Text: "Hello"}
```

### Interface Segregation

**Before (v0.x)**:
```go
var _ provider.LanguageModelV3 = (*MyModel)(nil)
```

**After (v1.0)**:
```go
var _ provider.Model = (*MyModel)(nil)
var _ provider.Generator = (*MyModel)(nil)
var _ provider.Streamer = (*MyModel)(nil)
var _ provider.LanguageModel = (*MyModel)(nil)
```
```

---

## Exception Process

### Security Exceptions

Security vulnerabilities may require immediate breaking changes:
- Emergency patches bypass normal deprecation process
- Affected APIs marked as deprecated immediately
- Migration guide provided within 48 hours

### Critical Bug Exceptions

Bugs that cause data loss or incorrect behavior:
- Fixes may be released as patches
- If fix requires breaking change, follow deprecation process
- Clear documentation of expected vs actual behavior

---

## Document History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-03-03 | Initial API stability policy |

---

## Questions

For questions about API stability:

1. Check this document
2. Check migration guides in `docs/migrations/`
3. Open a GitHub issue with the "question" label
4. Check existing GitHub discussions