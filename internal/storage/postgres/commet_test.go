package postgres

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/VitaminP8/postery/internal/mocks"
	"github.com/VitaminP8/postery/models"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentPostgresStorage_CreateComment(t *testing.T) {
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentPostgresStorage(subscriptionManager)

	t.Run("Successful comment creation", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		content := "Test Comment"
		comment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", content)
		require.NoError(t, err)
		assert.NotEmpty(t, comment.ID)
		assert.Equal(t, fmt.Sprint(postID), comment.PostID)
		assert.Equal(t, content, comment.Content)
		assert.Equal(t, fmt.Sprint(userID), comment.AuthorID)
		assert.Nil(t, comment.ParentID)
		assert.False(t, comment.HasReplies)
		assert.Empty(t, comment.Children)

		// Проверяем, что комментарий действительно создан в БД
		var dbComment models.Comment
		err = DB.First(&dbComment, comment.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, content, dbComment.Content)
		assert.Equal(t, userID, dbComment.UserID)
		assert.Equal(t, postID, dbComment.PostID)
	})

	t.Run("Creating nested comment", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		parentContent := "Parent Comment"
		parentComment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", parentContent)
		require.NoError(t, err)

		childContent := "Child Comment"
		childComment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), parentComment.ID, childContent)

		require.NoError(t, err)
		assert.NotEmpty(t, childComment.ID)
		assert.Equal(t, fmt.Sprint(postID), childComment.PostID)
		assert.Equal(t, childContent, childComment.Content)
		assert.Equal(t, fmt.Sprint(userID), childComment.AuthorID)
		assert.NotNil(t, childComment.ParentID)
		assert.Equal(t, parentComment.ID, *childComment.ParentID)

		// Проверяем, что родительский комментарий помечен как имеющий ответы
		var updatedParentComment models.Comment
		err = DB.First(&updatedParentComment, parentComment.ID).Error
		assert.NoError(t, err)
		assert.True(t, updatedParentComment.HasReplies)

		// Проверяем, что дочерний комментарий правильно связан с родительским
		parentIDUint, err := strconv.Atoi(*childComment.ParentID)
		require.NoError(t, err)

		var dbChildComment models.Comment
		err = DB.First(&dbChildComment, childComment.ID).Error
		assert.NoError(t, err)
		assert.NotNil(t, dbChildComment.ParentID)
		assert.Equal(t, uint(parentIDUint), *dbChildComment.ParentID)
	})

	t.Run("Error when creating comment for disabled comments", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		// Создаем тестовый пост с отключенными комментариями
		post := &models.Post{
			Title:            "Post with Disabled Comments",
			Content:          "Content",
			UserID:           userID,
			CommentsDisabled: true,
		}
		err := DB.Create(post).Error
		require.NoError(t, err)

		ctx := createUserContext(userID)

		_, err = commentStorage.CreateComment(ctx, fmt.Sprint(post.ID), "", "This should fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "comments are disabled")
	})

	t.Run("Error when creating comment with empty content", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is too long or empty")
	})

	t.Run("Error when creating comment with too long content", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		longContent := ""
		for i := 0; i < 2001; i++ {
			longContent += "a"
		}

		_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", longContent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content is too long or empty")
	})

	t.Run("Error when creating comment for non-existent post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		ctx := createUserContext(userID)

		_, err := commentStorage.CreateComment(ctx, "999", "", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "post not found")
	})

	t.Run("Error when creating comment with invalid parent ID", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "24235253", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parent ID")
	})

	t.Run("Error on unauthorized request", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		unauthorizedCtx := context.Background()

		_, err := commentStorage.CreateComment(unauthorizedCtx, fmt.Sprint(postID), "", "Test Comment")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unautorized")
	})
}

