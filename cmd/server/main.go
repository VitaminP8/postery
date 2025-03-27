package main

import (
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"

	"github.com/VitaminP8/postery/graph"
	"github.com/VitaminP8/postery/graph/generated"
)

func main() {
	// Инициализация резолверов
	resolver := &graph.Resolver{}

	// Создаем новый сервер GraphQL с резолверами
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Создаем HTTP маршруты
	http.Handle("/query", srv)

	// Страница с тестовым интерфейсом Playground
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	// Запуск сервера
	log.Println("Запуск сервера на http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
