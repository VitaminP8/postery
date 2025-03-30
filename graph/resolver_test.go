package graph

import (
	"context"
	"testing"
	"time"

	"github.com/VitaminP8/postery/graph/model"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createUserContext(userID uint) context.Context {
	ctx := context.Background()
	return auth.WithUserID(ctx, userID)
}

func TestMutationResolver_CreatePost(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()

	resolver := &Resolver{
		PostStore: mockPostStorage,
	}

	t.Run("Successful post creation", func(t *testing.T) {
		ctx := createUserContext(123)

		title := "Test Post"
		content := "Test Content"

		post, err := resolver.Mutation().CreatePost(ctx, title, content)
		require.NoError(t, err)
		assert.NotEmpty(t, post.ID)
		assert.Equal(t, title, post.Title)
		assert.Equal(t, content, post.Content)
		assert.Equal(t, "123", post.AuthorID)
		assert.False(t, post.CommentsDisabled)

		savedPost, err := mockPostStorage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.Equal(t, post, savedPost)
	})

	t.Run("Error when no authorization", func(t *testing.T) {
		ctx := context.Background()

		post, err := resolver.Mutation().CreatePost(ctx, "Title", "Content")
		assert.Error(t, err)
		assert.Nil(t, post)
	})
}

func TestMutationResolver_DeletePostById(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()

	resolver := &Resolver{
		PostStore: mockPostStorage,
	}

	ctx := createUserContext(123)

	post, err := mockPostStorage.CreatePost(ctx, "Test Post", "Content")
	require.NoError(t, err)

	t.Run("Successfully delete post", func(t *testing.T) {
		success, err := resolver.Mutation().DeletePostByID(ctx, post.ID)
		require.NoError(t, err)
		assert.True(t, success)

		_, err = mockPostStorage.GetPostById(post.ID)
		assert.Error(t, err)
	})

	t.Run("Error when deleting non-existent post", func(t *testing.T) {
		success, err := resolver.Mutation().DeletePostByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.False(t, success)
	})
}

func TestMutationResolver_DisableEnableComment(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()

	resolver := &Resolver{
		PostStore: mockPostStorage,
	}

	ctx := createUserContext(123)

	post, err := mockPostStorage.CreatePost(ctx, "Test Post", "Content")
	require.NoError(t, err)

	t.Run("Successfully disable comments", func(t *testing.T) {
		success, err := resolver.Mutation().DisableComment(ctx, post.ID)
		require.NoError(t, err)
		assert.True(t, success)

		updatedPost, err := mockPostStorage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.True(t, updatedPost.CommentsDisabled)
	})

	t.Run("Successfully enable comments", func(t *testing.T) {
		success, err := resolver.Mutation().EnableComment(ctx, post.ID)
		require.NoError(t, err)
		assert.True(t, success)

		updatedPost, err := mockPostStorage.GetPostById(post.ID)
		require.NoError(t, err)
		assert.False(t, updatedPost.CommentsDisabled)
	})

	t.Run("Error when post does not exist", func(t *testing.T) {
		success, err := resolver.Mutation().DisableComment(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.False(t, success)
	})
}

func TestQueryResolver_Posts(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()

	resolver := &Resolver{
		PostStore: mockPostStorage,
	}

	ctx := createUserContext(123)

	post1, err := mockPostStorage.CreatePost(ctx, "Post 1", "Content 1")
	require.NoError(t, err)

	post2, err := mockPostStorage.CreatePost(ctx, "Post 2", "Content 2")
	require.NoError(t, err)

	t.Run("Successfully get all posts", func(t *testing.T) {
		posts, err := resolver.Query().Posts(ctx)
		require.NoError(t, err)
		assert.Len(t, posts, 2)

		postMap := make(map[string]*model.Post)
		for _, p := range posts {
			postMap[p.ID] = p
		}
		assert.Contains(t, postMap, post1.ID)
		assert.Contains(t, postMap, post2.ID)
		assert.Equal(t, "Post 1", postMap[post1.ID].Title)
		assert.Equal(t, "Post 2", postMap[post2.ID].Title)
	})
}

func TestQueryResolver_Post(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()

	resolver := &Resolver{
		PostStore: mockPostStorage,
	}

	ctx := createUserContext(123)

	post, err := mockPostStorage.CreatePost(ctx, "Test Post", "Content")
	require.NoError(t, err)

	t.Run("Successfully get post by ID", func(t *testing.T) {
		retrievedPost, err := resolver.Query().Post(ctx, post.ID)
		require.NoError(t, err)
		assert.Equal(t, post.ID, retrievedPost.ID)
		assert.Equal(t, post.Title, retrievedPost.Title)
		assert.Equal(t, post.Content, retrievedPost.Content)
		assert.Equal(t, post.AuthorID, retrievedPost.AuthorID)
	})

	t.Run("Error when post not found", func(t *testing.T) {
		retrievedPost, err := resolver.Query().Post(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, retrievedPost)
	})
}

func TestMutationResolver_RegisterAndLoginUser(t *testing.T) {
	mockUserStorage := mocks.NewMockUserStorage()

	resolver := &Resolver{
		UserStore: mockUserStorage,
	}

	ctx := context.Background()

	t.Run("Successfully register user", func(t *testing.T) {
		username := "testuser"
		email := "test@example.com"
		password := "password123"

		user, err := resolver.Mutation().RegisterUser(ctx, username, email, password)

		require.NoError(t, err)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, email, user.Email)
	})

	t.Run("Successfully login user", func(t *testing.T) {
		username := "testuser"
		password := "password123"

		tokenPtr, err := resolver.Mutation().LoginUser(ctx, username, password)
		require.NoError(t, err)
		require.NotNil(t, tokenPtr)
		assert.Contains(t, *tokenPtr, "jwt-token-for-user-")
	})

	t.Run("Error when registering existing user", func(t *testing.T) {
		username := "testuser"
		email := "another@example.com"
		password := "password456"

		user, err := resolver.Mutation().RegisterUser(ctx, username, email, password)
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("Error when logging in with wrong password", func(t *testing.T) {
		username := "testuser"
		password := "wrongpassword"

		tokenPtr, err := resolver.Mutation().LoginUser(ctx, username, password)
		assert.Error(t, err)
		assert.Nil(t, tokenPtr)
	})
}

func TestMutationResolver_CreateComment(t *testing.T) {
	mockPostStorage := mocks.NewMockPostStorage()
	subscriptionManager := mocks.NewMockSubscriptionManager()
	mockCommentStorage := mocks.NewMockCommentStorage(subscriptionManager)

	resolver := &Resolver{
		PostStore:           mockPostStorage,
		CommentStore:        mockCommentStorage,
		SubscriptionManager: subscriptionManager,
	}

	ctx := createUserContext(123)

	post, err := mockPostStorage.CreatePost(ctx, "Test Post", "Content")
	require.NoError(t, err)

	t.Run("Successfully create root comment", func(t *testing.T) {
		content := "Root Comment"
		var parentIDPtr *string = nil

		comment, err := resolver.Mutation().CreateComment(ctx, post.ID, parentIDPtr, content)
		require.NoError(t, err)
		assert.NotEmpty(t, comment.ID)
		assert.Equal(t, post.ID, comment.PostID)
		assert.Equal(t, content, comment.Content)
		assert.Equal(t, "123", comment.AuthorID)
		assert.Nil(t, comment.ParentID)
	})

	t.Run("Successfully create nested comment", func(t *testing.T) {
		parentContent := "Parent Comment"
		var parentIDPtr *string

		parentComment, err := resolver.Mutation().CreateComment(ctx, post.ID, parentIDPtr, parentContent)
		require.NoError(t, err)

		childContent := "Child Comment"
		childParentID := parentComment.ID

		childComment, err := resolver.Mutation().CreateComment(ctx, post.ID, &childParentID, childContent)
		require.NoError(t, err)
		assert.NotEmpty(t, childComment.ID)
		assert.Equal(t, post.ID, childComment.PostID)
		assert.Equal(t, childContent, childComment.Content)
		assert.Equal(t, "123", childComment.AuthorID)
		assert.NotNil(t, childComment.ParentID)
		assert.Equal(t, parentComment.ID, *childComment.ParentID)
	})

	t.Run("Error when no authorization", func(t *testing.T) {
		ctx := context.Background()

		var parentIDPtr *string
		comment, err := resolver.Mutation().CreateComment(ctx, post.ID, parentIDPtr, "Comment")
		assert.Error(t, err)
		assert.Nil(t, comment)
	})
}

func TestSubscriptionResolver_CommentAdded(t *testing.T) {
	subscriptionManager := mocks.NewMockSubscriptionManager()

	resolver := &Resolver{
		SubscriptionManager: subscriptionManager,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	postID := "test-post-1"

	t.Run("Successfully subscribe to comments", func(t *testing.T) {
		commentChan, err := resolver.Subscription().CommentAdded(ctx, postID)
		require.NoError(t, err)
		assert.NotNil(t, commentChan)

		comment := &model.Comment{
			ID:        "test-comment-1",
			PostID:    postID,
			Content:   "Test Comment",
			AuthorID:  "123",
			CreatedAt: time.Now().Format(time.RFC3339),
		}

		subscriptionManager.Publish(postID, comment)

		select {
		case receivedComment := <-commentChan:
			assert.Equal(t, comment.ID, receivedComment.ID)
			assert.Equal(t, comment.Content, receivedComment.Content)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for comment")
		}

		notifications := subscriptionManager.GetNotificationsForPost(postID)
		assert.Len(t, notifications, 1)
		assert.Equal(t, comment.ID, notifications[0].ID)
	})
}
