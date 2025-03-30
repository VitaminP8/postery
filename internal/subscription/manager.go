package subscription

import (
	"sync"
	"time"

	"github.com/VitaminP8/postery/graph/model"
)

type SubscriptionManager struct {
	mu   sync.Mutex
	subs map[string][]chan *model.Comment // postID -> список каналов подписчиков
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subs: make(map[string][]chan *model.Comment),
	}
}

func (m *SubscriptionManager) Subscribe(postID string) (<-chan *model.Comment, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *model.Comment, 1) // Буфер 1, чтобы не блокировался писатель

	m.subs[postID] = append(m.subs[postID], ch)

	// функция для отписки
	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subscribers := m.subs[postID]
		for i, sub := range subscribers {
			if sub == ch {
				// Удаляем подписчика
				m.subs[postID] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}

	return ch, cancel
}

func (m *SubscriptionManager) Publish(postID string, comment *model.Comment) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sub := range m.subs[postID] {
		select {
		case sub <- comment:
		case <-time.After(500 * time.Millisecond):
			// Если канал заполнен, ждем короткое время
		}
	}
}
