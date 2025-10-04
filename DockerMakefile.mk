# Makefile для сборки и публикации Docker образа

# Переменные
DOCKER_REGISTRY ?= docker.io
DOCKER_USERNAME ?= langowen
IMAGE_NAME = qms-speedtest-exporter
VERSION ?= latest
FULL_IMAGE_NAME = $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(IMAGE_NAME):$(VERSION)
PLATFORM = linux/amd64

# Цвета для вывода
GREEN  := \033[0;32m
YELLOW := \033[0;33m
NC     := \033[0m # No Color

.PHONY: help build push login test clean run stop logs all

# Помощь
help:
	@echo "$(GREEN)Доступные команды:$(NC)"
	@echo "  $(YELLOW)make build$(NC)       - Собрать Docker образ"
	@echo "  $(YELLOW)make push$(NC)        - Опубликовать образ в registry"
	@echo "  $(YELLOW)make login$(NC)       - Войти в Docker registry"
	@echo "  $(YELLOW)make test$(NC)        - Протестировать образ локально"
	@echo "  $(YELLOW)make run$(NC)         - Запустить контейнер через compose"
	@echo "  $(YELLOW)make stop$(NC)        - Остановить контейнер"
	@echo "  $(YELLOW)make logs$(NC)        - Показать логи контейнера"
	@echo "  $(YELLOW)make clean$(NC)       - Удалить локальный образ"
	@echo "  $(YELLOW)make all$(NC)         - Собрать, протестировать и опубликовать"
	@echo ""
	@echo "$(GREEN)Переменные окружения:$(NC)"
	@echo "  DOCKER_REGISTRY  - Docker registry (по умолчанию: docker.io)"
	@echo "  DOCKER_USERNAME  - Имя пользователя в registry"
	@echo "  VERSION          - Версия образа (по умолчанию: latest)"
	@echo ""
	@echo "$(GREEN)Примеры:$(NC)"
	@echo "  make build VERSION=1.0.0"
	@echo "  make push DOCKER_USERNAME=myuser VERSION=1.0.0"

# Сборка образа
build:
	@echo "$(GREEN)Сборка Docker образа $(FULL_IMAGE_NAME)...$(NC)"
	docker buildx build \
		--platform $(PLATFORM) \
		--tag $(FULL_IMAGE_NAME) \
		--load \
		.
	@echo "$(GREEN)✓ Образ собран успешно!$(NC)"

# Вход в Docker registry
login:
	@echo "$(YELLOW)Вход в Docker registry $(DOCKER_REGISTRY)...$(NC)"
	@docker login $(DOCKER_REGISTRY)

# Публикация образа
push: build
	@echo "$(GREEN)Публикация образа $(FULL_IMAGE_NAME)...$(NC)"
	docker push $(FULL_IMAGE_NAME)
	@echo "$(GREEN)✓ Образ опубликован успешно!$(NC)"

# Тестирование образа локально
test: build
	@echo "$(GREEN)Запуск тестового контейнера...$(NC)"
	docker run --rm -d \
		--name qms-speedtest-exporter-test \
		--platform $(PLATFORM) \
		-p 8080:8080 \
		$(FULL_IMAGE_NAME)
	@echo "$(YELLOW)Ожидание запуска сервиса (10 секунд)...$(NC)"
	@sleep 10
	@echo "$(GREEN)Проверка health endpoint...$(NC)"
	@curl -f http://localhost:8080/health && echo "$(GREEN)✓ Health check OK$(NC)" || echo "$(YELLOW)⚠ Health check failed$(NC)"
	@echo "$(GREEN)Остановка тестового контейнера...$(NC)"
	@docker stop qms-speedtest-exporter-test
	@echo "$(GREEN)✓ Тест завершен$(NC)"

# Запуск через docker-compose
run:
	@echo "$(GREEN)Запуск контейнера через docker-compose...$(NC)"
	DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
	DOCKER_USERNAME=$(DOCKER_USERNAME) \
	VERSION=$(VERSION) \
	docker-compose up -d
	@echo "$(GREEN)✓ Контейнер запущен$(NC)"
	@echo "$(YELLOW)Используйте 'make logs' для просмотра логов$(NC)"

# Остановка контейнера
stop:
	@echo "$(GREEN)Остановка контейнера...$(NC)"
	docker-compose down
	@echo "$(GREEN)✓ Контейнер остановлен$(NC)"

# Просмотр логов
logs:
	docker-compose logs -f

# Удаление локального образа
clean:
	@echo "$(GREEN)Удаление локального образа...$(NC)"
	-docker rmi $(FULL_IMAGE_NAME)
	@echo "$(GREEN)✓ Образ удален$(NC)"

# Полный цикл: сборка, тест, публикация
all: build test push
	@echo "$(GREEN)✓ Полный цикл завершен успешно!$(NC)"

# Сборка с версией из git тега
build-release:
	@if [ -z "$$(git describe --tags --exact-match 2>/dev/null)" ]; then \
		echo "$(YELLOW)⚠ Нет git тега на текущем коммите$(NC)"; \
		exit 1; \
	fi
	@VERSION=$$(git describe --tags --exact-match) $(MAKE) build
	@echo "$(GREEN)✓ Релизный образ собран с версией $$(git describe --tags)$(NC)"

# Публикация релиза
release: build-release
	@VERSION=$$(git describe --tags --exact-match) $(MAKE) push
	@echo "$(GREEN)✓ Релиз $$(git describe --tags) опубликован$(NC)"
