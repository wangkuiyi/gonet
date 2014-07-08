# TODO with Gonet

- Named Pipe via `etcd`

  `etcd` supports TTL.  This enables clients to know channels on dead
  servers as dead.

  `gonet.OpenChan` should start a goroutine which monitors the
  existence of the name of a Gonet channel on etcd -- if it does not
  exists -- the server is dead -- close the client channel.

- Communication via HTTP.

  This would allow us to create a package like `net/rpc`, which make
  connections over an HTTP Mux.  So, with only one port, a process can
  make many Gonet channels.

- The Syntax of Go.

  There should be a function `gonet.Go`, which creates a process as
  easily as the `go` keyword creates goroutines.
