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

	"github.com/VitaminP8/postery/graph"
	"github.com/VitaminP8/postery/graph/generated"
	"github.com/VitaminP8/postery/internal/storage/memory"
	database "github.com/VitaminP8/postery/internal/storage/postgres"
	"github.com/VitaminP8/postery/models"
)

func main() {
	storageType := flag.String("storage", "memory", "Тип хранилища: storage или postgres")
	flag.Parse()

	postStore := memory.NewPostMemoryStorage()
	commentStore := memory.NewCommentMemoryStorage(postStore)

	switch *storageType {
	case "postgres":
		database.InitDB()
		// миграция таблиц
		err := database.DB.AutoMigrate(&models.User{}, &models.Post{}, &models.Comment{}).Error
		if err != nil {
			log.Fatalf("failed to migrate database: %v", err)
		}

		log.Println("Используется PostgreSQL хранилище")
		// TODO: заменить на реализацию PostgreSQL
		postStore = memory.NewPostMemoryStorage()
		commentStore = memory.NewCommentMemoryStorage(postStore)

	case "memory":
		log.Println("Используется in-memory хранилище")
		postStore = memory.NewPostMemoryStorage()
		commentStore = memory.NewCommentMemoryStorage(postStore)

	default:
		log.Fatalf("неизвестный тип хранилища: %s", *storageType)
	}

	// Инициализация резолвера
	resolver := &graph.Resolver{
		PostStore:    postStore,
		CommentStore: commentStore,
	}

	// Создаем новый сервер GraphQL с резолверами
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Создаем HTTP маршруты
	http.Handle("/query", srv)

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
		database.CloseDB()
	}

	if err := server.Shutdown(context.Background()); err != nil {
		log.Fatalf("Ошибка при завершении сервера: %v", err)
	}

	log.Println("Сервер остановлен корректно")
}
