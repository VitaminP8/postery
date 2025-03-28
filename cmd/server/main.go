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

	switch *storageType {
	case "postgres":
		postgres.InitDB()
		err := postgres.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}).Error
		if err != nil {
			log.Fatalf("failed to migrate database: %v", err)
		}

		log.Println("Используется PostgreSQL хранилище")
		postStore = postgres.NewPostPostgresStorage()
		commentStore = postgres.NewCommentPostgresStorage()
		userStore = postgres.NewUserPostgresStorage()

	case "memory":
		log.Println("Используется in-memory хранилище")
		// TODO: обновить реализацию для in-memory
		postStore = memory.NewPostMemoryStorage()
		commentStore = memory.NewCommentMemoryStorage(postStore)
		userStore = memory.NewUserMemoryStorage()

	default:
		log.Fatalf("неизвестный тип хранилища: %s", *storageType)
	}

	// Инициализация резолвера
	resolver := &graph.Resolver{
		PostStore:    postStore,
		CommentStore: commentStore,
		UserStore:    userStore,
	}

	// Создаем новый сервер GraphQL с резолверами
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// AuthMiddleware - http.Handler, который получает запрос, вытаскивает JWT токен из заголовка, проверяет и валидирует его, сохраняет userID в context,
	http.Handle("/query", auth.AuthMiddleware(srv))
	// Страница с тестовым интерфейсом Playground
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	//// Запуск сервера
	//log.Println("Запуск сервера на http://localhost:8080/")
	//log.Fatal(http.ListenAndServe(":8080", nil))

	// HTTP сервер
	server := &http.Server{
		Addr: ":8080",
	}

	// запуск HTTP сервер
	go func() {
		log.Println("Сервер запущен на http://localhost:8080/")
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
		postgres.CloseDB()
	}

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Ошибка при завершении сервера: %v", err)
	}

	log.Println("Сервер остановлен корректно")
}
