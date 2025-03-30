package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite" // Импортируем драйвер SQLite
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Создает контекст с ID пользователя
func createUserContext(userID uint) context.Context {
	ctx := context.Background()
	return auth.WithUserID(ctx, userID)
}

// setupTestDB создает тестовую БД в памяти и выполняет миграции
func setupTestDB(t *testing.T) *gorm.DB {
	// Сохраняем оригинальное соединение (если оно есть)
	oldDB := GetDB()

	// Создаем SQLite в памяти
	db, err := gorm.Open("sqlite3", ":memory:")
	require.NoError(t, err, "Failed to connect to in-memory SQLite")

	// Включаем foreign keys в SQLite
	db.Exec("PRAGMA foreign_keys = ON")
	// Отключаем логирование запросов для тестов
	db.LogMode(false)
	// Выполняем миграцию схемы базы данных
	err = db.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}).Error
	require.NoError(t, err, "Failed to migrate database schema")
	// Устанавливаем SQLite в качестве глобальной DB
	InitDBWithConnection(db)

	return oldDB
}

// teardownTestDB восстанавливает оригинальную базу данных
func teardownTestDB(db *gorm.DB) {
	// Восстанавливаем оригинальное соединение
	InitDBWithConnection(db)
}

// createTestUser создает тестового пользователя и возвращает его ID
func createTestUser(t *testing.T) uint {
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	err := DB.Create(user).Error
	require.NoError(t, err, "Failed to create test user")

	return user.ID
}

// createTestPost создает тестовый пост и возвращает его ID
func createTestPost(t *testing.T, userID uint, title, content string) uint {
	post := &models.Post{
		Title:            title,
		Content:          content,
		UserID:           userID,
		CommentsDisabled: false,
	}

	err := DB.Create(post).Error
	require.NoError(t, err, "Failed to create test post")

	return post.ID
}

func TestPostPostgresStorage_CreatePost(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Success post creation", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		ctx := createUserContext(userID)

		testTitle := "Test Post Title"
		testContent := "This is a test post content"
		post, err := storage.CreatePost(ctx, testTitle, testContent)
		assert.NoError(t, err)
		assert.NotNil(t, post)
		assert.Equal(t, testTitle, post.Title)
		assert.Equal(t, testContent, post.Content)
		assert.Equal(t, fmt.Sprint(userID), post.AuthorID)
		assert.False(t, post.CommentsDisabled)

		// Проверяем, что пост действительно создался в БД
		var dbPost models.Post
		err = DB.First(&dbPost, post.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, testTitle, dbPost.Title)
		assert.Equal(t, testContent, dbPost.Content)
		assert.Equal(t, userID, dbPost.UserID)
	})

	t.Run("Error: no authorization", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		ctx := context.Background()
		post, err := storage.CreatePost(ctx, "Test Title", "Test Content")
		assert.Error(t, err)
		assert.Nil(t, post)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestPostPostgresStorage_GetPostById(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Getting exists post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		testTitle := "Test Post Title"
		testContent := "This is a test post content"
		postID := createTestPost(t, userID, testTitle, testContent)
		post, err := storage.GetPostById(fmt.Sprint(postID))
		assert.NoError(t, err)
		assert.NotNil(t, post)
		assert.Equal(t, fmt.Sprint(postID), post.ID)
		assert.Equal(t, testTitle, post.Title)
		assert.Equal(t, testContent, post.Content)
		assert.Equal(t, fmt.Sprint(userID), post.AuthorID)
	})

	t.Run("Trying to get not exist post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		post, err := storage.GetPostById("999")
		assert.Error(t, err)
		assert.Nil(t, post)
		assert.Contains(t, err.Error(), "could not get post by id")
	})
}

func TestPostPostgresStorage_GetAllPosts(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Get all posts", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		post1ID := createTestPost(t, userID, "Post 1", "Content 1")
		post2ID := createTestPost(t, userID, "Post 2", "Content 2")
		post3ID := createTestPost(t, userID, "Post 3", "Content 3")

		posts, err := storage.GetAllPosts()
		assert.NoError(t, err)
		assert.NotNil(t, posts)
		assert.Len(t, posts, 3)

		postMap := make(map[string]*model.Post)
		for _, p := range posts {
			postMap[p.ID] = p
		}

		// Проверяем наличие всех созданных постов
		post1, exists := postMap[fmt.Sprint(post1ID)]
		assert.True(t, exists)
		assert.Equal(t, "Post 1", post1.Title)

		post2, exists := postMap[fmt.Sprint(post2ID)]
		assert.True(t, exists)
		assert.Equal(t, "Post 2", post2.Title)

		post3, exists := postMap[fmt.Sprint(post3ID)]
		assert.True(t, exists)
		assert.Equal(t, "Post 3", post3.Title)
	})
}

func TestPostPostgresStorage_DisableComment(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Disable comments by author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		err := storage.DisableComment(ctx, fmt.Sprint(postID))
		assert.NoError(t, err)

		var post models.Post
		err = DB.First(&post, postID).Error
		assert.NoError(t, err)
		assert.True(t, post.CommentsDisabled)
	})

	t.Run("Disable comments by not author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		authorID := createTestUser(t)

		anotherUser := &models.User{
			Username: "anotheruser",
			Email:    "another@example.com",
			Password: "password123",
		}
		err := DB.Create(anotherUser).Error
		require.NoError(t, err)

		// Создаем тестовый пост от имени первого пользователя
		postID := createTestPost(t, authorID, "Test Post", "Test Content")

		// Создаем контекст с ID второго пользователя (не автора)
		ctx := createUserContext(anotherUser.ID)

		// Вызываем тестируемый метод
		err = storage.DisableComment(ctx, fmt.Sprint(postID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")

		// Проверяем, что комментарии не были отключены
		var post models.Post
		err = DB.First(&post, postID).Error
		assert.NoError(t, err)
		assert.False(t, post.CommentsDisabled)
	})

	t.Run("Disable comment for not exists post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		ctx := createUserContext(userID)

		err := storage.DisableComment(ctx, "999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post not found")
	})

	t.Run("Disable comment by unauthorized user", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := context.Background()

		err := storage.DisableComment(ctx, fmt.Sprint(postID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// Проверяем, что комментарии не были отключены
		var post models.Post
		err = DB.First(&post, postID).Error
		assert.NoError(t, err)
		assert.False(t, post.CommentsDisabled)
	})
}

