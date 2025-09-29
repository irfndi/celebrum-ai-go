package services

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	HitRate     float64   `json:"hit_rate"`
	TotalOps    int64     `json:"total_ops"`
	LastUpdated time.Time `json:"last_updated"`
}

// CacheMetrics represents detailed cache metrics by category
type CacheMetrics struct {
	Overall          CacheStats            `json:"overall"`
	ByCategory       map[string]CacheStats `json:"by_category"`
	RedisInfo        map[string]string     `json:"redis_info"`
	MemoryUsage      int64                 `json:"memory_usage_bytes"`
	ConnectedClients int64                 `json:"connected_clients"`
	KeyCount         int64                 `json:"key_count"`
}

// CacheAnalyticsService tracks cache performance metrics
type CacheAnalyticsService struct {
	redisClient *redis.Client
	stats       map[string]*CacheStats
	mu          sync.RWMutex
}

// NewCacheAnalyticsService creates a new cache analytics service
func NewCacheAnalyticsService(redisClient *redis.Client) *CacheAnalyticsService {
	return &CacheAnalyticsService{
		redisClient: redisClient,
		stats:       make(map[string]*CacheStats),
	}
}

// RecordHit records a cache hit for the given category
func (c *CacheAnalyticsService) RecordHit(category string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stats[category] == nil {
		c.stats[category] = &CacheStats{}
	}

	c.stats[category].Hits++
	c.stats[category].TotalOps++
	c.stats[category].HitRate = float64(c.stats[category].Hits) / float64(c.stats[category].TotalOps)
	c.stats[category].LastUpdated = time.Now()

	// Also update overall stats
	if c.stats["overall"] == nil {
		c.stats["overall"] = &CacheStats{}
	}
	c.stats["overall"].Hits++
	c.stats["overall"].TotalOps++
	c.stats["overall"].HitRate = float64(c.stats["overall"].Hits) / float64(c.stats["overall"].TotalOps)
	c.stats["overall"].LastUpdated = time.Now()
}

// RecordMiss records a cache miss for the given category
func (c *CacheAnalyticsService) RecordMiss(category string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stats[category] == nil {
		c.stats[category] = &CacheStats{}
	}

	c.stats[category].Misses++
	c.stats[category].TotalOps++
	c.stats[category].HitRate = float64(c.stats[category].Hits) / float64(c.stats[category].TotalOps)
	c.stats[category].LastUpdated = time.Now()

	// Also update overall stats
	if c.stats["overall"] == nil {
		c.stats["overall"] = &CacheStats{}
	}
	c.stats["overall"].Misses++
	c.stats["overall"].TotalOps++
	c.stats["overall"].HitRate = float64(c.stats["overall"].Hits) / float64(c.stats["overall"].TotalOps)
	c.stats["overall"].LastUpdated = time.Now()
}

// GetStats returns cache statistics for a specific category
func (c *CacheAnalyticsService) GetStats(category string) CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if stats, exists := c.stats[category]; exists {
		return *stats
	}
	return CacheStats{}
}

// GetAllStats returns all cache statistics
func (c *CacheAnalyticsService) GetAllStats() map[string]CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]CacheStats)
	for category, stats := range c.stats {
		result[category] = *stats
	}
	return result
}

// GetMetrics returns comprehensive cache metrics including Redis info
func (c *CacheAnalyticsService) GetMetrics(ctx context.Context) (*CacheMetrics, error) {
	c.mu.RLock()
	allStats := make(map[string]CacheStats)
	for category, stats := range c.stats {
		allStats[category] = *stats
	}
	c.mu.RUnlock()

	// Get Redis info
	redisInfo, err := c.redisClient.Info(ctx, "memory", "clients", "keyspace").Result()
	if err != nil {
		return nil, err
	}

	// Parse Redis info
	infoMap := c.parseRedisInfo(redisInfo)

	// Get memory usage
	memoryUsage, _ := c.redisClient.MemoryUsage(ctx, "*").Result()

	// Get connected clients
	clientList, _ := c.redisClient.ClientList(ctx).Result()
	connectedClients := int64(len(clientList))

	// Get key count
	keyCount, _ := c.redisClient.DBSize(ctx).Result()

	metrics := &CacheMetrics{
		ByCategory:       allStats,
		RedisInfo:        infoMap,
		MemoryUsage:      memoryUsage,
		ConnectedClients: connectedClients,
		KeyCount:         keyCount,
	}

	// Set overall stats
	if overall, exists := allStats["overall"]; exists {
		metrics.Overall = overall
	}

	return metrics, nil
}

// parseRedisInfo parses Redis INFO command output
func (c *CacheAnalyticsService) parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	
	if info == "" {
		return result
	}

	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

// ResetStats resets all cache statistics
func (c *CacheAnalyticsService) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stats = make(map[string]*CacheStats)
}

// StartPeriodicReporting starts periodic reporting of cache stats to Redis
func (c *CacheAnalyticsService) StartPeriodicReporting(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.reportStats(ctx)
			}
		}
	}()
}

// reportStats reports current stats to Redis for persistence
func (c *CacheAnalyticsService) reportStats(ctx context.Context) {
	allStats := c.GetAllStats()
	statsJSON, err := json.Marshal(allStats)
	if err != nil {
		return
	}

	// Store stats in Redis with 24 hour TTL
	c.redisClient.Set(ctx, "cache:analytics:stats", statsJSON, 24*time.Hour)
}
