package troupe

import "errors"

// ErrNoEmbeddedBroker is returned when Embed is called without importing broker/.
var ErrNoEmbeddedBroker = errors.New("troupe: embedded broker not available — import github.com/dpopsuev/troupe/broker")

// EmbeddedBrokerFactory is set by the broker package at init time.
// Avoids import cycle: troupe → broker → troupe.
var EmbeddedBrokerFactory func() (Broker, error)

func embeddedBroker() (Broker, error) {
	if EmbeddedBrokerFactory == nil {
		return nil, ErrNoEmbeddedBroker
	}
	return EmbeddedBrokerFactory()
}
