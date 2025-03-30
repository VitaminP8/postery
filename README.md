# 📰 Postery


## Возможности

- Посты и комментарии (поддерживает вложенность)
- Регистрация и авторизация (JWT)
- GraphQL Subscriptions (realtime комментарии)
- Поддержка PostgreSQL и in-memory хранилищ
- Docker + Makefile для удобного запуска


---

## Инструкция по сборке

### Создай `.env` файл в корне проекта:

Пример .env файла:

```env
DB_HOST=localhost
DB_USER=user
DB_PASSWORD=password
DB_NAME=postery
DB_PORT=5432
DB_SSLMODE=disable

JWT_SECRET=very-secret-key

APP_PORT= (оставьте пустым)
```


### Быстрый запуск в Docker

Собрать образы:

```bash
make docker-build
```

#### Запуск PostgreSQL-версии:

```bash
make docker-postgres
# или с логами:
make docker-postgres-logs
```

#### Запуск in-memory версии:

```bash
make docker-memory
# или с логами:
make docker-memory-logs
```

### Makefile: команды

Для просмотра полного списка доступных make команд:

```bash
make help
```

**Основные команды:**

| Команда                        | Описание                                           |
|-------------------------------|----------------------------------------------------|
| `make docker-build`           | Собрать Docker образы                              |
| `make docker-postgres`        | Запустить PostgreSQL-версию в Docker               |
| `make docker-postgres-logs`   | Запустить PostgreSQL-версию в Docker с логами      |
| `make docker-memory`          | Запустить in-memory версию в Docker                |
| `make docker-memory-logs`     | Запустить in-memory версию в Docker с логами       |
| `make docker-stop`            | Остановить все Docker-контейнеры                   |
| `make run-postgres`           | Локальный запуск с PostgreSQL                      |
| `make run-memory`             | Локальный запуск с in-memory хранилищем            |
| `make test-clear`             | Запустить тесты без флагов                         |
| `make test`                   | Запустить тесты с флагом `-v`                      |
| `make test-race`              | Запустить тесты с флагами `-v -race`               |

---

## Тестирование в GraphQL Playground

Для быстрого тестирования API в `GraphQL Playground` используйте файл `test_commands` — он содержит минимальный набор запросов, охватывающий весь основной функционал.

>  *Файл `test_commands` находится в корне проекта.*

Запросы в файле покрывают **базовые кейсы**, но **не содержат проверок на все ошибки**. Подробные проверки граничных и ошибочных ситуаций реализованы в юнит-тестах.

### Аутентификация:

- Без регистрации (`loginUser`) доступны только **запросы на чтение**.
- После регистрации (`loginUser`) вы получите **JWT-токен**, который необходимо передавать в заголовке авторизации (`Headers`) для выполнения **записей (mutation)**:

```json
{
  "Authorization": "Bearer <ваш JWT токен>"
}
```

###  Тестирование подписок (`subscription`)

1. Выполните подписку на новые комментарии к посту (команда указана в `test_commands`).
2. Откройте новую вкладку в GraphQL Playground.
3. Создайте новый комментарий к тому же посту — результат моментально появится в первой вкладке.

---

## Система пагинации

### Корневые комментарии

При запросе поста с комментариями или отдельных комментариев к посту:

- Возвращаются **только первые _n_ корневых комментариев**
- Если комментарии **еще остались**, то:
    - флаг `hasMore: true` указывает, что есть следующая страница
    - поле `nextOffset` содержит значение, с которого нужно продолжить загрузку


### Вложенные комментарии (ответы)

У каждого комментария:

- Есть флаг `hasReplies`, указывающий, есть ли у него вложенные ответы
- При вызове поля `replies` возвращается **первые _n_ вложенных комментариев** (логика как и с корневыми комментариями есть флаг `hasMore` и поле `nextOffset`)


```text
Post
├─ Comment #1  (hasReplies: true)
│   ├─ Reply #1.1 (hasReplies: true)
│   └─ Reply #1.2 (hasReplies: false)
├─ Comment #2  (hasReplies: true)
├─ Comment #3
└─ ... (hasMore: true, nextOffset: 3)
```
