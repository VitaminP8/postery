package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/VitaminP8/postery/internal/auth"
	"github.com/VitaminP8/postery/internal/comment"
	"github.com/VitaminP8/postery/internal/config"
	"github.com/VitaminP8/postery/internal/post"
	"github.com/VitaminP8/postery/internal/subscription"
	"github.com/VitaminP8/postery/internal/user"

	"github.com/VitaminP8/postery/graph"
	"github.com/VitaminP8/postery/graph/generated"
	"github.com/VitaminP8/postery/internal/storage/memory"
	"github.com/VitaminP8/postery/internal/storage/postgres"
	"github.com/VitaminP8/postery/models"
)

func main() {
	storageType := flag.String("storage", "memory", "Тип хранилища: storage или postgres")
	flag.Parse()

	// загружаем .env из нашего config.go
	config.LoadEnv()

	var postStore post.PostStorage
	var commentStore comment.CommentStorage
	var userStore user.UserStorage
	var subMngr subscription.Manager

	switch *storageType {
	case "postgres":
		err := postgres.InitDB()
		if err != nil {
			log.Fatalf("failed to connect to the database: %v", err)
		}

		err = postgres.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}).Error
		if err != nil {
			log.Fatalf("failed to migrate database: %v", err)
		}

		log.Println("Используется PostgreSQL хранилище")
		subMngr = subscription.NewSubscriptionManager()
		postStore = postgres.NewPostPostgresStorage()
		commentStore = postgres.NewCommentPostgresStorage(subMngr)
		userStore = postgres.NewUserPostgresStorage()

	case "memory":
		log.Println("Используется in-memory хранилище")
		subMngr = subscription.NewSubscriptionManager()
		postStore = memory.NewPostMemoryStorage()
		commentStore = memory.NewCommentMemoryStorage(postStore, subMngr)
		userStore = memory.NewUserMemoryStorage()

	default:
		log.Fatalf("неизвестный тип хранилища: %s", *storageType)
	}

	// Инициализация резолвера
	resolver := &graph.Resolver{
		PostStore:           postStore,
		CommentStore:        commentStore,
		UserStore:           userStore,
		SubscriptionManager: subMngr,
	}

	// Создаем новый сервер GraphQL с резолверами
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// AuthMiddleware - http.Handler, который получает запрос, вытаскивает JWT токен из заголовка, проверяет и валидирует его, сохраняет userID в context,
	http.Handle("/query", auth.AuthMiddleware(srv))
	// Страница с тестовым интерфейсом Playground
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	// HTTP сервер
	server := &http.Server{
		Addr: ":8080",
	}

	// для запуска через докер
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080" // Значение по умолчанию
	}

	// запуск HTTP сервер
	go func() {
		log.Printf("Сервер запущен на http://localhost:%s/", port)
		// строка не возвращается (блокирует поток) пока не выполнится server.Shutdown() или не произойдет фатальная ошибка
		// Поэтому запускаем goroutine
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	// Ожидание SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // ждет сигнал

	log.Println("Завершение...")

	if *storageType == "postgres" {
		err := postgres.CloseDB()
		if err != nil {
			log.Printf("Ошибка при закрытии соединения с БД: %v", err)
		}
	}

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Ошибка при завершении сервера: %v", err)
	}

	log.Println("Сервер остановлен корректно")
}
