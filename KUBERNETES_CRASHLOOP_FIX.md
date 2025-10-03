# Решение проблемы CrashLoopBackOff в Kubernetes

## Описание проблемы

При развёртывании микросервисов в Kubernetes через Helm возникали следующие ошибки:

1. **"exec format error"** для сервисов monolith и movies-service - несоответствие архитектуры Docker-образов
2. **PostgreSQL падал** из-за отсутствия переменной окружения PGDATA
3. **Kafka имел конфликт Cluster ID** из-за старых данных в PersistentVolume

## Корневые причины

### 1. Несоответствие архитектуры
- Docker-образы были собраны для архитектуры ARM64 (Apple Silicon M1/M2)
- Kubernetes кластер работает на архитектуре AMD64 (x86_64)
- При попытке запуска ARM64 бинарника на AMD64 системе возникает ошибка "exec format error"

### 2. Отсутствие PGDATA
- PostgreSQL требует явного указания директории для хранения данных через переменную окружения PGDATA
- Без этой переменной PostgreSQL не может инициализировать базу данных

### 3. Конфликт Kafka Cluster ID
- При пересоздании Kafka с использованием существующего PersistentVolume возникает конфликт Cluster ID
- Старые метаданные в volume конфликтуют с новой конфигурацией

## Примененные решения

### 1. Поддержка Multi-Architecture в Dockerfiles

Обновлены все Dockerfiles для поддержки кросс-платформенной сборки:

```dockerfile
# Build stage с поддержкой multi-arch
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

# Аргументы для кросс-компиляции
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

# Сборка для целевой платформы
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -installsuffix cgo -o app .

# Runtime stage для целевой платформы
FROM --platform=$TARGETPLATFORM alpine:latest
```

### 2. Исправление PostgreSQL

Добавлена переменная окружения PGDATA в `src/kubernetes/helm/templates/services/postgres.yaml`:

```yaml
env:
- name: PGDATA
  value: /var/lib/postgresql/data/pgdata
```

### 3. Улучшение конфигурации Kafka

Добавлены параметры для управления логами и предотвращения конфликтов:

```yaml
env:
- name: KAFKA_CLEANUP_POLICY
  value: "delete"
- name: KAFKA_LOG_RETENTION_HOURS
  value: "168"
- name: KAFKA_LOG_SEGMENT_BYTES
  value: "1073741824"
- name: KAFKA_LOG_RETENTION_CHECK_INTERVAL_MS
  value: "300000"
```

### 4. GitHub Actions Workflow для Multi-Arch

Обновлен `.github/workflows/docker-build-push.yml`:

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v2
  with:
    platforms: linux/amd64,linux/arm64

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v2
  with:
    platforms: linux/amd64,linux/arm64

- name: Build and push Docker image
  uses: docker/build-push-action@v4
  with:
    context: ./src/service
    platforms: linux/amd64,linux/arm64  # Multi-arch support
    push: true
    tags: ${{ steps.meta.outputs.tags }}
```

## Инструкция по развертыванию

### 1. Очистка старых ресурсов (если есть)

```bash
# Удаление namespace с застрявшими ресурсами
kubectl delete namespace cinemaabyss --ignore-not-found=true

# Если namespace застрял в состоянии Terminating
kubectl get pvc -n cinemaabyss -o name | xargs -I {} kubectl patch {} -n cinemaabyss -p '{"metadata":{"finalizers":null}}'
kubectl get namespace cinemaabyss -o json | jq '.spec.finalizers = []' | kubectl replace --raw "/api/v1/namespaces/cinemaabyss/finalize" -f -
```

### 2. Сборка и публикация образов

#### Вариант A: Через GitHub Actions
```bash
# Сделайте push в main ветку для автоматической сборки
git add .
git commit -m "Fix: Add multi-arch support for Docker images"
git push origin main
```

#### Вариант B: Локальная сборка
```bash
# Используйте предоставленный скрипт
./build-and-push-multiarch.sh
```

### 3. Развертывание через Helm

```bash
# Установка
helm install cinemaabyss ./src/kubernetes/helm -n cinemaabyss --create-namespace

# Проверка статуса
kubectl get pods -n cinemaabyss

# Просмотр логов при проблемах
kubectl logs <pod-name> -n cinemaabyss
```

### 4. Обновление существующего релиза

```bash
# Обновление с новыми образами
helm upgrade cinemaabyss ./src/kubernetes/helm -n cinemaabyss

# Принудительный перезапуск подов для загрузки новых образов
kubectl rollout restart deployment -n cinemaabyss
kubectl rollout restart statefulset -n cinemaabyss
```

## Проверка архитектуры

### Проверка архитектуры Kubernetes кластера
```bash
kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.nodeInfo.architecture}{"\n"}{end}'
```

### Проверка архитектуры Docker образа
```bash
docker inspect ghcr.io/nikpeg/cinema-microservices-migration/monolith:latest | jq '.[0].Architecture'
```

## Мониторинг и отладка

### Полезные команды для отладки

```bash
# Статус всех подов
kubectl get pods -n cinemaabyss -w

# Детальная информация о поде
kubectl describe pod <pod-name> -n cinemaabyss

# События в namespace
kubectl get events -n cinemaabyss --sort-by='.lastTimestamp'

# Проверка PVC
kubectl get pvc -n cinemaabyss

# Проверка готовности сервисов
kubectl get svc -n cinemaabyss
kubectl get endpoints -n cinemaabyss
```

## Предотвращение проблем в будущем

### Рекомендации

1. **Всегда используйте multi-arch сборку** для Docker образов при работе с разными архитектурами
2. **Тестируйте образы** на целевой архитектуре перед развертыванием
3. **Документируйте требования** к архитектуре в README проекта
4. **Используйте CI/CD** для автоматической сборки multi-arch образов
5. **Регулярно очищайте** старые PVC и PV для предотвращения конфликтов

### Checklist перед развертыванием

- [ ] Проверена архитектура целевого кластера
- [ ] Образы собраны для правильной архитектуры
- [ ] Все необходимые переменные окружения настроены
- [ ] Старые ресурсы очищены (если требуется)
- [ ] Helm values проверены и актуальны

## Дополнительные ресурсы

- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [Kubernetes Multi-Architecture Images](https://kubernetes.io/docs/concepts/containers/images/#multi-architecture-images)
- [Helm Best Practices](https://helm.sh/docs/chart_best_practices/)
