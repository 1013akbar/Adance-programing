```mermaid
flowchart LR
    Client[Frontend Client] --> OrderAPI[Order Service API - REST]
    OrderAPI --> RateLimit[Rate Limiter<br/>Redis-based<br/>10 req/min]
    RateLimit --> Cache[Redis Cache<br/>TTL: 5min<br/>Cache-aside]
    Cache --> OrderDB[(Order DB<br/>PostgreSQL)]
    OrderAPI --> PaymentGRPC[Payment Service - gRPC]

    PaymentGRPC --> PaymentDB[(Payment DB<br/>PostgreSQL)]
    PaymentGRPC --> RabbitMQ[(RabbitMQ<br/>payment.completed)]

    RabbitMQ --> NotificationSVC[Notification Service<br/>Background Worker]
    NotificationSVC --> RedisIdem[Redis Idempotency<br/>notification:{id}]
    NotificationSVC --> EmailAdapter[Email Adapter<br/>SMTP/Simulated]
    NotificationSVC --> RetryQueue[Retry Queue<br/>Exponential Backoff]

    subgraph "Performance Layer"
        Cache
        RedisIdem
        RetryQueue
        RateLimit
    end

    subgraph "External Integrations"
        EmailAdapter
    end

    subgraph "Message Queue"
        RabbitMQ
    end

    subgraph "Databases"
        OrderDB
        PaymentDB
    end
```