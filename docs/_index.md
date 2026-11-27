---
title: Chatto
meta_desc: Provides an overview of the Chatto provider for Pulumi.
layout: package
---

The Chatto provider manages [Chatto](https://chatto.run) servers: bots, bot
permissions, room groups, rooms, room members, roles, and role permissions.
Configure it with the URL of the server and a bot API key. See
[Installation & Configuration](./installation-configuration) for details.

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

A bot can post only to rooms it belongs to. Universal rooms include it
automatically; add it to any other room with `RoomMember`.

## Limitations

- Chatto cannot delete rooms. When you delete a `Room`, the provider archives
  the room.
- Room names are unique on a server. An archived room keeps its name.
- Universal, DM, and archived rooms can't take a `RoomMember`.
- Chatto deletes only empty room groups, and archived rooms stay in their
  group. Thus, you cannot delete a group that held a room.