func TestCommentPostgresStorage_GetComments(t *testing.T) {
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentPostgresStorage(subscriptionManager)

	t.Run("Getting all comments without pagination", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		numRootComments := 5
		for i := 0; i < numRootComments; i++ {
			content := "Root Comment " + strconv.Itoa(i)
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", content)
			require.NoError(t, err)

			// Небольшая задержка для гарантии разного времени создания
			time.Sleep(10 * time.Millisecond)
		}

		comments, err := commentStorage.GetComments(fmt.Sprint(postID), 10, 0)
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
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		numRootComments := 5
		for i := 0; i < numRootComments; i++ {
			content := "Root Comment " + strconv.Itoa(i)
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", content)
			require.NoError(t, err)

			// Небольшая задержка для гарантии разного времени создания
			time.Sleep(10 * time.Millisecond)
		}

		// Получаем первую страницу (2 комментария)
		page1, err := commentStorage.GetComments(fmt.Sprint(postID), 2, 0)
		require.NoError(t, err)
		assert.Len(t, page1.Items, 2)
		assert.True(t, page1.HasMore)
		assert.Equal(t, 2, page1.NextOffset)

		// Получаем вторую страницу (2 комментария)
		page2, err := commentStorage.GetComments(fmt.Sprint(postID), 2, 2)
		require.NoError(t, err)
		assert.Len(t, page2.Items, 2)
		assert.True(t, page2.HasMore)
		assert.Equal(t, 4, page2.NextOffset)

		// Получаем третью страницу (1 комментарий)
		page3, err := commentStorage.GetComments(fmt.Sprint(postID), 2, 4)
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
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)

		// Создаем тестовый пост с отключенными комментариями
		post := &models.Post{
			Title:            "Post with Disabled Comments",
			Content:          "Content",
			UserID:           userID,
			CommentsDisabled: true,
		}
		err := DB.Create(post).Error
		require.NoError(t, err)

		// Получаем комментарии для поста с отключенными комментариями
		comments, err := commentStorage.GetComments(fmt.Sprint(post.ID), 10, 0)
		require.NoError(t, err)
		assert.Len(t, comments.Items, 0)
		assert.False(t, comments.HasMore)
	})

	t.Run("Getting comments for non-existent post", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		_, err := commentStorage.GetComments("999", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not get post")
	})

	t.Run("Getting comments with large offset", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		for i := 0; i < 3; i++ {
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", "Comment "+strconv.Itoa(i))
			require.NoError(t, err)
		}

		// Получаем комментарии с большим смещением
		comments, err := commentStorage.GetComments(fmt.Sprint(postID), 10, 100)
		require.NoError(t, err)
		assert.Len(t, comments.Items, 0)
		assert.False(t, comments.HasMore)
		assert.Equal(t, 110, comments.NextOffset)
	})
}

func TestCommentPostgresStorage_GetReplies(t *testing.T) {
	subscriptionManager := mocks.NewMockSubscriptionManager()
	commentStorage := NewCommentPostgresStorage(subscriptionManager)

	t.Run("Getting all child comments", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		parentComment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", "Parent Comment")
		require.NoError(t, err)

		numChildComments := 5
		for i := 0; i < numChildComments; i++ {
			content := "Child Comment " + strconv.Itoa(i)
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), parentComment.ID, content)
			require.NoError(t, err)

			// Небольшая задержка для гарантии разного времени создания
			time.Sleep(10 * time.Millisecond)
		}

		// Получаем все дочерние комментарии
		replies, err := commentStorage.GetReplies(parentComment.ID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, replies.Items, numChildComments)
		assert.False(t, replies.HasMore)

		// Проверяем, что комментарии отсортированы по времени создания
		for i := 1; i < len(replies.Items); i++ {
			assert.True(t, replies.Items[i-1].CreatedAt <= replies.Items[i].CreatedAt,
				"Комментарии должны быть отсортированы по возрастанию CreatedAt")
		}
	})

	t.Run("Getting child comments with pagination", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		parentComment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", "Parent Comment")
		require.NoError(t, err)

		numChildComments := 5
		for i := 0; i < numChildComments; i++ {
			content := "Child Comment " + strconv.Itoa(i)
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), parentComment.ID, content)
			require.NoError(t, err)

			// Небольшая задержка для гарантии разного времени создания
			time.Sleep(10 * time.Millisecond)
		}

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

		// Проверяем, что все комментарии разные
		allCommentIDs := make(map[string]bool)
		for _, comment := range append(append(page1.Items, page2.Items...), page3.Items...) {
			allCommentIDs[comment.ID] = true
		}
		assert.Len(t, allCommentIDs, numChildComments)
	})

	t.Run("Getting child comments for non-existent parent", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		_, err := commentStorage.GetReplies("999", 10, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid parent ID")
	})

	t.Run("Getting child comments with large offset", func(t *testing.T) {
		// Настраиваем тестовую БД
		oldDB := setupTestDB(t)
		defer teardownTestDB(oldDB)

		userID := createTestUser(t)
		postID := createTestPost(t, userID, "Test Post", "Test Content")

		ctx := createUserContext(userID)

		parentComment, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), "", "Parent Comment")
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			_, err := commentStorage.CreateComment(ctx, fmt.Sprint(postID), parentComment.ID, "Child Comment "+strconv.Itoa(i))
			require.NoError(t, err)
		}

		// Получаем дочерние комментарии с большим смещением
		replies, err := commentStorage.GetReplies(parentComment.ID, 10, 100)
		require.NoError(t, err)
		assert.Len(t, replies.Items, 0)
		assert.False(t, replies.HasMore)
		assert.Equal(t, 110, replies.NextOffset)
	})
}

// Тестирование многопоточности с использованием SQLite в режиме in-memory не имеет смысла
// SQLite не предназначен для интенсивного параллельного доступа, особенно в режиме in-memory
// Код в CommentPostgresStorage делегирует всю работу с данными базе данных PostgreSQL, которая имеет встроенное управление параллельным доступом.
