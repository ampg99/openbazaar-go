package net

import (
	"context"
	"io"
	"time"

	"gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	pstore "gx/ipfs/QmXZSd1qR5BxZkPyuwfT5jpqQFScZccoZvDneXsKzCNHWX/go-libp2p-peerstore"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	iconn "gx/ipfs/QmcXRdAP5bCCm51X7XfDUrQ8Q9PsrKbU75pyvB18iuKob5/go-libp2p-interface-conn"
	ma "gx/ipfs/QmcyqRMCAXVtYPS4DiBrA7sezL9rRGfW8Ctx7cywL4TXJj/go-multiaddr"
	peer "gx/ipfs/QmdS9KpbDyPrieswibZhkod1oXqRwZJrUPzxCofAMWpFGq/go-libp2p-peer"
)

// MessageSizeMax is a soft (recommended) maximum for network messages.
// One can write more, as the interface is a stream. But it is useful
// to bunch it up into multiple read/writes when the whole message is
// a single, large serialized object.
const MessageSizeMax = 2 << 22 // 4MB

// Stream represents a bidirectional channel between two agents in
// the IPFS network. "agent" is as granular as desired, potentially
// being a "request -> reply" pair, or whole protocols.
// Streams are backed by SPDY streams underneath the hood.
type Stream interface {
	io.Reader
	io.Writer
	io.Closer

	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error

	Protocol() protocol.ID
	SetProtocol(protocol.ID)

	// Conn returns the connection this stream is part of.
	Conn() Conn
}

// StreamHandler is the type of function used to listen for
// streams opened by the remote side.
type StreamHandler func(Stream)

// Conn is a connection to a remote peer. It multiplexes streams.
// Usually there is no need to use a Conn directly, but it may
// be useful to get information about the peer on the other side:
//  stream.Conn().RemotePeer()
type Conn interface {
	iconn.PeerConn

	// NewStream constructs a new Stream over this conn.
	NewStream() (Stream, error)

	// GetStreams returns all open streams over this conn.
	GetStreams() ([]Stream, error)
}

// ConnHandler is the type of function used to listen for
// connections opened by the remote side.
type ConnHandler func(Conn)

// Network is the interface used to connect to the outside world.
// It dials and listens for connections. it uses a Swarm to pool
// connnections (see swarm pkg, and peerstream.Swarm). Connections
// are encrypted with a TLS-like protocol.
type Network interface {
	Dialer
	io.Closer

	// SetStreamHandler sets the handler for new streams opened by the
	// remote side. This operation is threadsafe.
	SetStreamHandler(StreamHandler)

	// SetConnHandler sets the handler for new connections opened by the
	// remote side. This operation is threadsafe.
	SetConnHandler(ConnHandler)

	// NewStream returns a new stream to given peer p.
	// If there is no connection to p, attempts to create one.
	NewStream(context.Context, peer.ID) (Stream, error)

	// Listen tells the network to start listening on given multiaddrs.
	Listen(...ma.Multiaddr) error

	// ListenAddresses returns a list of addresses at which this network listens.
	ListenAddresses() []ma.Multiaddr

	// InterfaceListenAddresses returns a list of addresses at which this network
	// listens. It expands "any interface" addresses (/ip4/0.0.0.0, /ip6/::) to
	// use the known local interfaces.
	InterfaceListenAddresses() ([]ma.Multiaddr, error)

	// Process returns the network's Process
	Process() goprocess.Process
}

// Dialer represents a service that can dial out to peers
// (this is usually just a Network, but other services may not need the whole
// stack, and thus it becomes easier to mock)
type Dialer interface {

	// Peerstore returns the internal peerstore
	// This is useful to tell the dialer about a new address for a peer.
	// Or use one of the public keys found out over the network.
	Peerstore() pstore.Peerstore

	// LocalPeer returns the local peer associated with this network
	LocalPeer() peer.ID

	// DialPeer establishes a connection to a given peer
	DialPeer(context.Context, peer.ID) (Conn, error)

	// ClosePeer closes the connection to a given peer
	ClosePeer(peer.ID) error

	// Connectedness returns a state signaling connection capabilities
	Connectedness(peer.ID) Connectedness

	// Peers returns the peers connected
	Peers() []peer.ID

	// Conns returns the connections in this Netowrk
	Conns() []Conn

	// ConnsToPeer returns the connections in this Netowrk for given peer.
	ConnsToPeer(p peer.ID) []Conn

	// Notify/StopNotify register and unregister a notifiee for signals
	Notify(Notifiee)
	StopNotify(Notifiee)
}

// Connectedness signals the capacity for a connection with a given node.
// It is used to signal to services and other peers whether a node is reachable.
type Connectedness int

const (
	// NotConnected means no connection to peer, and no extra information (default)
	NotConnected Connectedness = iota

	// Connected means has an open, live connection to peer
	Connected

	// CanConnect means recently connected to peer, terminated gracefully
	CanConnect

	// CannotConnect means recently attempted connecting but failed to connect.
	// (should signal "made effort, failed")
	CannotConnect
)

// Notifiee is an interface for an object wishing to receive
// notifications from a Network.
type Notifiee interface {
	Listen(Network, ma.Multiaddr)      // called when network starts listening on an addr
	ListenClose(Network, ma.Multiaddr) // called when network starts listening on an addr
	Connected(Network, Conn)           // called when a connection opened
	Disconnected(Network, Conn)        // called when a connection closed
	OpenedStream(Network, Stream)      // called when a stream opened
	ClosedStream(Network, Stream)      // called when a stream closed

	// TODO
	// PeerConnected(Network, peer.ID)    // called when a peer connected
	// PeerDisconnected(Network, peer.ID) // called when a peer disconnected
}
