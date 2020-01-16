package pubsub

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Broker the brocker related meta data
type Broker struct {
	subscribers Subscribers
	slock       sync.RWMutex
	topics      map[string]Subscribers
	tlock       sync.RWMutex
}

// NewBroker create new broker
func NewBroker() *Broker {
	return &Broker{
		subscribers: Subscribers{},
		slock:       sync.RWMutex{},
		topics:      map[string]Subscribers{},
		tlock:       sync.RWMutex{},
	}
}

// Attach Create a new subscriber and register it into our main broker
func (b *Broker) Attach() (*Subscriber, error) {
	b.slock.Lock()
	defer b.slock.Unlock()
	id := make([]byte, 50)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	s := &Subscriber{
		id:        hex.EncodeToString(id),
		messages:  make(chan *Message),
		createdAt: time.Now().UnixNano(),
		destroyed: false,
		lock:      &sync.RWMutex{},
		topics:    map[string]bool{},
	}
	b.subscribers[s.id] = s
	return s, nil
}

// Subscribe subscribes the speicifed subscriber "s" to the specified list of topic(s)
func (b *Broker) Subscribe(s *Subscriber, topics ...string) {
	b.tlock.Lock()
	defer b.tlock.Unlock()
	for _, topic := range topics {
		if nil == b.topics[topic] {
			b.topics[topic] = Subscribers{}
		}
		s.topics[topic] = true
		b.topics[topic][s.id] = s
	}
}

// Unsubscribe unsubscribes the specified subscriber from the specified topic(s)
func (b *Broker) Unsubscribe(s *Subscriber, topics ...string) {
	b.tlock.Lock()
	defer b.tlock.Unlock()
	for _, topic := range topics {
		if nil == b.topics[topic] {
			continue
		}
		delete(b.topics[topic], s.id)
		delete(s.topics, topic)
	}
}

// Detach remove the specified subscriber from the broker
func (b *Broker) Detach(s *Subscriber) {
	b.slock.Lock()
	defer b.slock.Unlock()
	s.destroy()
	b.Unsubscribe(s, s.GetTopics()...)
	delete(b.subscribers, s.id)
}

// Broadcast broadcast the specified payload to all the topic(s) subscribers
func (b *Broker) Broadcast(payload interface{}, topics ...string) {
	for _, topic := range topics {
		if b.Subscribers(topic) < 1 {
			continue
		}
		for _, s := range b.topics[topic] {
			m := &Message{
				topic:     topic,
				payload:   payload,
				createdAt: time.Now().UnixNano(),
			}
			go (func(s *Subscriber) {
				s.Signal(m)
			})(s)
		}
	}
}

// Subscribers get the subscribers count
func (b *Broker) Subscribers(topic string) int {
	b.tlock.RLock()
	defer b.tlock.RUnlock()
	return len(b.topics[topic])
}

// GetTopics returns a slice of topics
func (b *Broker) GetTopics() []string {
	b.tlock.RLock()
	defer b.tlock.RUnlock()
	topics := []string{}
	for topic := range b.topics {
		topics = append(topics, topic)
	}
	return topics
}
