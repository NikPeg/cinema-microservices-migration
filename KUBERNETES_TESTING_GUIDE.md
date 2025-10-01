# Руководство по тестированию CinemaAbyss в Kubernetes

## Получение скриншотов для Project_template.md

### 1. Просмотр логов events-service

После развертывания в Kubernetes и запуска тестов, выполните следующие команды для просмотра логов:

```bash
# Просмотр логов events-service в реальном времени
kubectl logs -n cinemaabyss -l app=events-service -f

# Или просмотр логов конкретного пода
kubectl get pods -n cinemaabyss | grep events-service
kubectl logs -n cinemaabyss events-service-<pod-id>

# Для просмотра последних 100 строк логов
kubectl logs -n cinemaabyss -l app=events-service --tail=100
```

**Что должно быть в логах после запуска тестов:**
- `Processing movie event: {...}` - обработка событий о фильмах
- `Processing user event: {...}` - обработка событий пользователей
- `Processing payment event: {...}` - обработка платежных событий
- `Event published to topic: movie-events` - публикация событий

### 2. Проверка работы API через браузер

**Шаг 1:** Убедитесь, что minikube tunnel запущен:
```bash
minikube tunnel
```

**Шаг 2:** Проверьте /etc/hosts:
```bash
cat /etc/hosts | grep cinemaabyss
# Должна быть строка: 127.0.0.1 cinemaabyss.example.com
```

**Шаг 3:** Откройте в браузере:
```
http://cinemaabyss.example.com/api/movies
```

**Ожидаемый результат:** JSON массив с фильмами:
```json
[
  {
    "id": 1,
    "title": "The Shawshank Redemption",
    "year": 1994,
    "genre": "Drama"
  },
  ...
]
```

### 3. Запуск тестов для генерации событий

```bash
cd tests/postman

# Установка зависимостей (если еще не установлены)
npm install

# Для Kubernetes
npm run test:kubernetes

# Или напрямую с указанием URL
npm test -- --env-var "BASE_URL=http://cinemaabyss.example.com"
```

### 4. Команды для диагностики

**Проверка статуса подов:**
```bash
kubectl get pods -n cinemaabyss
```

Все поды должны быть в статусе `Running`:
- events-service-xxxxx
- proxy-service-xxxxx
- monolith-xxxxx
- movies-service-xxxxx
- postgres-0
- kafka-0
- zookeeper-0

**Проверка сервисов:**
```bash
kubectl get services -n cinemaabyss
```

**Проверка ingress:**
```bash
kubectl get ingress -n cinemaabyss
kubectl describe ingress cinemaabyss-ingress -n cinemaabyss
```

**Если есть проблемы с events-service:**
```bash
# Детальная информация о поде
kubectl describe pod -n cinemaabyss -l app=events-service

# События в namespace
kubectl get events -n cinemaabyss --sort-by='.lastTimestamp'
```

### 5. Проверка Kafka (опционально)

```bash
# Подключение к поду Kafka
kubectl exec -it kafka-0 -n cinemaabyss -- bash

# Внутри пода - список топиков
kafka-topics.sh --list --zookeeper zookeeper:2181

# Выход из пода
exit
```

## Где разместить скриншоты

В файле `Project_template.md` в разделе **"Задание 3 → Шаг 3"** нужно добавить:

1. **Скриншот API response** - результат вызова `http://cinemaabyss.example.com/api/movies`
2. **Скриншот логов events-service** - показывающий обработку событий после запуска тестов

## Проверочный чек-лист

- [ ] Minikube запущен (`minikube status`)
- [ ] Tunnel активен (`minikube tunnel`)
- [ ] Все поды в статусе Running
- [ ] Ingress настроен и доступен
- [ ] /etc/hosts содержит запись для cinemaabyss.example.com
- [ ] API отвечает на запросы
- [ ] Events-service обрабатывает события
- [ ] Тесты Postman проходят успешно

## Устранение типичных проблем

### Проблема: "Connection refused" при обращении к API

**Решение:**
```bash
# Проверить tunnel
minikube tunnel

# Проверить ingress controller
kubectl get pods -n ingress-nginx
```

### Проблема: События не обрабатываются

**Решение:**
```bash
# Проверить Kafka
kubectl logs kafka-0 -n cinemaabyss --tail=50

# Проверить events-service
kubectl logs -n cinemaabyss -l app=events-service --tail=50
```

### Проблема: Поды не запускаются

**Решение:**
```bash
# Проверить секреты
kubectl get secrets -n cinemaabyss

# Проверить образы
kubectl describe pod <pod-name> -n cinemaabyss
