---
title: Chatto Installation & Configuration
meta_desc: Information on how to install and configure the Chatto provider for Pulumi.
layout: package
---

## Installation

The provider plugin is published on
[GitHub releases](https://github.com/xtian/pulumi-chatto/releases). Add the
package to `Pulumi.yaml`, then run `pulumi install`:

```yaml
packages:
  chatto:
    source: chatto
    pluginDownloadURL: github://api.github.com/xtian/pulumi-chatto
```

`pulumi install` downloads the plugin and generates a local SDK in `sdks/` for
the language of your program. In TypeScript, import it as `@xtian/chatto`.

To install only the plugin:

```sh
pulumi plugin install resource chatto --server github://api.github.com/xtian/pulumi-chatto
```

## Configuration

| Key             | Environment variable | Description                                                    |
| --------------- | -------------------- | -------------------------------------------------------------- |
| `chatto:url`    | `CHATTO_URL`         | Base URL of the server, for example `https://chat.example.com` |
| `chatto:apiKey` | `CHATTO_API_KEY`     | Bot API key (secret)                                           |

```sh
pulumi config set chatto:url https://chat.example.com
pulumi config set --secret chatto:apiKey <api-key>
```

## Permissions

The provider uses a bot API key. Chatto does not give roles to bots, so grant
each permission to the bot directly:

- `room.manage` for room groups and room settings
- `room.create` to create rooms
- `role.manage` for roles and role permissions
- `bot.create` to create bots, and `bot.manage` to manage bots that another
  user owns
- `room.manage` for the room, or `user.manage-accounts`, to add room members

The owner of the bot must also hold these permissions. If they come from a
role that requires Shield Mode, the admin must activate Shield Mode before they
grant them.

## Import

Import a resource with its resource ID:

```sh
pulumi import chatto:index:Room alerts R7ooYlq4dv59W6j
pulumi import chatto:index:RolePermission techPins tech/message.pin/room/R7ooYlq4dv59W6j
pulumi import chatto:index:Bot alertBot U5kq2b7xS1fYh0a
pulumi import chatto:index:BotPermission alertBotPosts U5kq2b7xS1fYh0a/message.post/room/R7ooYlq4dv59W6j
pulumi import chatto:index:RoomMember alertBotInAlerts R7ooYlq4dv59W6j/U5kq2b7xS1fYh0a
```
