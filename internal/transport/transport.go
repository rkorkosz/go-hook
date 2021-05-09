package transport

import (
	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

type Subscriber interface {
	Subscribe(id, topic string) (pubsub.DataChannel, error)
	Unsubscribe(id, topic string)
}

type Publisher interface {
	Publish(source, topic string, data []byte)
}

type PubSub interface {
	Publisher
	Subscriber
}

type Servers interface {
	Iter() chan string
}
