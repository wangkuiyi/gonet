package gonet

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

const (
	kMaxOpeningChannels = 5
)

type Chan struct {
	listener net.Listener
	connChan chan net.Conn
	readEnd  chan interface{}
}

var (
	namedChans map[string]*Chan // named channels will be registered here.
)

func init() {
	namedChans = make(map[string]*Chan)
}

// MakeChan creates a Gonet channel, which collects typed data from
// the network, and returns the read-end Go channel.  MakeChan is
// supposed to be called by the data reader process.  A made channel
// is named by the network address that the reader is listening on.
// From technical perspective, each channel is associated with a TCP
// listener and a connection acceptance goroutine.  Thus, writers can
// make network connections through this listener and feed data
// through these connections.  Note that the maximum number of
// concurrent connections is limited by the global variable
// kMaxOpeningChannels.  This limitation protects the reader/server
// process from being too busy in channel I/O.
func MakeChan(addr string, value interface{}) (
	<-chan interface{}, error) {

	if ch, ok := namedChans[addr]; ok {
		return ch.readEnd, nil
	}

	ln, e := net.Listen("tcp", addr)
	if e != nil {
		return nil, fmt.Errorf("Cannot listen on %s: %v", addr, e)
	}

	connChan := make(chan net.Conn)
	valueChan := make(chan interface{})

	for i := 0; i < kMaxOpeningChannels; i++ {
		go serveConn(connChan, value, valueChan)
	}

	go func() {
		for {
			conn, e := ln.Accept()
			if e != nil {
				log.Printf("gonet: accept goroutine exits due to Accept error")
				return
			} else {
				connChan <- conn
			}
		}
	}()

	namedChans[addr] = &Chan{ln, connChan, valueChan}
	return valueChan, nil
}

func serveConn(connChan chan net.Conn, value interface{}, r chan interface{}) {
	for {
		if conn, ok := <-connChan; ok {
			transcode(conn, value, r)
		} else {
			log.Printf("gonet: serveConn goroutine exits due to CloseChan")
			return
		}
	}
}

func transcode(conn net.Conn, value interface{}, valueChan chan interface{}) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("gonet: Read-end closed. Close server connection.")
			conn.Close()
		}
	}()

	dec := gob.NewDecoder(conn)

	for {
		if e := dec.Decode(value); e != nil {
			log.Printf("gonet: decoding: %v. Close server connection", e)
			// Note we close only network connection but not the
			// read-side channel here, because there might exist other
			// sending clients.
			conn.Close()
			return
		}
		valueChan <- value
	}
}

// CloseChan closes a named channel if it is created, or does nothing
// otherwise.  The reader/server process which called MakeChan to
// create the channel is expected to call CloseChan when it does no
// longer want to serve this channel.  This frees the network address
// (port) that the process is listening on.
func CloseChan(addr string) {
	if ch, ok := namedChans[addr]; ok {
		log.Printf("Close channel named %s", addr)
		ch.listener.Close()
		close(ch.connChan)
		close(ch.readEnd)
		delete(namedChans, addr)
	}
}

// OpenChan opens the write-end channle of a Gonet channel.  It is
// expected to be called by a process which wants to feed data into
// the Gonet channel.  After the feeding, the caller process simply
// close this write-end Go channel to free all related resource.
func OpenChan(addr string) (chan<- interface{}, error) {
	conn, e := net.Dial("tcp", addr)
	if e != nil {
		return nil, fmt.Errorf("Cannot dial %s: %v", addr, e)
	}

	enc := gob.NewEncoder(conn)

	valueChan := make(chan interface{})

	go func(addr string) {
		for {
			v, ok := <-valueChan
			if !ok {
				log.Printf("gonet: write-end to %s closed. "+
					"Close client connection.", addr)
				conn.Close()
				return
			}

			if e := enc.Encode(v); e != nil {
				log.Printf("gonet: encoding: %v. Close client connection", e)
				conn.Close()
				close(valueChan)
			}
		}
	}(addr)

	return valueChan, nil
}
