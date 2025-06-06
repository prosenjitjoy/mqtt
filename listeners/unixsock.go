// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: jason@zgwit.com

package listeners

import (
	"net"
	"os"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

const TypeUnix = "unix"

// UnixSock is a listener for establishing client connections on basic UnixSock protocol.
type UnixSock struct {
	sync.RWMutex
	id      string          // the internal id of the listener.
	address string          // the network address to bind to.
	config  Config          // configuration values for the listener
	listen  net.Listener    // a net.Listener which will listen for new clients.
	log     *zerolog.Logger // server logger
	end     uint32          // ensure the close methods are only called once.
}

// NewUnixSock initializes and returns a new UnixSock listener, listening on an address.
func NewUnixSock(config Config) *UnixSock {
	return &UnixSock{
		id:      config.ID,
		address: config.Address,
		config:  config,
	}
}

// ID returns the id of the listener.
func (l *UnixSock) ID() string {
	return l.id
}

// Address returns the address of the listener.
func (l *UnixSock) Address() string {
	return l.address
}

// Protocol returns the address of the listener.
func (l *UnixSock) Protocol() string {
	return "unix"
}

// Init initializes the listener.
func (l *UnixSock) Init(log *zerolog.Logger) error {
	l.log = log

	var err error
	_ = os.Remove(l.address)
	l.listen, err = net.Listen("unix", l.address)
	return err
}

// Serve starts waiting for new UnixSock connections, and calls the establish
// connection callback for any received.
func (l *UnixSock) Serve(establish EstablishFn) {
	for {
		if atomic.LoadUint32(&l.end) == 1 {
			return
		}

		conn, err := l.listen.Accept()
		if err != nil {
			return
		}

		if atomic.LoadUint32(&l.end) == 0 {
			go func() {
				err = establish(l.id, conn)
				if err != nil {
					l.log.Warn().Err(err).Send()
				}
			}()
		}
	}
}

// Close closes the listener and any client connections.
func (l *UnixSock) Close(closeClients CloseFn) {
	l.Lock()
	defer l.Unlock()

	if atomic.CompareAndSwapUint32(&l.end, 0, 1) {
		closeClients(l.id)
	}

	if l.listen != nil {
		err := l.listen.Close()
		if err != nil {
			return
		}
	}
}
