---
title: Chatto
meta_desc: Use the Chatto provider for Pulumi to manage Chatto bots, rooms, room groups, roles, and permissions.
layout: package
---

The Chatto provider manages [Chatto](https://chatto.run) servers: bots, bot
permissions, room groups, rooms, room members, roles, and role permissions.

## Installation

The provider has no published SDKs. Install the plugin from its
[GitHub releases](https://github.com/xtian/pulumi-chatto/releases), then
generate an SDK for your program's language:

```sh
pulumi plugin install resource chatto --server github://api.github.com/xtian/pulumi-chatto
pulumi package add chatto
```

`pulumi package add` reads the language from the `runtime` in `Pulumi.yaml`,
generates the SDK in `sdks/`, records the package in `Pulumi.yaml`, and prints
the import to use.

## Example Usage

Set the URL of the server and a bot API key:

```sh
pulumi config set chatto:url https://chat.example.com
pulumi config set --secret chatto:apiKey <api-key>
```

Then declare a room group:

{{< chooser language "typescript,python,go,csharp,java,yaml" >}}

{{% choosable language typescript %}}

```typescript
import * as chatto from "@xtian/chatto";

const operations = new chatto.RoomGroup("operations", { name: "Operations" });

export const groupId = operations.id;
```

{{% /choosable %}}

{{% choosable language python %}}

```python
import pulumi
import xtian_chatto as chatto

operations = chatto.RoomGroup("operations", name="Operations")

pulumi.export("group_id", operations.id)
```

{{% /choosable %}}

{{% choosable language go %}}

```go
package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/xtian/pulumi-chatto/sdk/go/chatto"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		operations, err := chatto.NewRoomGroup(ctx, "operations", &chatto.RoomGroupArgs{
			Name: pulumi.String("Operations"),
		})
		if err != nil {
			return err
		}

		ctx.Export("groupId", operations.ID())
		return nil
	})
}
```

{{% /choosable %}}

{{% choosable language csharp %}}

```csharp
using System.Collections.Generic;
using Pulumi;
using Xtian.Chatto;

return await Deployment.RunAsync(() =>
{
    var operations = new RoomGroup("operations", new RoomGroupArgs
    {
        Name = "Operations",
    });

    return new Dictionary<string, object?>
    {
        ["groupId"] = operations.Id,
    };
});
```

{{% /choosable %}}

{{% choosable language java %}}

```java
package myproject;

import com.pulumi.Pulumi;
import com.xtian.chatto.RoomGroup;
import com.xtian.chatto.RoomGroupArgs;

public class App {
    public static void main(String[] args) {
        Pulumi.run(ctx -> {
            var operations = new RoomGroup("operations", RoomGroupArgs.builder()
                .name("Operations")
                .build());

            ctx.export("groupId", operations.id());
        });
    }
}
```

{{% /choosable %}}

{{% choosable language yaml %}}

```yaml
resources:
  operations:
    type: chatto:RoomGroup
    properties:
      name: Operations

outputs:
  groupId: ${operations.id}
```

{{% /choosable %}}

{{< /chooser >}}

## Configuration

- `url` (Required) — Base URL of the Chatto server, for example
  `https://chat.example.com`. May also be set with the `CHATTO_URL` environment
  variable.
- `apiKey` (Required, Secret) — Bot API key. May also be set with the
  `CHATTO_API_KEY` environment variable.

## Prerequisites

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

## Import

Import a resource with its resource ID:

```sh
pulumi import chatto:index:Room alerts R7ooYlq4dv59W6j
pulumi import chatto:index:RolePermission techPins tech/message.pin/room/R7ooYlq4dv59W6j
pulumi import chatto:index:Bot alertBot U5kq2b7xS1fYh0a
pulumi import chatto:index:BotPermission alertBotPosts U5kq2b7xS1fYh0a/message.post/room/R7ooYlq4dv59W6j
pulumi import chatto:index:RoomMember alertBotInAlerts R7ooYlq4dv59W6j/U5kq2b7xS1fYh0a
```

## Limitations

- Chatto cannot delete rooms. When you delete a `Room`, the provider archives
  the room.
- Room names are unique on a server. An archived room keeps its name.
- Universal, DM, and archived rooms can't take a `RoomMember`.
- Chatto deletes only empty room groups, and archived rooms stay in their
  group. Thus, you cannot delete a group that held a room.
