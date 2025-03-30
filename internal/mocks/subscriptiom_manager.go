package mocks

import (
	"sync"
	"time"

	"github.com/VitaminP8/postery/graph/model"
)

type MockSubscriptionManager struct {
	mu            sync.Mutex
	subs          map[string][]chan *model.Comment // postID -> список каналов подписчиков
	notifications map[string][]*model.Comment      // Для отслеживания в тестах
}

func NewMockSubscriptionManager() *MockSubscriptionManager {
	return &MockSubscriptionManager{
		subs:          make(map[string][]chan *model.Comment),
		notifications: make(map[string][]*model.Comment),
	}
}

func (m *MockSubscriptionManager) Subscribe(postID string) (<-chan *model.Comment, func()) {
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

func (m *MockSubscriptionManager) Publish(postID string, comment *model.Comment) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, sub := range m.subs[postID] {
		select {
		case sub <- comment:
		case <-time.After(500 * time.Millisecond):
		}
	}

	// Сохраняем уведомление для тестирования
	if _, ok := m.notifications[postID]; !ok {
		m.notifications[postID] = make([]*model.Comment, 0)
	}
	m.notifications[postID] = append(m.notifications[postID], comment)
}

// GetNotificationsForPost - вспомогательный метод для тестирования,
// возвращает все уведомления для конкретного поста
func (m *MockSubscriptionManager) GetNotificationsForPost(postID string) []*model.Comment {
	m.mu.Lock()
	defer m.mu.Unlock()

	comments, ok := m.notifications[postID]
	if !ok {
		return nil
	}
	return comments
}
