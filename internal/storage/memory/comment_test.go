package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentMemoryStorage_CreateComment(t *testing.T) {
	postStorage := mocks.NewMockPostStorage()
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentMemoryStorage(postStorage, subscriptionManager)

	// Создаем пост для тестирования
	ctx := createUserContext(uint(1))
	post, err := postStorage.CreatePost(ctx, "Test Post", "Test Content")
	require.NoError(t, err)

	t.Run("Successful comment creation", func(t *testing.T) {
		content := "Test Comment"
		comment, err := commentStorage.CreateComment(ctx, post.ID, "", content)

		require.NoError(t, err)
		assert.NotEmpty(t, comment.ID)
		assert.Equal(t, post.ID, comment.PostID)
		assert.Equal(t, content, comment.Content)
		assert.Equal(t, "1", comment.AuthorID)
		assert.Nil(t, comment.ParentID)
		assert.False(t, comment.HasReplies)
		assert.Empty(t, comment.Children)

		// Проверяем, что отправлено уведомление
		notifications := subscriptionManager.GetNotificationsForPost(post.ID)
		assert.Len(t, notifications, 1)
		assert.Equal(t, comment.ID, notifications[0].ID)
	})

	t.Run("Creating nested comment", func(t *testing.T) {
		// Сначала создаем родительский комментарий
		parentComment, err := commentStorage.CreateComment(ctx, post.ID, "", "Parent Comment")
		require.NoError(t, err)

		// Теперь создаем дочерний комментарий
		childContent := "Child Comment"
		childComment, err := commentStorage.CreateComment(ctx, post.ID, parentComment.ID, childContent)

		require.NoError(t, err)
		assert.NotEmpty(t, childComment.ID)
		assert.Equal(t, post.ID, childComment.PostID)
		assert.Equal(t, childContent, childComment.Content)
		assert.Equal(t, "1", childComment.AuthorID)
		assert.NotNil(t, childComment.ParentID)
		assert.Equal(t, parentComment.ID, *childComment.ParentID)

		// Проверяем, что родительский комментарий помечен как имеющий ответы
		commentWithReplies, err := commentStorage.GetReplies(parentComment.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, commentWithReplies.Items, 1)
		assert.Equal(t, childComment.ID, commentWithReplies.Items[0].ID)
	})

	t.Run("Error when creating comment for disabled comments", func(t *testing.T) {
		// Отключаем комментарии у поста
		err = postStorage.DisableComment(ctx, post.ID)
		require.NoError(t, err)

		_, err = commentStorage.CreateComment(ctx, post.ID, "", "This should fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "comments are disabled")

		// Включаем комментарии обратно для следующих тестов
		err = postStorage.EnableComment(ctx, post.ID)
		require.NoError(t, err)
	})

	t.Run("Error when creating comment with empty content", func(t *testing.T) {
		_, err = commentStorage.CreateComment(ctx, post.ID, "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is too long or empty")
	})

	t.Run("Error when creating comment with too long content", func(t *testing.T) {
		longContent := ""
		for i := 0; i < 2001; i++ {
			longContent += "a"
		}

		_, err = commentStorage.CreateComment(ctx, post.ID, "", longContent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is too long or empty")
	})

	t.Run("Error when creating comment for non-existent post", func(t *testing.T) {
		_, err = commentStorage.CreateComment(ctx, "non-existent-post", "", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Error when creating comment with non-existent parent", func(t *testing.T) {
		_, err = commentStorage.CreateComment(ctx, post.ID, "non-existent-parent", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parent comment with ID")
	})

	t.Run("Error on unauthorized request", func(t *testing.T) {
		unauthorizedCtx := context.Background()
		_, err = commentStorage.CreateComment(unauthorizedCtx, post.ID, "", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestCommentMemoryStorage_GetComments(t *testing.T) {
	postStorage := mocks.NewMockPostStorage()
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentMemoryStorage(postStorage, subscriptionManager)

	// Создаем пост
	ctx := createUserContext(uint(1))
	post, err := postStorage.CreatePost(ctx, "Test Post", "Test Content")
	require.NoError(t, err)

	// Создаем корневые комментарии
	numRootComments := 5
	for i := 0; i < numRootComments; i++ {
		content := "Root Comment " + strconv.Itoa(i)
		_, err := commentStorage.CreateComment(ctx, post.ID, "", content)
		require.NoError(t, err)

		// Небольшая задержка для гарантии разного времени создания
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("Getting all comments without pagination", func(t *testing.T) {
		comments, err := commentStorage.GetComments(post.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, comments.Items, numRootComments)
		assert.False(t, comments.HasMore)

		// Проверяем, что комментарии отсортированы по времени создания
		for i := 1; i < len(comments.Items); i++ {
			assert.True(t, comments.Items[i-1].CreatedAt <= comments.Items[i].CreatedAt,
				"Комментарии должны быть отсортированы по возрастанию CreatedAt")
		}
	})

	t.Run("Getting comments with pagination", func(t *testing.T) {
		// Получаем первую страницу (2 комментария)
		page1, err := commentStorage.GetComments(post.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1.Items, 2)
		assert.True(t, page1.HasMore)
		assert.Equal(t, 2, page1.NextOffset)

		// Получаем вторую страницу (2 комментария)
		page2, err := commentStorage.GetComments(post.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2.Items, 2)
		assert.True(t, page2.HasMore)
		assert.Equal(t, 4, page2.NextOffset)

		// Получаем третью страницу (1 комментарий)
		page3, err := commentStorage.GetComments(post.ID, 2, 4)
		require.NoError(t, err)
		assert.Len(t, page3.Items, 1)
		assert.False(t, page3.HasMore)
		assert.Equal(t, 6, page3.NextOffset)

		// Проверяем, что все комментарии разные
		allCommentIDs := make(map[string]bool)
		for _, comment := range append(append(page1.Items, page2.Items...), page3.Items...) {
			allCommentIDs[comment.ID] = true
		}
		assert.Len(t, allCommentIDs, numRootComments)
	})

	t.Run("Getting comments for post with disabled comments", func(t *testing.T) {
		// Отключаем комментарии у поста
		err = postStorage.DisableComment(ctx, post.ID)
		require.NoError(t, err)

		comments, err := commentStorage.GetComments(post.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, comments.Items, 0)
		assert.False(t, comments.HasMore)

		// Включаем комментарии обратно
		err = postStorage.EnableComment(ctx, post.ID)
		require.NoError(t, err)
	})

	t.Run("Getting comments for non-existent post", func(t *testing.T) {
		_, err := commentStorage.GetComments("non-existent-post", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Getting comments with large offset", func(t *testing.T) {
		comments, err := commentStorage.GetComments(post.ID, 10, 100)
		require.NoError(t, err)
		assert.Len(t, comments.Items, 0)
		assert.False(t, comments.HasMore)
		assert.Equal(t, 100, comments.NextOffset)
	})
}

func TestCommentMemoryStorage_GetReplies(t *testing.T) {
	postStorage := mocks.NewMockPostStorage()
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentMemoryStorage(postStorage, subscriptionManager)

	// Создаем пост
	ctx := createUserContext(uint(1))
	post, err := postStorage.CreatePost(ctx, "Test Post", "Test Content")
	require.NoError(t, err)

	// Создаем родительский комментарий
	parentComment, err := commentStorage.CreateComment(ctx, post.ID, "", "Parent Comment")
	require.NoError(t, err)

	// Создаем дочерние комментарии
	numChildComments := 5
	for i := 0; i < numChildComments; i++ {
		content := "Child Comment " + strconv.Itoa(i)
		_, err := commentStorage.CreateComment(ctx, post.ID, parentComment.ID, content)
		require.NoError(t, err)

		// Небольшая задержка для гарантии разного времени создания
		time.Sleep(10 * time.Millisecond)
	}

	t.Run("Getting all child comments", func(t *testing.T) {
		replies, err := commentStorage.GetReplies(parentComment.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, replies.Items, numChildComments)
		assert.False(t, replies.HasMore)
		assert.Equal(t, 10, replies.NextOffset)

		// Проверяем, что комментарии отсортированы по времени создания
		for i := 1; i < len(replies.Items); i++ {
			assert.True(t, replies.Items[i-1].CreatedAt <= replies.Items[i].CreatedAt,
				"Комментарии должны быть отсортированы по возрастанию CreatedAt")
		}
	})

	t.Run("Getting child comments with pagination", func(t *testing.T) {
		// Получаем первую страницу (2 комментария)
		page1, err := commentStorage.GetReplies(parentComment.ID, 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1.Items, 2)
		assert.True(t, page1.HasMore)
		assert.Equal(t, 2, page1.NextOffset)

		// Получаем вторую страницу (2 комментария)
		page2, err := commentStorage.GetReplies(parentComment.ID, 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2.Items, 2)
		assert.True(t, page2.HasMore)
		assert.Equal(t, 4, page2.NextOffset)

		// Получаем третью страницу (1 комментарий)
		page3, err := commentStorage.GetReplies(parentComment.ID, 2, 4)
		require.NoError(t, err)
		assert.Len(t, page3.Items, 1)
		assert.False(t, page3.HasMore)
		assert.Equal(t, 6, page3.NextOffset)
	})

	t.Run("Getting child comments for non-existent parent", func(t *testing.T) {
		_, err := commentStorage.GetReplies("non-existent-parent", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "parent comment with ID")
	})

	t.Run("Getting child comments with large offset", func(t *testing.T) {
		replies, err := commentStorage.GetReplies(parentComment.ID, 10, 100)
		require.NoError(t, err)
		assert.Len(t, replies.Items, 0)
		assert.False(t, replies.HasMore)
		assert.Equal(t, 100, replies.NextOffset)
	})
}

func TestCommentMemoryStorage_ConcurrentOperations(t *testing.T) {
	postStorage := mocks.NewMockPostStorage()
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentMemoryStorage(postStorage, subscriptionManager)

	// Создаем пост
	ctx := createUserContext(uint(1))
	post, err := postStorage.CreatePost(ctx, "Test Post", "Test Content")
	require.NoError(t, err)

	t.Run("Concurrent comment creation", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := idx + 100
				ctx := createUserContext(uint(userID))

				content := "Concurrent Comment " + strconv.Itoa(idx)
				comment, err := commentStorage.CreateComment(ctx, post.ID, "", content)
				assert.NoError(t, err)
				assert.NotEmpty(t, comment.ID)
				assert.Equal(t, strconv.Itoa(userID), comment.AuthorID)
			}(i)
		}

		wg.Wait()

		// Проверяем, что все комментарии были созданы
		comments, err := commentStorage.GetComments(post.ID, 20, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(comments.Items), numGoroutines)
	})

	t.Run("Concurrent nested comment creation", func(t *testing.T) {
		// Создаем родительский комментарий
		parentComment, err := commentStorage.CreateComment(ctx, post.ID, "", "Parent for concurrent replies")
		require.NoError(t, err)

		var wg sync.WaitGroup
		numGoroutines := 5

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := idx + 200
				ctx := createUserContext(uint(userID))

				content := "Concurrent Reply " + strconv.Itoa(idx)
				comment, err := commentStorage.CreateComment(ctx, post.ID, parentComment.ID, content)
				assert.NoError(t, err)
				assert.NotEmpty(t, comment.ID)
				assert.Equal(t, strconv.Itoa(userID), comment.AuthorID)
				assert.Equal(t, parentComment.ID, *comment.ParentID)
			}(i)
		}

		wg.Wait()

		// Проверяем, что все дочерние комментарии были созданы
		replies, err := commentStorage.GetReplies(parentComment.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, replies.Items, numGoroutines)
	})

	t.Run("Concurrent reading and writing", func(t *testing.T) {
		// Создаем несколько комментариев для чтения
		for i := 0; i < 5; i++ {
			_, err := commentStorage.CreateComment(ctx, post.ID, "", "Comment for concurrent reading "+strconv.Itoa(i))
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		numReaders := 5
		numWriters := 10

		// Запускаем читателей
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 5; j++ {
					_, err := commentStorage.GetComments(post.ID, 10, 0)
					assert.NoError(t, err)
					time.Sleep(5 * time.Millisecond)
				}
			}()
		}

		// Запускаем писателей
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				userID := idx + 300
				ctx := createUserContext(uint(userID))

				content := "Write " + strconv.Itoa(idx)
				_, err := commentStorage.CreateComment(ctx, post.ID, "", content)
				assert.NoError(t, err)
				time.Sleep(10 * time.Millisecond)
			}(i)
		}

		wg.Wait()

		// Проверяем, что операции чтения и записи не конфликтовали
		comments, err := commentStorage.GetComments(post.ID, 50, 0)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(comments.Items), numWriters)
	})

	t.Run("Deeply nested comments", func(t *testing.T) {
		// Создаем цепочку вложенных комментариев
		depth := 5
		userID := 400

		var currentParentID string
		for i := 0; i < depth; i++ {
			ctx := createUserContext(uint(userID))
			content := "Nested Comment Level " + strconv.Itoa(i)

			var comment *model.Comment
			var err error

			if i == 0 {
				// Первый комментарий (корневой)
				comment, err = commentStorage.CreateComment(ctx, post.ID, "", content)
			} else {
				// Вложенный комментарий
				comment, err = commentStorage.CreateComment(ctx, post.ID, currentParentID, content)
			}

			require.NoError(t, err)
			currentParentID = comment.ID
		}

		// Проверяем, что можем получить комментарии на каждом уровне
		var parentID string
		currentLevel := 0

		rootComments, err := commentStorage.GetComments(post.ID, 100, 0)
		require.NoError(t, err)

		for _, comment := range rootComments.Items {
			if comment.Content == "Nested Comment Level 0" {
				parentID = comment.ID
				break
			}
		}

		require.NotEmpty(t, parentID, "Не найден корневой комментарий для цепочки")

		for currentLevel < depth-1 {
			currentLevel++
			replies, err := commentStorage.GetReplies(parentID, 10, 0)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(replies.Items), 1)

			found := false
			for _, reply := range replies.Items {
				if reply.Content == "Nested Comment Level "+strconv.Itoa(currentLevel) {
					parentID = reply.ID
					found = true
					break
				}
			}

			assert.True(t, found, "Не найден комментарий для уровня %d", currentLevel)
		}
	})
}
