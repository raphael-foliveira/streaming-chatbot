package domain

type PubSub[T any] interface {
	Subscribe(topic string) (chan T, func(), error)
	Publish(topic string, message T) error
}
