# QMS Speedtest Exporter

Сервис на Go для запуска speedtest через внешний бинарник qms_lib и экспонирования результатов в формате Prometheus.

Для запуска тестов скорости используется cli от QMS (https://www.qms.ru/applications/linux).

- HTTP‑сервер на chi с middleware (recovery, requestID, realIP, логирование на slog) и Graceful Shutdown
- Эндпойнты: /health, /server_list, /speedtest
- Конфиг через переменные окружения и .env
- Готов к запуску в Docker/Compose (в т.ч. на ARM‑хостах с контейнером amd64)

## Требования для запуска бинарника qms_lib
- Требования
- Процессор x86_64
- Linux kernel >= 3.2
- 300Мб свободной оперативной памяти
- Доступ в интернет

## Описание
Экспортер вызывает внешний бинарник qms_lib:
- Получение списка серверов (/server_list).
- Запуск теста скорости и чтение результата из JSON-файла (/speedtest).
Результаты отдаются как текст в формате Prometheus exposition format (text/plain; version=0.0.4).

По умолчанию сервер слушает порт 8080: http://localhost:8080

## Конфигурация
Переменные окружения (файл .env поддерживается):
- HTTP_PORT — порт HTTP сервера (по умолчанию 8080)
- BINARY_PATH — путь к бинарнику qms_lib (по умолчанию bin/qms_lib)
- SERVER_DATA_PATH — путь к файлу с серверами (по умолчанию server_data) (не менять, всегда создается в каталоге запуска)
- TEST_RESULT_PATH — путь к файлу результата теста (по умолчанию data/test.json)
- EXEC_TIMEOUT_SEC — таймаут выполнения запроса/теста (по умолчанию 120s)
- SERVER_ID — опционально, фиксированный ID сервера для теста (по умолчанию 0 — не задан), список ID можно получить /server_list

Пример .env:
```
HTTP_PORT=8080
BINARY_PATH=bin/qms_lib
SERVER_DATA_PATH=server_data
TEST_RESULT_PATH=data/test.json
EXEC_TIMEOUT_SEC=120s
SERVER_ID=0
```

## Эндпойнты
- GET / — простая HTML‑страница со ссылками на хэндлеры.
- GET /health — проверка доступности сервиса. Возвращает "OK".
- GET /server_list — возвращает JSON со списком серверов.
- GET /speedtest — запускает тест скорости и возвращает метрики Prometheus.

Контент‑тайп для /speedtest: `text/plain; version=0.0.4; charset=utf-8`.


## Docker Compose
Пример compose.yml уже в репозитории. Запуск:
```
docker compose up -d --build
```
Полезные переменные окружения для Compose:
- DOCKER_REGISTRY, DOCKER_USERNAME, VERSION — для имени образа
- HTTP_PORT — проброс порта на хост

## Замечания и ограничения
- Время выполнения speedtest может занимать десятки секунд; на запрос /speedtest заложен таймаут EXEC_TIMEOUT_SEC.
- Бинарник qms_lib должен быть совместим с окружением, в котором он запускается. В Docker образе предполагается linux/amd64. На ARM‑хосте не работает.
- Файлы результатов/серверов должны быть доступны по путям в конфиге (смонтируйте volume на /app/data при запуске в Docker), результаты получения списка серверов всегда создаются в каталоге запуска приложения.
