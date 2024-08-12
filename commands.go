package main

import "go.minekube.com/brigodier"

var CommandDispatcher = &brigodier.Dispatcher{}

var (
	SenderContext = new(any)
	RoomContext   = new(any)
)
