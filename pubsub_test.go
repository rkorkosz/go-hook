package main

import (
	"testing"
)

func Test_Subscribe(t *testing.T) {
	s := newStage(t)
	s.pubsub_is_created().
		user_is_subscribed_to_a_topic("user", "topic").
		subscribe_was_succesful()
}

func Test_Publish(t *testing.T) {
	s := newStage(t)
	s.pubsub_is_created().
		user_is_subscribed_to_a_topic("user", "topic").
		subscribe_was_succesful().
		message_is_published_in_a_topic("user", "topic", `{"a": 1}`).
		message_was_succesfully_received()
}

type stage struct {
	t        testing.TB
	pubSub   *PubSub
	topics   []string
	users    []string
	chans    []DataChannel
	messages []string
}

func newStage(t *testing.T) *stage {
	return &stage{t: t}
}

func (s *stage) pubsub_is_created() *stage {
	s.pubSub = NewPubSub(1)
	s.t.Cleanup(s.pubSub.Close)
	return s
}

func (s *stage) user_is_subscribed_to_a_topic(user, topic string) *stage {
	ch := make(DataChannel, 1)
	s.chans = append(s.chans, ch)
	s.pubSub.Subscribe(user, topic, ch)
	s.topics = append(s.topics, topic)
	s.users = append(s.users, user)
	return s
}

func (s *stage) subscribe_was_succesful() *stage {
	for _, topic := range s.topics {
		_, ok := s.pubSub.subs[topic]
		if !ok {
			s.t.Error("subscribe did not succeeded")
		}
		for _, user := range s.users {
			ch, ok := s.pubSub.subs[topic][user]
			if !ok {
				s.t.Error("subscribe did not succeeded")
			}
			if ch == nil {
				s.t.Error("channel was not set")
			}
		}
	}
	return s
}

func (s *stage) message_is_published_in_a_topic(source, topic, msg string) *stage {
	s.messages = append(s.messages, msg)
	s.pubSub.Publish(source, topic, []byte(msg))
	return s
}

func (s *stage) message_was_succesfully_received() *stage {
	for _, msg := range s.messages {
		for _, ch := range s.chans {
			m := <-ch
			if string(m.Data) != msg {
				s.t.Error("message was not received")
			}
		}
	}
	return s
}
