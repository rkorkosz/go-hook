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

type pubSub interface {
	Publisher
	Subscriber
}

type servers interface {
	Iter() chan string
}
