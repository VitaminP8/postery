package memory

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/VitaminP8/postery/internal/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createUserContext(userID uint) context.Context {
	ctx := context.Background()
	return auth.WithUserID(ctx, userID)
}

func TestPostMemoryStorage_CreatePost(t *testing.T) {
	storage := NewPostMemoryStorage()

	t.Run("Success post creation", func(t *testing.T) {
		userID := 1
		ctx := createUserContext(uint(userID))

		title := "Test post"
		content := "Test content"

		post, err := storage.CreatePost(ctx, title, content)
		require.NoError(t, err)
		assert.NotEmpty(t, post.ID)
		assert.Equal(t, title, post.Title)
		assert.Equal(t, content, post.Content)
		assert.Equal(t, strconv.Itoa(userID), post.AuthorID)
		assert.False(t, post.CommentsDisabled)

		postFromStorage, err := storage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.Equal(t, postFromStorage.ID, post.ID)
	})

	t.Run("Error: no authorization", func(t *testing.T) {
		// Используем контекст без информации о пользователе
		ctx := context.Background()

		_, err := storage.CreatePost(ctx, "title", "content")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestPostMemoryStorage_GetPostById(t *testing.T) {
	storage := NewPostMemoryStorage()
	userID := 1
	ctx := createUserContext(uint(userID))

	// Создаем пост для тестирования
	post, err := storage.CreatePost(ctx, "Test Post", "test content")
	require.NoError(t, err)

	t.Run("Getting exists post", func(t *testing.T) {
		retrievedPost, err := storage.GetPostById(post.ID)

		require.NoError(t, err)
		assert.Equal(t, post.ID, retrievedPost.ID)
		assert.Equal(t, post.Title, retrievedPost.Title)
		assert.Equal(t, post.Content, retrievedPost.Content)
		assert.Equal(t, post.AuthorID, retrievedPost.AuthorID)
	})

	t.Run("Trying to get not exist post", func(t *testing.T) {
		_, err := storage.GetPostById("23425532")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestPostMemoryStorage_GetAllPosts(t *testing.T) {
	storage := NewPostMemoryStorage()
	userID := 1
	ctx := createUserContext(uint(userID))

	_, err := storage.CreatePost(ctx, "post 1", "content 1")
	require.NoError(t, err)
	_, err = storage.CreatePost(ctx, "post 2", "content 2")
	require.NoError(t, err)
	_, err = storage.CreatePost(ctx, "post 3", "content 3")
	require.NoError(t, err)

	t.Run("Get all posts", func(t *testing.T) {
		posts, err := storage.GetAllPosts()

		require.NoError(t, err)
		assert.Len(t, posts, 3)
	})
}

func TestPostMemoryStorage_DisableComment(t *testing.T) {
	storage := NewPostMemoryStorage()
	userID := 1
	ctx := createUserContext(uint(userID))

	post, err := storage.CreatePost(ctx, "Test Post", "test content")
	require.NoError(t, err)

	t.Run("Disable comments by author", func(t *testing.T) {
		err := storage.DisableComment(ctx, post.ID)
		require.NoError(t, err)

		updatedPost, err := storage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.True(t, updatedPost.CommentsDisabled)
	})

	t.Run("Disable comments by not author", func(t *testing.T) {
		otherUserCtx := createUserContext(2)

		err := storage.DisableComment(otherUserCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("Disable comment for not exists post", func(t *testing.T) {
		err := storage.DisableComment(ctx, "234234")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Disable comment by unauthorized user", func(t *testing.T) {
		unauthorizedCtx := context.Background()

		err := storage.DisableComment(unauthorizedCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestPostMemoryStorage_EnableComment(t *testing.T) {
	storage := NewPostMemoryStorage()
	userID := 1
	ctx := createUserContext(uint(userID))

	post, err := storage.CreatePost(ctx, "title", "content")
	require.NoError(t, err)

	err = storage.DisableComment(ctx, post.ID)
	require.NoError(t, err)

	t.Run("Enable comment by author", func(t *testing.T) {
		err := storage.EnableComment(ctx, post.ID)
		require.NoError(t, err)

		updatedPost, err := storage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.False(t, updatedPost.CommentsDisabled)
	})

	t.Run("Enable comment by not author", func(t *testing.T) {
		// Сначала отключаем комментарии
		err := storage.DisableComment(ctx, post.ID)
		require.NoError(t, err)

		otherUserCtx := createUserContext(2)

		err = storage.EnableComment(otherUserCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")
	})

	t.Run("Enable comment for not exists post", func(t *testing.T) {
		err := storage.EnableComment(ctx, "25325")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Enable comment by unauthorized user", func(t *testing.T) {
		unauthorizedCtx := context.Background()

		err := storage.EnableComment(unauthorizedCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestPostMemoryStorage_DeletePostById(t *testing.T) {
	storage := NewPostMemoryStorage()
	userID := 1
	ctx := createUserContext(uint(userID))

	post, err := storage.CreatePost(ctx, "title", "content")
	require.NoError(t, err)

	t.Run("Delete post by author", func(t *testing.T) {
		err := storage.DeletePostById(ctx, post.ID)
		require.NoError(t, err)

		_, err = storage.GetPostById(post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	// Создаем еще один пост для тестирования других случаев
	post, err = storage.CreatePost(ctx, "title 2", "content 2")
	require.NoError(t, err)

	t.Run("Delete post by not author", func(t *testing.T) {
		otherUserCtx := createUserContext(2)

		err := storage.DeletePostById(otherUserCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")

		// Проверяем, что пост не был удален
		_, err = storage.GetPostById(post.ID)
		assert.NoError(t, err)
	})

	t.Run("Delete not exist post", func(t *testing.T) {
		err := storage.DeletePostById(ctx, "345345")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete by unauthorized user", func(t *testing.T) {
		unauthorizedCtx := context.Background()

		err := storage.DeletePostById(unauthorizedCtx, post.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestPostMemoryStorage_ConcurrentOperations(t *testing.T) {
	storage := NewPostMemoryStorage()

	t.Run("Concurrent post creation", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				userID := idx + 1
				ctx := createUserContext(uint(userID))

				title := "Post " + strconv.Itoa(idx)
				content := "Content " + strconv.Itoa(idx)

				post, err := storage.CreatePost(ctx, title, content)
				assert.NoError(t, err)
				assert.NotEmpty(t, post.ID)
				assert.Equal(t, strconv.Itoa(userID), post.AuthorID)
			}(i)
		}

		wg.Wait()

		// Проверяем, что все посты были созданы
		allPosts, err := storage.GetAllPosts()
		require.NoError(t, err)
		assert.Len(t, allPosts, numGoroutines)
	})

	t.Run("Concurrent read and change", func(t *testing.T) {
		userID := 100
		ctx := createUserContext(uint(userID))
		post, err := storage.CreatePost(ctx, "test post", "test content")
		require.NoError(t, err)

		var wg sync.WaitGroup
		numReaders := 5
		numWriters := 5

		// Запускаем читателей
		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 10; j++ {
					_, err := storage.GetPostById(post.ID)
					assert.NoError(t, err)
				}
			}()
		}

		// Запускаем писателей, которые включают/отключают комментарии
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				writerCtx := createUserContext(uint(userID))

				if idx%2 == 0 {
					err := storage.DisableComment(writerCtx, post.ID)
					assert.NoError(t, err)
				} else {
					err := storage.EnableComment(writerCtx, post.ID)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		finalPost, err := storage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.Equal(t, post.ID, finalPost.ID)
	})

	t.Run("Multiple authors create posts, then multiple readers view all posts", func(t *testing.T) {
		numAuthors := 5
		postIDs := make([]string, 0, numAuthors)
		var wg sync.WaitGroup
		var postIDsMutex sync.Mutex

		// Несколько авторов создают посты
		for i := 0; i < numAuthors; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Каждый автор имеет свой ID
				authorID := idx + 101
				ctx := createUserContext(uint(authorID))

				title := "Author " + strconv.Itoa(authorID) + " Post"
				content := "Content from author " + strconv.Itoa(authorID)

				post, err := storage.CreatePost(ctx, title, content)
				assert.NoError(t, err)
				assert.Equal(t, strconv.Itoa(authorID), post.AuthorID)

				// Сохраняем ID созданного поста
				postIDsMutex.Lock()
				postIDs = append(postIDs, post.ID)
				postIDsMutex.Unlock()

				// Имитируем задержку между созданиями постов
				time.Sleep(100 * time.Millisecond)
			}(i)
		}

		wg.Wait()

		// Проверяем, что все посты были созданы
		for _, postID := range postIDs {
			_, err := storage.GetPostById(postID)
			assert.NoError(t, err, "Пост с ID %s должен существовать", postID)
		}
		assert.Equal(t, numAuthors, len(postIDs))

		// Несколько читателей просматривают все посты
		numReaders := 5
		var readWg sync.WaitGroup

		for i := 0; i < numReaders; i++ {
			readWg.Add(1)
			go func(readerIdx int) {
				defer readWg.Done()

				// Каждый читатель просматривает все созданные посты
				for _, postID := range postIDs {
					post, err := storage.GetPostById(postID)
					assert.NoError(t, err)
					assert.NotEmpty(t, post.Title)
					assert.NotEmpty(t, post.Content)
					assert.NotEmpty(t, post.AuthorID)

					// Имитируем "чтение" поста
					time.Sleep(5 * time.Millisecond)
				}
			}(i)
		}

		readWg.Wait()
	})
}