func TestPostPostgresStorage_EnableComment(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Enable comment by author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		// Создаем тестовый пост с отключенными комментариями
		post := &models.Post{
			Title:            "Test Post",
			Content:          "Test Content",
			UserID:           userID,
			CommentsDisabled: true,
		}
		err := DB.Create(post).Error
		require.NoError(t, err)

		ctx := createUserContext(userID)

		err = storage.EnableComment(ctx, fmt.Sprint(post.ID))
		assert.NoError(t, err)

		// Проверяем, что комментарии действительно включены в БД
		var updatedPost models.Post
		err = DB.First(&updatedPost, post.ID).Error
		assert.NoError(t, err)
		assert.False(t, updatedPost.CommentsDisabled)
	})

	t.Run("Enable comment by not author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		authorID := createTestUser(t)

		// Создаем второго пользователя
		anotherUser := &models.User{
			Username: "anotheruser",
			Email:    "another@example.com",
			Password: "password123",
		}
		err := DB.Create(anotherUser).Error
		require.NoError(t, err)

		// Создаем тестовый пост с отключенными комментариями
		post := &models.Post{
			Title:            "Test Post",
			Content:          "Test Content",
			UserID:           authorID,
			CommentsDisabled: true,
		}
		err = DB.Create(post).Error
		require.NoError(t, err)

		// Создаем контекст с ID второго пользователя (не автора)
		ctx := createUserContext(anotherUser.ID)

		err = storage.EnableComment(ctx, fmt.Sprint(post.ID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")

		// Проверяем, что комментарии остались отключенными
		var updatedPost models.Post
		err = DB.First(&updatedPost, post.ID).Error
		assert.NoError(t, err)
		assert.True(t, updatedPost.CommentsDisabled)
	})

	t.Run("Enable comment for not exists post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		ctx := createUserContext(userID)

		err := storage.EnableComment(ctx, "999")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post not found")
	})

	t.Run("Enable comment by unauthorized user", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		// Создаем тестовый пост с отключенными комментариями
		post := &models.Post{
			Title:            "Test Post",
			Content:          "Test Content",
			UserID:           userID,
			CommentsDisabled: true,
		}
		err := DB.Create(post).Error
		require.NoError(t, err)

		ctx := context.Background()
		err = storage.EnableComment(ctx, fmt.Sprint(post.ID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// Проверяем, что комментарии остались отключенными
		var updatedPost models.Post
		err = DB.First(&updatedPost, post.ID).Error
		assert.NoError(t, err)
		assert.True(t, updatedPost.CommentsDisabled)
	})
}

func TestPostPostgresStorage_DeletePostById(t *testing.T) {
	storage := NewPostPostgresStorage()

	t.Run("Delete post by author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		err := storage.DeletePostById(ctx, fmt.Sprint(postID))
		assert.NoError(t, err)

		// Проверяем, что пост действительно удален из БД
		var post models.Post
		err = DB.First(&post, postID).Error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})

	t.Run("Delete not exist post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		// Создаем тестового пользователя
		userID := createTestUser(t)

		// Создаем контекст с ID пользователя
		ctx := createUserContext(userID)

		// Вызываем тестируемый метод с несуществующим ID
		err := storage.DeletePostById(ctx, "999")

		// Проверяем, что возникла ошибка "пост не найден"
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post not found")
	})

	t.Run("Delete post by not author", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		// Создаем двух тестовых пользователей
		authorID := createTestUser(t)

		// Создаем второго пользователя
		anotherUser := &models.User{
			Username: "anotheruser",
			Email:    "another@example.com",
			Password: "password123",
		}
		err := DB.Create(anotherUser).Error
		require.NoError(t, err)

		// Создаем тестовый пост от имени первого пользователя
		postID := createTestPost(t, authorID, "Test Post", "Test Content")

		// Создаем контекст с ID второго пользователя (не автора)
		ctx := createUserContext(anotherUser.ID)

		// Вызываем тестируемый метод
		err = storage.DeletePostById(ctx, fmt.Sprint(postID))

		// Проверяем, что возникла ошибка "не автор"
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden")

		// Проверяем, что пост не был удален
		var post models.Post
		err = DB.First(&post, postID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Post", post.Title)
	})

	t.Run("Delete by unauthorized user", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := context.Background()

		err := storage.DeletePostById(ctx, fmt.Sprint(postID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")

		// Проверяем, что пост не был удален
		var post models.Post
		err = DB.First(&post, postID).Error
		assert.NoError(t, err)
		assert.Equal(t, "Test Post", post.Title)
	})
}

// Тестирование многопоточности с использованием SQLite в режиме in-memory не имеет смысла
// SQLite не предназначен для интенсивного параллельного доступа, особенно в режиме in-memory
// Мой код в PostPostgresStorage делегирует всю работу с данными базе данных PostgreSQL, которая имеет встроенное управление параллельным доступом.
