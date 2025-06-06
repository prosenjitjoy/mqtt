// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prosenjitjoy/mqtt"
	"github.com/prosenjitjoy/mqtt/hooks/auth"
	"github.com/prosenjitjoy/mqtt/listeners"
)

func main() {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	// An example of configuring various server options...
	options := &mqtt.Options{
		// InflightTTL: 60 * 15, // Set an example custom 15-min TTL for inflight messages
	}

	server := mqtt.New(options)

	// For security reasons, the default implementation disallows all connections.
	// If you want to allow all connections, you must specifically allow it.
	err := server.AddHook(new(auth.AllowHook), nil)
	if err != nil {
		log.Fatal(err)
	}

	tcp := listeners.NewTCP(listeners.Config{
		ID:      "t1",
		Address: ":1883",
	})
	err = server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-done
	server.Log.Warn().Msg("caught signal, stopping...")
	server.Close()
	server.Log.Info().Msg("main.go finished")
}
