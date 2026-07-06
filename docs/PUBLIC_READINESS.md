# Public Readiness

> **Historical note:** `M31-Labs/gosx-studio` went public some time ago — the
> "Current Gate" and "Before Visibility Changes" sections below describe the
> pre-visibility-change state and are kept as history, not as an active gate.
> The portal definition and package structure also postdate this document;
> see [README.md](../README.md) and [ARCHITECTURE.md](ARCHITECTURE.md) for
> the current framing.

`M31-Labs/gosx-studio` is intended to become the public package for the website authoring layer.

## Current Gate

- Keep the repository private until the owner explicitly confirms the visibility change.
- Keep host-specific secrets, customer data, generated media, and Noni-only implementation details out of this repo.
- Keep public package APIs focused on configuration contracts, `.gsx` surfaces, engines, islands, and plugin boundaries.
- Keep Noni's Mud Relics as a reference implementation, not as package source.

## Before Visibility Changes

- Confirm license and release policy.
- Confirm package path casing and module import path.
- Run `go test ./...`.
- Search for credentials, private routes, customer records, and app-specific generated assets.
- Confirm docs explain how `gosx-cms`, `gosx-admin`, and `gosx-studio` compose without implying this repo owns CMS or admin resources.
