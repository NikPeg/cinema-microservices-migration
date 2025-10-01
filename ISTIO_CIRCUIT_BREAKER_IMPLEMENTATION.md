# Реализация Circuit Breaker с использованием Istio

## Оглавление
1. [Описание задачи](#описание-задачи)
2. [Архитектура решения](#архитектура-решения)
3. [Установка и настройка](#установка-и-настройка)
4. [Конфигурация Circuit Breaker](#конфигурация-circuit-breaker)
5. [Тестирование с Fortio](#тестирование-с-fortio)
6. [Результаты тестирования](#результаты-тестирования)
7. [Анализ и выводы](#анализ-и-выводы)

## Описание задачи

Для повышения надёжности и безопасности микросервисной архитектуры CinemaAbyss было необходимо:
- Развернуть Istio Service Mesh
- Настроить Circuit Breaker для сервисов `monolith` и `movies-service`
- Провести нагрузочное тестирование с помощью Fortio
- Проверить эффективность Circuit Breaker

## Архитектура решения

### Компоненты системы

```
┌─────────────────────────────────────────────────────────┐
│                    Istio Control Plane                   │
│                        (istiod)                          │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                    Data Plane (Envoy)                    │
├──────────────────┬──────────────────┬──────────────────┤
│   Monolith       │  Movies Service  │  Events Service  │
│   + Sidecar      │   + Sidecar      │   + Sidecar      │
└──────────────────┴──────────────────┴──────────────────┘
```

### Circuit Breaker Pattern

Circuit Breaker защищает сервисы от каскадных сбоев путем:
- Ограничения количества одновременных подключений
- Изоляции неисправных экземпляров
- Автоматического восстановления после сбоев

## Установка и настройка

### 1. Проверка Istio

```bash
# Проверка установки Istio
kubectl get namespace istio-system
kubectl get pods -n istio-system
```

### 2. Включение Istio Injection

```bash
# Включение автоматической инъекции sidecar для namespace
kubectl label namespace cinemaabyss istio-injection=enabled --overwrite

# Перезапуск подов для добавления sidecar
kubectl rollout restart deployment -n cinemaabyss
```

### 3. Проверка sidecar контейнеров

```bash
kubectl get pods -n cinemaabyss
# Должны видеть 2/2 READY для каждого пода (основной контейнер + Envoy sidecar)
```

## Конфигурация Circuit Breaker

### DestinationRule для Monolith

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: monolith-circuit-breaker
  namespace: cinemaabyss
spec:
  host: monolith
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 10        # Максимум TCP соединений
      http:
        http1MaxPendingRequests: 10  # Максимум ожидающих HTTP запросов
        http2MaxRequests: 20          # Максимум активных HTTP/2 запросов
        maxRequestsPerConnection: 2   # Максимум запросов на соединение
    outlierDetection:
      consecutiveErrors: 5            # Количество ошибок для изоляции
      interval: 30s                   # Интервал анализа
      baseEjectionTime: 30s          # Время изоляции
      maxEjectionPercent: 50         # Максимум изолированных хостов
      minHealthPercent: 30           # Минимум здоровых хостов
```

### DestinationRule для Movies Service

```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: movies-service-circuit-breaker
  namespace: cinemaabyss
spec:
  host: movies-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 10
      http:
        http1MaxPendingRequests: 10
        http2MaxRequests: 20
        maxRequestsPerConnection: 2
    outlierDetection:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
      minHealthPercent: 30
```

### Параметры Circuit Breaker

| Параметр | Значение | Описание |
|----------|----------|----------|
| `maxConnections` | 10 | Максимальное количество TCP соединений к сервису |
| `http1MaxPendingRequests` | 10 | Максимум ожидающих HTTP/1.1 запросов |
| `http2MaxRequests` | 20 | Максимум активных HTTP/2 запросов |
| `maxRequestsPerConnection` | 2 | Максимум запросов через одно соединение |
| `consecutiveErrors` | 5 | Порог последовательных ошибок для срабатывания |
| `interval` | 30s | Интервал проверки состояния |
| `baseEjectionTime` | 30s | Базовое время изоляции неисправного экземпляра |
| `maxEjectionPercent` | 50 | Максимальный процент изолированных экземпляров |

## Тестирование с Fortio

### Установка Fortio

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: fortio
  namespace: cinemaabyss
  annotations:
    sidecar.istio.io/inject: "false"  # Отключаем sidecar для тестового пода
spec:
  containers:
  - name: fortio
    image: fortio/fortio:latest
    command: ["/usr/bin/fortio"]
    args: ["server"]
```

### Команды тестирования

#### Тест 1: Monolith Service (умеренная нагрузка)
```bash
kubectl exec fortio -n cinemaabyss -- \
  fortio load -c 20 -qps 0 -n 100 -loglevel Warning \
  http://monolith:8080/health
```

**Параметры:**
- `-c 20`: 20 параллельных соединений
- `-qps 0`: Максимальная скорость (без ограничений)
- `-n 100`: 100 запросов всего

#### Тест 2: Movies Service (высокая нагрузка)
```bash
kubectl exec fortio -n cinemaabyss -- \
  fortio load -c 50 -qps 0 -n 200 -loglevel Warning \
  http://movies-service:8081/api/movies/health
```

**Параметры:**
- `-c 50`: 50 параллельных соединений
- `-n 200`: 200 запросов всего

## Результаты тестирования

### Monolith Service

```
Fortio 1.72.0 running at 0 queries per second
Starting at max qps with 20 thread(s) for exactly 100 calls

Results:
- Code 200: 98 (98.0%)  ✅ Успешные запросы
- Code 503: 2 (2.0%)    ⚠️ Circuit Breaker сработал
- QPS: 335.86
- Avg latency: 54.564 ms
- Min latency: 4.76 ms
- Max latency: 112.55 ms
```

**Анализ:**
- Circuit Breaker эффективно ограничил нагрузку
- 98% запросов успешно обработаны
- Только 2% отклонены для защиты сервиса

### Movies Service

```
Fortio 1.72.0 running at 0 queries per second
Starting at max qps with 50 thread(s) for exactly 200 calls

Results:
- Code 200: 110 (55.0%) ✅ Успешные запросы
- Code 503: 90 (45.0%)  ⚠️ Circuit Breaker сработал
- QPS: 537.9
- Avg latency: 50.773 ms
- Min latency: 3.86 ms
- Max latency: 254.97 ms
```

**Анализ:**
- При высокой нагрузке Circuit Breaker активно защищает сервис
- 45% запросов отклонены для предотвращения перегрузки
- Сервис остается доступным для 55% запросов

## Анализ и выводы

### Эффективность Circuit Breaker

1. **Защита от перегрузки**
   - При умеренной нагрузке (20 соединений) - 2% отклонений
   - При высокой нагрузке (50 соединений) - 45% отклонений
   - Сервисы остаются доступными даже под нагрузкой

2. **Быстрое реагирование**
   - Circuit Breaker срабатывает мгновенно при превышении лимитов
   - Возвращает HTTP 503 без ожидания таймаута

3. **Предотвращение каскадных сбоев**
   - Изоляция проблемных экземпляров
   - Автоматическое восстановление через 30 секунд

### Преимущества реализации

✅ **Повышенная надежность**: Защита от каскадных сбоев
✅ **Улучшенная производительность**: Быстрый отказ вместо долгих таймаутов
✅ **Автоматическое восстановление**: Без ручного вмешательства
✅ **Гибкая настройка**: Индивидуальные параметры для каждого сервиса
✅ **Наблюдаемость**: Интеграция с Kiali и Jaeger для мониторинга

### Рекомендации по настройке

1. **Для критичных сервисов**:
   - Увеличить `maxConnections` и `http2MaxRequests`
   - Уменьшить `consecutiveErrors` для более быстрого реагирования

2. **Для некритичных сервисов**:
   - Уменьшить лимиты для экономии ресурсов
   - Увеличить `baseEjectionTime` для более длительной изоляции

3. **Мониторинг**:
   - Регулярно проверять метрики в Kiali
   - Настроить алерты на частые срабатывания Circuit Breaker

## Команды для управления

### Просмотр конфигурации
```bash
# DestinationRules
kubectl get destinationrules -n cinemaabyss

# VirtualServices
kubectl get virtualservices -n cinemaabyss

# Детальная информация
kubectl describe destinationrule monolith-circuit-breaker -n cinemaabyss
```

### Обновление конфигурации
```bash
# Применение изменений
kubectl apply -f src/kubernetes/istio-circuit-breaker.yaml

# Удаление конфигурации
kubectl delete -f src/kubernetes/istio-circuit-breaker.yaml
```

### Мониторинг
```bash
# Просмотр метрик Envoy
kubectl exec <pod-name> -c istio-proxy -n cinemaabyss -- \
  curl -s localhost:15000/stats/prometheus | grep circuit

# Логи Envoy proxy
kubectl logs <pod-name> -c istio-proxy -n cinemaabyss
```

## Заключение

Успешно реализован Circuit Breaker паттерн с использованием Istio для микросервисов CinemaAbyss. Тестирование с Fortio подтвердило эффективность защиты от перегрузок и каскадных сбоев. Система готова к production использованию с возможностью тонкой настройки параметров под конкретные требования нагрузки.
