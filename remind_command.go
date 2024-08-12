package main

import (
	"time"

	"go.minekube.com/brigodier"
	"maunium.net/go/mautrix/id"
)

func init() {
	CommandDispatcher.Register(
		brigodier.Literal("remind").
			Then(
				brigodier.Argument("time", brigodier.Int64).
					Executes(
						brigodier.CommandFunc(func(c *brigodier.CommandContext) error {
							room_id := c.Context.Value(RoomContext).(id.RoomID)
							sender := c.Context.Value(SenderContext).(string)
							ts := c.Int64("time")

							t := time.Unix(ts, 0)

							AddSchedule(sender, "no message", room_id, &t)

							_, err := client.SendText(c.Context, room_id, "Added your schedule")
							return err
						})).
					Then(brigodier.Argument("message", brigodier.String).
						Executes(
							brigodier.CommandFunc(
								func(c *brigodier.CommandContext) error {
									room_id := c.Context.Value(RoomContext).(id.RoomID)
									sender := c.Context.Value(SenderContext).(string)
									ts := c.Int64("time")
									msg := c.String("message")
									t := time.Unix(ts, 0)

									AddSchedule(sender, msg, room_id, &t)

									_, err := client.SendText(c.Context, room_id, "Added your schedule")
									return err
								})))),
	)
}
