# Chatto Pulumi Provider

Manage [Chatto](https://chatto.run) servers with [Pulumi](https://www.pulumi.com). The provider supports
these resources:

| Resource                      | Purpose                                                              | Resource ID                                        |
| ----------------------------- | -------------------------------------------------------------------- | -------------------------------------------------- |
| `chatto:index:Bot`            | A bot account                                                        | Bot user ID                                        |
| `chatto:index:BotPermission`  | A permission granted to a bot at server, group, room, or DM scope    | `<botUserId>/<permission>/<scopeKind>[/<scopeId>]` |
| `chatto:index:RoomGroup`      | A sidebar section that holds rooms                                   | Group ID                                           |
| `chatto:index:Room`           | A channel room                                                       | Room ID                                            |
| `chatto:index:RoomMember`     | An explicit member of a channel room                                 | `<roomId>/<userId>`                                |
| `chatto:index:Role`           | A custom server role                                                 | Role name                                          |
| `chatto:index:RolePermission` | An allow or deny decision for a role at server, group, or room scope | `<roleName>/<permission>/<scopeKind>[/<scopeId>]`  |

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
- `bot.create` to create bots, and `bot.manage` to manage bots that another
  user owns
- `room.manage` for the room, or `user.manage-accounts`, to add room members

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

## Bots

A `Bot` gets no permissions from roles. Grant each one with `BotPermission`.
A bot can use a permission only while its human owner also holds it. Chatto
makes the human owner of the provider's API key the owner of each bot that the
provider creates.

To grant permissions to the provider's own bot, look up its ID with the
`chatto:index:getViewer` function, which returns the `id`, `login`,
`displayName`, and `bot` flag of the API key's account.

Chatto shows a bot's first API key only once. The provider stores it in the
secret `apiKey` output, which you can read with
`pulumi stack output --show-secrets`. An imported bot has no `apiKey` output.

```yaml
resources:
  alertBot:
    type: chatto:Bot
    properties:
      displayName: Alerts
      login: alert-bot

  alertBotPosts:
    type: chatto:BotPermission
    properties:
      botUserId: ${alertBot.id}
      permission: message.post
      scopeId: ${alerts.id}
      scopeKind: room

  alertBotInAlerts:
    type: chatto:RoomMember
    properties:
      roomId: ${alerts.id}
      userId: ${alertBot.id}
```

A bot can post only to rooms it belongs to. Universal rooms include it
automatically; add it to any other room with `RoomMember`.

See `examples/yaml` for a complete program.

## Limitations

- Chatto cannot delete rooms. When you delete a `Room`, the provider archives
  the room.
- Room names are unique on a server. An archived room keeps its name.
- Universal, DM, and archived rooms can't take a `RoomMember`.
- Chatto deletes only empty room groups, and archived rooms stay in their
  group. Thus, you cannot delete a group that held a room.

## Import

Import a resource with its resource ID:

```sh
pulumi import chatto:index:Room alerts R7ooYlq4dv59W6j
pulumi import chatto:index:RolePermission techPins tech/message.pin/room/R7ooYlq4dv59W6j
pulumi import chatto:index:Bot alertBot U5kq2b7xS1fYh0a
pulumi import chatto:index:BotPermission alertBotPosts U5kq2b7xS1fYh0a/message.post/room/R7ooYlq4dv59W6j
pulumi import chatto:index:RoomMember alertBotInAlerts R7ooYlq4dv59W6j/U5kq2b7xS1fYh0a
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
