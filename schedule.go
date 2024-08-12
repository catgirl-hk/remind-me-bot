package main

import (
	"slices"
	"time"

	"maunium.net/go/mautrix/id"
)

type Schedule struct {
	User    string
	Room    id.RoomID
	Message string
	At      *time.Time
}

var schedules []*Schedule = make([]*Schedule, 0)

func AddSchedule(user, message string, room_id id.RoomID, at *time.Time) {
	schedules = append(schedules, &Schedule{
		User:    user,
		Room:    room_id,
		Message: message,
		At:      at,
	})
}

func ScheduleNotify() chan *Schedule {
	c := make(chan *Schedule)
	go func() {
		for {
			now := time.Now().Unix()
			for i, schedule := range schedules {
				if now >= schedule.At.Unix() {
					schedules = slices.Delete(schedules, i, i+1)
					c <- schedule
				}
			}
		}
	}()
	return c
}
