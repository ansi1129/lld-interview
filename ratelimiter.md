# 🚦 Rate Limiter — Low Level Design (LLD)

> A production-ready Rate Limiter system designed using the **Strategy Pattern**, supporting multiple algorithms with O(1) decision-making, thread safety, and distributed scalability.

---

## 📋 Table of Contents

- [Requirements](#-requirements--clarifications)
- [Approach](#-approach)
- [Models & Design](#-models--design)
- [DB Schema](#-db-schema-production)
- [Algorithms](#-algorithms-core-logic)
- [Complexity](#-complexity)
- [Design Patterns](#-design-patterns-used)
- [SOLID Principles](#-solid-principles)
- [Scaling & Improvements](#-scaling--improvements)
- [Conclusion](#-final-conclusion)

---

## 📌 Requirements & Clarifications

### ✅ Functional
- Allow / reject requests based on rate limits
- Support multiple algorithms:
  - Fixed Window
  - Sliding Window (Log & Counter)
  - Token Bucket
  - Leaky Bucket
- Per-key limiting (User / API / IP)
- Dynamic rule configuration

### ⚠️ Non-Functional
- Low latency — **O(1) decision**
- High throughput
- Thread-safe
- Scalable (distributed support via Redis)

---

## 🧠 Approach

Uses the **Strategy Pattern** to swap algorithms at runtime without changing the core service.

```
RateLimiterService → Strategy → Allow(key)
```

### Request Flow

```
1. Client request arrives
2. RateLimiterService identifies the key (user / IP / API)
3. Fetch the limiter associated with the key
4. Call strategy.Allow(key)
5. Return allow / reject to client
```

---

## 🏗️ Models & Design

### `RateLimitConfig`
```go
type RateLimitConfig struct {
    Key        string
    Limit      int
    WindowSize time.Duration
    Algorithm  string
}
```
> Configures the limiter per key (stored in DB or Redis).

---

### `RateLimiter` Interface (Strategy Abstraction)
```go
type RateLimiter interface {
    Allow(key string) bool
}
```

---

### `RateLimiterService`
```go
type RateLimiterService struct {
    limiters map[string]RateLimiter
}
```
> Maintains a map of key → limiter strategy.

---

### Algorithm Implementations (Strategies)

| Strategy | Struct |
|---|---|
| Fixed Window | `FixedWindowLimiter` |
| Sliding Window Log | `SlidingWindowLimiter` |
| Sliding Window Counter | `SlidingWindowCounterLimiter` |
| Token Bucket | `TokenBucketLimiter` |
| Leaky Bucket | `LeakyBucketLimiter` |

---

### Internal Storage Per Key

| Algorithm | Data Structure |
|---|---|
| Fixed Window | `counter` + `timestamp` |
| Sliding Window Log | `timestamp list` |
| Sliding Window Counter | `current` + `previous` counters |
| Token Bucket | `tokens` + `lastRefill` |
| Leaky Bucket | `queue size` |

---

## 🗄️ DB Schema (Production)

```sql
rate_limit_config (
    key         VARCHAR,
    limit       INT,
    window_size INTERVAL,
    algorithm   VARCHAR
)
```
> Stored in **DB / Redis** for dynamic, distributed configuration.

---

## 🧠 Algorithms (Core Logic)

### 🔹 Fixed Window
```
1. If window expired → reset count
2. If count < limit  → allow
3. Else              → reject
```

### 🔹 Sliding Window Log
```
1. Remove timestamps outside the window
2. If count < limit → allow
3. Else             → reject
```

### 🔹 Sliding Window Counter
```
1. Maintain current + previous window counts
2. Compute weighted count = prev × (overlap ratio) + current
3. If weighted count < limit → allow
```

### 🔹 Token Bucket
```
1. Refill tokens based on elapsed time
2. If tokens > 0 → allow (consume 1 token)
3. Else          → reject
```
> ✅ Best for **burst traffic handling**

### 🔹 Leaky Bucket
```
1. Leak (process) requests at a fixed rate
2. If queue size < capacity → allow (enqueue)
3. Else                     → reject
```
> ✅ Best for **strict, uniform rate enforcement**

---

## 🔍 Complexity

| Operation | Time Complexity |
|---|---|
| `Allow()` | O(1) |
| Sliding Window Log cleanup | O(N) |
| Space | O(N keys) |

---

## 🎯 Design Patterns Used

| Pattern | Usage |
|---|---|
| **Strategy** | Swap algorithms at runtime |
| **Factory** | Limiter creation based on config |
| **Singleton** *(optional)* | Single service instance |

---

## ✅ SOLID Principles

| Principle | Application |
|---|---|
| **SRP** | Each algorithm in its own struct |
| **OCP** | Add new algorithms without modifying existing code |
| **DIP** | `RateLimiterService` depends on the `RateLimiter` interface |
| **LSP** | All strategy implementations are interchangeable |

---

## 🚀 Scaling & Improvements

- 🔴 **Redis** for distributed rate limiting across nodes
- 📊 **Sliding Window** for higher accuracy over Fixed Window
- 🔀 **Sharding keys** for horizontal scaling
- 📝 **Async logging** to reduce latency overhead
- 🔗 **Per-endpoint rate limiting** for finer-grained control

---

## 🏁 Final Conclusion

> *"I designed the rate limiter using a strategy-based approach to support multiple algorithms, ensuring O(1) decision-making and thread safety. The system is extensible and can be scaled using Redis for distributed environments."*

---

## 🔥 Key Insight

> **Token Bucket** is preferred for burst handling, while **Sliding Window** provides stricter, more accurate enforcement.

---

## 📁 Project Structure

```
rate-limiter/
├── config/
│   └── rate_limit_config.go       # RateLimitConfig struct
├── interfaces/
│   └── rate_limiter.go            # RateLimiter interface
├── service/
│   └── rate_limiter_service.go    # Core service
├── algorithms/
│   ├── fixed_window.go
│   ├── sliding_window_log.go
│   ├── sliding_window_counter.go
│   ├── token_bucket.go
│   └── leaky_bucket.go
├── factory/
│   └── limiter_factory.go         # Factory for limiter creation
└── main.go
```