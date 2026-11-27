# Changelog

This file lists the notable changes in each release. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-11-27

### Changed

- Updated the provider for Chatto v0.6.0.

## [0.2.1] - 2026-11-23

### Fixed

- `BotPermission` and `RolePermission` accept a `scopeId` that is unknown
  during preview. Before this fix, `pulumi preview` failed with "is required
  for group scope" (or room scope) when `scopeId` came from a resource that
  did not exist yet.

## [0.2.0] - 2026-11-19

### Added

- The `Bot` resource creates a bot account. The provider stores the first API
  key of the bot in the secret `apiKey` output. An imported bot has no
  `apiKey` output.
- The `BotPermission` resource grants a permission to a bot at server, group,
  room, or DM scope.
- The `RoomMember` resource adds an explicit member to a channel room.
  Universal, DM, and archived rooms can't take a `RoomMember`.
- The `getViewer` function returns the `id`, `login`, `displayName`, and `bot`
  flag of the account that owns the API key.

## [0.1.0] - 2026-11-19

### Added

- Initial release

[0.3.0]: https://github.com/xtian/pulumi-chatto/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/xtian/pulumi-chatto/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/xtian/pulumi-chatto/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/xtian/pulumi-chatto/releases/tag/v0.1.0
