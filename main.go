package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exzerolog"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
)

var homeserver = flag.String("homeserver", "", "Matrix homeserver")
var username = flag.String("username", "", "Matrix username localpart")
var password = flag.String("password", "", "Matrix password")
var database = flag.String("database", "mautrix-example.db", "SQLite database path")
var debug = flag.Bool("debug", false, "Enable debug logs")

var client *mautrix.Client

func main() {
	flag.Parse()
	var err error
	if *username == "" || *password == "" || *homeserver == "" {
		_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	client, err = mautrix.NewClient(*homeserver, "", "")
	if err != nil {
		panic(err)
	}
	rl, err := readline.New("")
	if err != nil {
		panic(err)
	}
	defer rl.Close()
	log := zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = rl.Stdout()
		w.TimeFormat = time.Stamp
	})).With().Timestamp().Logger()
	if !*debug {
		log = log.Level(zerolog.InfoLevel)
	}
	exzerolog.SetupDefaults(&log)
	client.Log = log

	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		sender := evt.Sender.String()
		body := evt.Content.AsMessage().Body

		log.Info().
			Str("sender", sender).
			Str("type", evt.Type.String()).
			Str("id", evt.ID.String()).
			Str("body", body).
			Msg("Received message")

		if cmd, found := strings.CutPrefix(body, "!"); found {
			ctx = context.WithValue(ctx, SenderContext, sender)
			ctx = context.WithValue(ctx, RoomContext, evt.RoomID)

			r := CommandDispatcher.Parse(ctx, cmd)
			err = CommandDispatcher.Execute(r)
			if err != nil {
				return
			}
		}

	})
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := client.JoinRoomByID(ctx, evt.RoomID)
			if err == nil {
				log.Info().
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Joined room after invite")
			} else {
				log.Error().Err(err).
					Str("room_id", evt.RoomID.String()).
					Str("inviter", evt.Sender.String()).
					Msg("Failed to join room after invite")
			}
		}
	})

	cryptoHelper, err := cryptohelper.NewCryptoHelper(client, []byte("meow"), *database)
	if err != nil {
		panic(err)
	}

	// You can also store the user/device IDs and access token and put them in the client beforehand instead of using LoginAs.
	//client.UserID = "..."
	//client.DeviceID = "..."
	//client.AccessToken = "..."
	// You don't need to set a device ID in LoginAs because the crypto helper will set it for you if necessary.
	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: *username},
		Password:   *password,
	}
	// If you want to use multiple clients with the same DB, you should set a distinct database account ID for each one.
	//cryptoHelper.DBAccountID = ""
	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		panic(err)
	}
	// Set the client crypto helper in order to automatically encrypt outgoing messages
	client.Crypto = cryptoHelper

	log.Info().Msg("Now running")
	syncCtx, cancelSync := context.WithCancel(context.Background())
	var syncStopWait sync.WaitGroup
	syncStopWait.Add(1)

	go func() {
		err = client.SyncWithContext(syncCtx)
		defer syncStopWait.Done()
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}()

	c := ScheduleNotify()

	go func() {
		s := make(chan os.Signal, 1)
		go signal.Notify(s, os.Interrupt)
		<-s
		c <- nil
	}()

	for {
		schedule := <-c
		if schedule == nil {
			break
		}
		client.SendText(context.TODO(), schedule.Room,
			fmt.Sprintf("%s Reminder: %s", schedule.User, schedule.Message),
		)
	}
	cancelSync()
	syncStopWait.Wait()
	err = cryptoHelper.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error closing database")
	}
}
