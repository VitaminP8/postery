package subscription

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionManager_Subscribe(t *testing.T) {
	t.Run("Should create a subscription channel", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		ch, cancel := manager.Subscribe(postID)
		assert.NotNil(t, ch)
		assert.NotNil(t, cancel)

		manager.mu.Lock()
		subscribers, exists := manager.subs[postID]
		manager.mu.Unlock()
		assert.True(t, exists)
		assert.Len(t, subscribers, 1)

		// Вызываем отмену подписки
		cancel()

		manager.mu.Lock()
		subscribers, exists = manager.subs[postID]
		manager.mu.Unlock()
		assert.True(t, exists)
		assert.Len(t, subscribers, 0)
	})

	t.Run("Multiple subscriptions to the same post", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		// Создаем 3 подписки
		_, cancel1 := manager.Subscribe(postID)
		_, cancel2 := manager.Subscribe(postID)
		_, cancel3 := manager.Subscribe(postID)

		manager.mu.Lock()
		subscribers, exists := manager.subs[postID]
		manager.mu.Unlock()
		assert.True(t, exists)
		assert.Len(t, subscribers, 3)

		// Отменяем вторую подписку
		cancel2()

		manager.mu.Lock()
		subscribers, exists = manager.subs[postID]
		manager.mu.Unlock()
		assert.True(t, exists)
		assert.Len(t, subscribers, 2)

		// Отменяем остальные подписки
		cancel1()
		cancel3()

		manager.mu.Lock()
		subscribers, exists = manager.subs[postID]
		manager.mu.Unlock()
		assert.True(t, exists)
		assert.Len(t, subscribers, 0)
	})

	t.Run("Subscriptions to different posts", func(t *testing.T) {
		manager := NewSubscriptionManager()

		// Создаем подписки на разные посты
		_, cancel1 := manager.Subscribe("post1")
		_, cancel2 := manager.Subscribe("post2")
		_, cancel3 := manager.Subscribe("post3")

		manager.mu.Lock()
		assert.Len(t, manager.subs, 3)
		manager.mu.Unlock()

		// Отменяем все подписки
		cancel1()
		cancel2()
		cancel3()

		manager.mu.Lock()
		assert.Len(t, manager.subs["post1"], 0)
		assert.Len(t, manager.subs["post2"], 0)
		assert.Len(t, manager.subs["post3"], 0)
		manager.mu.Unlock()
	})
}

func TestSubscriptionManager_Publish(t *testing.T) {
	t.Run("Should send comment to subscribers", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		ch, cancel := manager.Subscribe(postID)
		defer cancel()

		comment := &model.Comment{
			ID:        "456",
			PostID:    postID,
			Content:   "Test comment",
			AuthorID:  "789",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		// Публикуем комментарий
		manager.Publish(postID, comment)

		// Проверяем, что комментарий получен
		select {
		case receivedComment := <-ch:
			assert.Equal(t, comment, receivedComment)
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for comment")
		}
	})

	t.Run("Multiple subscribers should all receive the comment", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		ch1, cancel1 := manager.Subscribe(postID)
		ch2, cancel2 := manager.Subscribe(postID)
		ch3, cancel3 := manager.Subscribe(postID)
		defer cancel1()
		defer cancel2()
		defer cancel3()

		comment := &model.Comment{
			ID:        "456",
			PostID:    postID,
			Content:   "Test comment",
			AuthorID:  "789",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		manager.Publish(postID, comment)

		for i, ch := range []<-chan *model.Comment{ch1, ch2, ch3} {
			select {
			case receivedComment := <-ch:
				assert.Equal(t, comment, receivedComment, "Subscriber %d did not receive correct comment", i+1)
			case <-time.After(time.Second):
				t.Fatalf("Subscriber %d timed out waiting for comment", i+1)
			}
		}
	})

	t.Run("Should only send to subscribers of the specific post", func(t *testing.T) {
		manager := NewSubscriptionManager()

		ch1, cancel1 := manager.Subscribe("post1")
		ch2, cancel2 := manager.Subscribe("post2")
		defer cancel1()
		defer cancel2()

		comment := &model.Comment{
			ID:        "456",
			PostID:    "post1",
			Content:   "Test comment",
			AuthorID:  "789",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		manager.Publish("post1", comment)

		select {
		case receivedComment := <-ch1:
			assert.Equal(t, comment, receivedComment)
		case <-time.After(time.Second):
			t.Fatal("Subscriber of post1 timed out waiting for comment")
		}

		select {
		case <-ch2:
			t.Fatal("Subscriber of post2 should not receive the comment")
		case <-time.After(100 * time.Millisecond):
		}
	})

	t.Run("Publishing to a post with no subscribers should not panic", func(t *testing.T) {
		manager := NewSubscriptionManager()

		comment := &model.Comment{
			ID:        "456",
			PostID:    "post1",
			Content:   "Test comment",
			AuthorID:  "789",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		assert.NotPanics(t, func() {
			manager.Publish("post1", comment)
		})
	})
}

func TestSubscriptionManager_Concurrent(t *testing.T) {
	t.Run("Concurrent subscriptions and publications", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		// Количество подписчиков и публикаций
		numSubscribers := 10
		numPublications := 5

		var wg sync.WaitGroup

		// Создаем подписчиков
		chans := make([]<-chan *model.Comment, numSubscribers)
		cancels := make([]func(), numSubscribers)

		// Счетчик полученных комментариев для каждого подписчика
		received := make([]int, numSubscribers)

		var mu sync.Mutex

		for i := 0; i < numSubscribers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ch, cancel := manager.Subscribe(postID)
				chans[idx] = ch
				cancels[idx] = cancel

				// Запускаем горутину для чтения из канала
				go func(idx int, ch <-chan *model.Comment) {
					for comment := range ch {
						require.Equal(t, postID, comment.PostID)
						mu.Lock()
						received[idx]++
						mu.Unlock()
					}
				}(idx, ch)
			}(i)
		}

		// Ожидаем завершения подписок
		wg.Wait()

		// Публикуем комментарии
		for i := 0; i < numPublications; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				comment := &model.Comment{
					ID:        strconv.Itoa(1000 + idx),
					PostID:    postID,
					Content:   "Concurrent test comment " + strconv.Itoa(idx),
					AuthorID:  "789",
					CreatedAt: time.Now().Format(time.RFC3339),
				}
				manager.Publish(postID, comment)
			}(i)
		}

		wg.Wait()

		// Даем время на обработку всех сообщений
		time.Sleep(1000 * time.Millisecond)

		// Отменяем все подписки
		for _, cancel := range cancels {
			cancel()
		}

		// Проверяем, что все подписчики получили все публикации
		mu.Lock()
		for i := 0; i < numSubscribers; i++ {
			assert.Equal(t, numPublications, received[i], "Subscriber %d did not receive all publications", i)
		}
		mu.Unlock()
	})

	t.Run("Concurrent subscribes and unsubscribes", func(t *testing.T) {
		manager := NewSubscriptionManager()
		postID := "123"

		var wg sync.WaitGroup
		numOperations := 100

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Подписываемся
				ch, cancel := manager.Subscribe(postID)

				// Небольшая задержка
				time.Sleep(5 * time.Millisecond)

				// Отписываемся
				cancel()

				// Проверяем, что канал закрыт
				_, ok := <-ch
				assert.False(t, ok, "Channel should be closed after cancel")
			}()
		}

		wg.Wait()

		// Проверяем, что все подписки были корректно удалены
		manager.mu.Lock()
		assert.Len(t, manager.subs[postID], 0)
		manager.mu.Unlock()
	})
}
