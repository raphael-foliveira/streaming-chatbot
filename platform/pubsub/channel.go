package pubsub

import "sync"

type Channel[T any] struct {
	subscriptions map[string][]chan T
	mu            sync.RWMutex
}

func NewChannel[T any](subscriptions map[string][]chan T) *Channel[T] {
	return &Channel[T]{
		subscriptions: subscriptions,
	}
}

func (p *Channel[T]) Subscribe(topic string) (chan T, func(), error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.subscriptions[topic]; !ok {
		p.subscriptions[topic] = []chan T{}
	}
	newChannel := make(chan T, 1000)
	p.subscriptions[topic] = append(p.subscriptions[topic], newChannel)
	return newChannel, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for i, channel := range p.subscriptions[topic] {
			if channel == newChannel {
				p.subscriptions[topic] = append(p.subscriptions[topic][:i], p.subscriptions[topic][i+1:]...)
				break
			}
		}
		close(newChannel)
	}, nil
}

func (p *Channel[T]) Publish(topic string, message T) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, channel := range p.subscriptions[topic] {
		select {
		case channel <- message:
		default:
		}
	}
	return nil
}
