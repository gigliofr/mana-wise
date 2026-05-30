# MongoDB Optimization Guide

## Connection Pool Configuration

### Recommended Settings
- **minPoolSize**: 5 (minimum connections to maintain)
- **maxPoolSize**: 50 (maximum concurrent connections)
- **maxIdleTimeMS**: 60000 (1 minute idle timeout)
- **serverSelectionTimeoutMS**: 30000 (30 second server selection timeout)

### Connection String Example
```
mongodb+srv://user:password@cluster.mongodb.net/manawise?minPoolSize=5&maxPoolSize=50&maxIdleTimeMS=60000
```

## Critical Database Indexes

### Required Indexes for Optimal Performance

#### 1. Cards Collection
```javascript
// Index: card name lookups (high cardinality)
db.cards.createIndex({ "name": 1 })

// Index: card types and properties
db.cards.createIndex({ "type_line": 1, "colors": 1 })

// Index: scryfall_id for data synchronization
db.cards.createIndex({ "scryfall_id": 1 }, { unique: true })
```

#### 2. Users Collection
```javascript
// Index: email-based login (unique)
db.users.createIndex({ "email": 1 }, { unique: true })

// Index: created_at for analytics
db.users.createIndex({ "created_at": 1 })

// Index: plan for freemium gating
db.users.createIndex({ "plan": 1, "daily_analyses_count": 1 })
```

#### 3. Decks Collection
```javascript
// Index: user's decks (most common query)
db.decks.createIndex({ "user_id": 1, "_id": -1 })

// Index: deck name search within user
db.decks.createIndex({ "user_id": 1, "name": 1 })

// Index: created_at for sorting
db.decks.createIndex({ "user_id": 1, "created_at": -1 })

// Index: format and legality queries
db.decks.createIndex({ "format": 1, "legality_status": 1 })
```

#### 4. SharedAnalysisLinks Collection
```javascript
// Index: token-based access (high frequency)
db.shared_analysis_links.createIndex({ "token": 1 }, { unique: true })

// Index: expiration for cleanup
db.shared_analysis_links.createIndex({ "expires_at": 1 }, { expireAfterSeconds: 0 })
```

#### 5. PasswordResetTokens Collection
```javascript
// Index: token lookup
db.password_reset_tokens.createIndex({ "token": 1 }, { unique: true })

// Index: TTL cleanup (auto-delete after expiration)
db.password_reset_tokens.createIndex({ "created_at": 1 }, { expireAfterSeconds: 3600 })
```

#### 6. OTA Updates Collection
```javascript
// Index: version lookups
db.ota_updates.createIndex({ "version": 1 }, { unique: true })

// Index: release date queries
db.ota_updates.createIndex({ "release_date": -1 })
```

## Query Timeout Configuration

### Standard Timeouts (Configured in Go)
- **Read Operations**: 10 seconds (card lookups, deck retrieval)
- **Write Operations**: 5 seconds (user updates, deck saves)
- **Index Operations**: 30 seconds (initial index creation)
- **Request Handler Timeout**: 30 seconds (end-to-end for API endpoints)

### Usage in Code
```go
// Read timeout
ctx, cancel := mongodb.WithReadTimeout(context.Background())
defer cancel()
user, err := userRepo.FindByID(ctx, userID)

// Write timeout
ctx, cancel := mongodb.WithWriteTimeout(context.Background())
defer cancel()
err := deckRepo.Update(ctx, deck)
```

## Monitoring and Optimization

### Key Metrics to Monitor
1. **Connection Pool Utilization**: Target 30-70% of maxPoolSize
2. **Query Performance**: 95th percentile should be < 100ms for read, < 50ms for write
3. **Index Hit Ratio**: Aim for > 95% of queries using indexes
4. **Memory Usage**: Monitor for memory pressure, adjust pool size if needed

### Slow Query Logs
Enable MongoDB slow query logging to identify optimization opportunities:
```
setParameter: {logLevel: 1, slowms: 100}
```

## Deployment Checklist

- [ ] Create all required indexes before going to production
- [ ] Verify connection pool settings in environment configuration
- [ ] Enable monitoring and alerting for slow queries
- [ ] Test failover behavior with circuit breaker enabled
- [ ] Validate query timeout values under high load
- [ ] Document custom indexes for your deployment

## Circuit Breaker Integration

The application automatically uses circuit breaker protection for MongoDB:
- **Failure Threshold**: 5 consecutive failures open the circuit
- **Recovery Timeout**: 30 seconds before attempting half-open state
- **Success Threshold**: 1 successful request closes the circuit in half-open state

When circuit is open, requests fail fast with `ERR_CIRCUIT_BREAKER_OPEN` error code.

## Performance Tuning Guide

### For High Read Workloads
- Increase `maxPoolSize` to 75-100
- Add read preference replicas
- Consider caching layer for frequently accessed documents

### For High Write Workloads
- Use write concern "majority" for consistency
- Batch writes where possible
- Monitor oplog size

### For Mixed Workloads
- Default settings (minPool: 5, maxPool: 50) are suitable
- Use separate read and write replicas if available
