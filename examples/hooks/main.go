// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package main

import (
	"bytes"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prosenjitjoy/mqtt"
	"github.com/prosenjitjoy/mqtt/hooks/auth"
	"github.com/prosenjitjoy/mqtt/listeners"
	"github.com/prosenjitjoy/mqtt/packets"
)

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	server := mqtt.New(&mqtt.Options{
		InlineClient: true, // you must enable inline client to use direct publishing and subscribing.
	})

	_ = server.AddHook(new(auth.AllowHook), nil)
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "t1",
		Address: ":1883",
	})
	err := server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	// Add custom hook (ExampleHook) to the server
	err = server.AddHook(new(ExampleHook), &ExampleHookOptions{
		Server: server,
	})

	if err != nil {
		log.Fatal(err)
	}

	// Start the server
	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Demonstration of directly publishing messages to a topic via the
	// `server.Publish` method. Subscribe to `direct/publish` using your
	// MQTT client to see the messages.
	go func() {
		cl := server.NewClient(nil, "local", "inline", true)
		for range time.Tick(time.Second * 1) {
			err := server.InjectPacket(cl, packets.Packet{
				FixedHeader: packets.FixedHeader{
					Type: packets.Publish,
				},
				TopicName: "direct/publish",
				Payload:   []byte("injected scheduled message"),
			})
			if err != nil {
				server.Log.Error().Err(err).Msg("server.InjectPacket")
			}
			server.Log.Info().Msgf("main.go injected packet to direct/publish")
		}
	}()

	// There is also a shorthand convenience function, Publish, for easily sending
	// publish packets if you are not concerned with creating your own packets.
	go func() {
		for range time.Tick(time.Second * 5) {
			err := server.Publish("direct/publish", []byte("packet scheduled message"), false, 0)
			if err != nil {
				server.Log.Error().Err(err).Msg("server.Publish")
			}
			server.Log.Info().Msgf("main.go issued direct message to direct/publish")
		}
	}()

	<-done
	server.Log.Warn().Msg("caught signal, stopping...")
	server.Close()
	server.Log.Info().Msg("main.go finished")
}

// Options contains configuration settings for the hook.
type ExampleHookOptions struct {
	Server *mqtt.Server
}

type ExampleHook struct {
	mqtt.HookBase
	config *ExampleHookOptions
}

func (h *ExampleHook) ID() string {
	return "events-example"
}

func (h *ExampleHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnect,
		mqtt.OnDisconnect,
		mqtt.OnSubscribed,
		mqtt.OnUnsubscribed,
		mqtt.OnPublished,
		mqtt.OnPublish,
	}, []byte{b})
}

func (h *ExampleHook) Init(config any) error {
	h.Log.Info().Msg("initialised")
	if _, ok := config.(*ExampleHookOptions); !ok && config != nil {
		return mqtt.ErrInvalidConfigType
	}

	h.config = config.(*ExampleHookOptions)
	if h.config.Server == nil {
		return mqtt.ErrInvalidConfigType
	}
	return nil
}

// subscribeCallback handles messages for subscribed topics
func (h *ExampleHook) subscribeCallback(cl *mqtt.Client, sub packets.Subscription, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Str("topic", pk.TopicName).Msg("hook subscribed message")
}

func (h *ExampleHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.Log.Info().Str("client", cl.ID).Msgf("client connected")

	// Example demonstrating how to subscribe to a topic within the hook.
	h.config.Server.Subscribe("hook/direct/publish", 1, h.subscribeCallback)

	// Example demonstrating how to publish a message within the hook
	err := h.config.Server.Publish("hook/direct/publish", []byte("packet hook message"), false, 0)
	if err != nil {
		h.Log.Error().Err(err).Msg("hook.publish")
	}

	return nil
}

func (h *ExampleHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {
	h.Log.Info().Str("client", cl.ID).Bool("expire", expire).Err(err).Msg("client disconnected")
}

func (h *ExampleHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msgf("subscribed qos=%v", reasonCodes)
}

func (h *ExampleHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Interface("filters", pk.Filters).Msg("unsubscribed")
}

func (h *ExampleHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Msg("received from client")

	pkx := pk
	if string(pk.Payload) == "hello" {
		pkx.Payload = []byte("hello world")
		h.Log.Info().Str("client", cl.ID).Str("payload", string(pkx.Payload)).Msg("received modified packet from client")
	}

	return pkx, nil
}

func (h *ExampleHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	h.Log.Info().Str("client", cl.ID).Str("payload", string(pk.Payload)).Msg("published to client")
}
