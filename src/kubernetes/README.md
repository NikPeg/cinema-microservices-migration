# Развертывание CinemaAbyss в Kubernetes

## Предварительные требования

- Установленный kubectl
- Доступ к Kubernetes кластеру
- Docker образы в GitHub Container Registry

## Порядок развертывания

### 1. Создание namespace и секретов

```bash
# Создание namespace
kubectl apply -f namespace.yaml

# Создание секрета для доступа к GitHub Container Registry
kubectl apply -f dockerconfigsecret.yaml

# Создание секрета с паролями
kubectl apply -f secret.yaml
```

### 2. Развертывание базы данных

```bash
# ConfigMap с init скриптом для PostgreSQL
kubectl apply -f postgres-init-configmap.yaml

# Развертывание PostgreSQL
kubectl apply -f postgres.yaml
```

### 3. Развертывание Kafka

```bash
kubectl apply -f kafka/kafka.yaml
```

### 4. Развертывание конфигураций

```bash
# ConfigMap с настройками приложения
kubectl apply -f configmap.yaml
```

### 5. Развертывание сервисов

```bash
# Монолит
kubectl apply -f monolith.yaml

# Микросервис movies
kubectl apply -f movies-service.yaml

# Proxy-service (API Gateway)
kubectl apply -f proxy-service.yaml

# Events-service
kubectl apply -f events-service.yaml
```

### 6. Настройка Ingress

```bash
kubectl apply -f ingress.yaml
```

## Проверка развертывания

### Проверка статуса подов

```bash
kubectl get pods -n cinemaabyss
```

### Проверка сервисов

```bash
kubectl get services -n cinemaabyss
```

### Проверка логов

```bash
# Логи proxy-service
kubectl logs -n cinemaabyss -l app=proxy-service

# Логи events-service
kubectl logs -n cinemaabyss -l app=events-service

# Логи монолита
kubectl logs -n cinemaabyss -l app=monolith

# Логи movies-service
kubectl logs -n cinemaabyss -l app=movies-service
```

## Тестирование

### Локальное тестирование через port-forward

```bash
# Проброс порта для proxy-service
kubectl port-forward -n cinemaabyss service/proxy-service 8080:8080

# В другом терминале - тест API
curl http://localhost:8080/api/movies
```

### Тестирование через Ingress

Добавьте в /etc/hosts:
```
127.0.0.1 cinemaabyss.example.com
```

Затем откройте в браузере:
- http://cinemaabyss.example.com/api/movies
- http://cinemaabyss.example.com/api/events/health

### Запуск Postman тестов

```bash
cd tests/postman
npm test -- --env-var "BASE_URL=http://cinemaabyss.example.com"
```

## Управление миграцией

Процент трафика, направляемого на микросервис movies, управляется через ConfigMap:

```bash
# Изменение процента миграции на 50%
kubectl patch configmap cinemaabyss-config -n cinemaabyss \
  --type merge -p '{"data":{"MOVIES_MIGRATION_PERCENT":"50"}}'

# Перезапуск proxy-service для применения изменений
kubectl rollout restart deployment/proxy-service -n cinemaabyss
```

### Сценарии миграции

1. **0% миграции** - весь трафик идет на монолит
   ```bash
   MOVIES_MIGRATION_PERCENT: "0"
   ```

2. **50% миграции** - трафик распределяется поровну
   ```bash
   MOVIES_MIGRATION_PERCENT: "50"
   ```

3. **100% миграции** - весь трафик идет на микросервис
   ```bash
   MOVIES_MIGRATION_PERCENT: "100"
   ```

## Мониторинг событий

События автоматически публикуются в Kafka при операциях с фильмами:

```bash
# Просмотр логов events-service для отслеживания обработки событий
kubectl logs -n cinemaabyss -l app=events-service -f
```

## Удаление развертывания

```bash
# Удаление всех ресурсов в namespace
kubectl delete namespace cinemaabyss
```

## Использование Helm

Для упрощения развертывания можно использовать Helm charts:

```bash
cd helm
helm install cinemaabyss . -n cinemaabyss --create-namespace
```

## Troubleshooting

### Проблемы с pull образов

Проверьте секрет для Docker Registry:
```bash
kubectl get secret dockerconfigjson-github-com -n cinemaabyss -o yaml
```

### Проблемы с подключением к БД

Проверьте, что PostgreSQL запущен:
```bash
kubectl get pod -n cinemaabyss -l app=postgres
kubectl logs -n cinemaabyss -l app=postgres
```

### Проблемы с Kafka

Проверьте статус Kafka:
```bash
kubectl get pod -n cinemaabyss -l app=kafka
kubectl logs -n cinemaabyss -l app=kafka
