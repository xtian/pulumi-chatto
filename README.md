# Chatto Pulumi Provider

Manage [Chatto](https://chatto.run) servers with [Pulumi](https://www.pulumi.com). The provider supports
these resources:

| Resource                      | Purpose                                                              | Resource ID                                       |
| ----------------------------- | -------------------------------------------------------------------- | ------------------------------------------------- |
| `chatto:index:RoomGroup`      | A sidebar section that holds rooms                                   | Group ID                                          |
| `chatto:index:Room`           | A channel room                                                       | Room ID                                           |
| `chatto:index:Role`           | A custom server role                                                 | Role name                                         |
| `chatto:index:RolePermission` | An allow or deny decision for a role at server, group, or room scope | `<roleName>/<permission>/<scopeKind>[/<scopeId>]` |

## Installation

The provider is not in the Pulumi Registry. Add it to `Pulumi.yaml` with the
URL of its GitHub releases, then run `pulumi install`:

```yaml
packages:
  chatto:
    source: chatto
    version: 0.1.0
    pluginDownloadURL: github://api.github.com/xtian/pulumi-chatto
```

`pulumi install` generates a local SDK in `sdks/`. In TypeScript, import it
as `@xtian/chatto`.

## Configuration

| Key             | Environment variable | Description                                                    |
| --------------- | -------------------- | -------------------------------------------------------------- |
| `chatto:url`    | `CHATTO_URL`         | Base URL of the server, for example `https://chat.example.com` |
| `chatto:apiKey` | `CHATTO_API_KEY`     | Bot API key (secret)                                           |

The provider uses a bot API key. Chatto does not give roles to bots, so grant
each permission to the bot directly:

- `room.manage` for room groups and room settings
- `room.create` to create rooms
- `role.manage` for roles and role permissions

The owner of the bot must also hold these permissions. An admin must activate
privileged mode before they grant them.

## Example

```yaml
resources:
  operations:
    type: chatto:RoomGroup
    properties:
      name: Operations

  alerts:
    type: chatto:Room
    properties:
      groupId: ${operations.id}
      name: alerts
      threadingMode: required

  tech:
    type: chatto:Role
    properties:
      name: tech
      pingable: true

  techManagesRooms:
    type: chatto:RolePermission
    properties:
      decision: allow
      permission: room.manage
      roleName: ${tech.name}
```

See `examples/yaml` for a complete program.

## Limitations

- Chatto cannot delete rooms. When you delete a `Room`, the provider archives
  the room.
- Room names are unique on a server. An archived room keeps its name.
- Chatto deletes only empty room groups, and archived rooms stay in their
  group. Thus, you cannot delete a group that held a room.

## Import

Import a resource with its resource ID:

```sh
pulumi import chatto:index:Room alerts R7ooYlq4dv59W6j
pulumi import chatto:index:RolePermission techPins tech/message.pin/room/R7ooYlq4dv59W6j
```

## Development

Enter the development shell with `nix develop`, then:

```sh
generate   # regenerate the API client and permission enum from Chatto
build      # build bin/pulumi-resource-chatto
treefmt    # format every file
check      # everything CI runs: generated code, licenses, lint, tests
sdk        # generate the language SDKs into sdk/
```

Outside the shell, run any of these with `nix run .#<name>`, for example
`nix run .#check`.

Push a `v*` tag to release. GoReleaser publishes the plugin archives that
Pulumi downloads.

## Chatto Version

The flake pins the Chatto source (`v0.5.0`). The API client and the
`Permission` enum come from that version. To support a newer Chatto, change the
`chatto` input in `flake.nix`, then run `nix flake update chatto` and
`generate`.

## License

Apache-2.0. The provider calls Chatto's public API, whose protobuf definitions
are Apache-2.0. It does not include code from the Chatto server.
