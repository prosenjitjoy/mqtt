// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package main

import (
	"flag"
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

	// You can also run from top-level server.go folder:
	// go run examples/auth/encoded/main.go --path=examples/auth/encoded/auth.yaml
	// go run examples/auth/encoded/main.go --path=examples/auth/encoded/auth.json
	path := flag.String("path", "auth.yaml", "path to data auth file")
	flag.Parse()

	// Get ledger from yaml file
	data, err := os.ReadFile(*path)
	if err != nil {
		log.Fatal(err)
	}

	server := mqtt.New(nil)
	err = server.AddHook(new(auth.Hook), &auth.Options{
		Data: data, // build ledger from byte slice, yaml or json
	})
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
