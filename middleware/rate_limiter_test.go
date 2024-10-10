package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"gotest.tools/v3/assert"
)

func TestNewRedisRateLimiter(t *testing.T) {
	client, _ := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	assert.Equal(t, client, limiter.client)
}

func TestLock(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	mock.ExpectSetNX("test-lock", 1, 5*time.Minute).SetVal(true)
	res, err := limiter.Lock(context.Background(), "test")
	assert.Equal(t, nil, err)
	assert.Equal(t, true, res)
}

func TestUnlock(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	mock.ExpectDel("test-lock").SetVal(1)
	err := limiter.Unlock(context.Background(), "test")
	assert.Equal(t, nil, err)
}

func TestAddLimiter(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	mock.ExpectIncr("test").SetVal(1)
	mock.ExpectExpire("test", 1*time.Minute).SetVal(true)
	err := limiter.AddLimiter(context.Background(), "test", 1*time.Minute)
	assert.Equal(t, nil, err)

	mock.ClearExpect()
	mock.ExpectIncr("test").SetVal(2)
	err = limiter.AddLimiter(context.Background(), "test", 1*time.Minute)
	assert.Equal(t, nil, err)

	mock.ClearExpect()
	mock.ExpectIncr("test").SetErr(errors.New("err incr"))
	err = limiter.AddLimiter(context.Background(), "test", 1*time.Minute)
	assert.Error(t, err, "err incr")

	mock.ClearExpect()
	mock.ExpectIncr("test").SetVal(1)
	mock.ExpectExpire("test", 1*time.Minute).SetErr(errors.New("err expired"))
	err = limiter.AddLimiter(context.Background(), "test", 1*time.Minute)
	assert.Error(t, err, "err expired")
}

func TestIsBellowLimit(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	mock.ExpectGet("test").SetVal("1")
	isLimit := limiter.IsBellowLimit(context.Background(), "test", 5)
	assert.Equal(t, true, isLimit)

	mock.ClearExpect()
	mock.ExpectGet("test").SetVal("5")
	isLimit = limiter.IsBellowLimit(context.Background(), "test", 5)
	assert.Equal(t, false, isLimit)

	mock.ClearExpect()
	mock.ExpectGet("test").SetErr(redis.Nil)
	isLimit = limiter.IsBellowLimit(context.Background(), "test", 5)
	assert.Equal(t, true, isLimit)

	mock.ClearExpect()
	mock.ExpectGet("test").SetErr(errors.New("err get"))
	isLimit = limiter.IsBellowLimit(context.Background(), "test", 5)
	assert.Equal(t, false, isLimit)
}

func TestRateLimitChecker(t *testing.T) {
	client, mock := redismock.NewClientMock()
	limiter := NewRedisRateLimiter(client)

	mock.ExpectSetNX("1234:-lock", 1, 5*time.Minute).SetVal(true)
	mock.ExpectGet("1234:").SetVal("1")
	mock.ExpectIncr("1234:").SetVal(2)
	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("x-api-key", "1234")
	engine.Use(limiter.RateLimitChecker())
	engine.GET("/", func(ctx *gin.Context) {
	})
	engine.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Result().StatusCode)

	mock.ClearExpect()
	mock.ExpectSetNX("1234:-lock", 1, 5*time.Minute).SetVal(true)
	mock.ExpectGet("1234:").SetVal("1")
	mock.ExpectIncr("1234:").SetErr(errors.New("err incr"))
	w = httptest.NewRecorder()
	_, engine = gin.CreateTestContext(w)
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("x-api-key", "1234")
	engine.Use(limiter.RateLimitChecker())
	engine.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Result().StatusCode)

	mock.ClearExpect()
	mock.ExpectSetNX("1234:-lock", 1, 5*time.Minute).SetVal(true)
	mock.ExpectGet("1234:").SetVal("5")
	w = httptest.NewRecorder()
	_, engine = gin.CreateTestContext(w)
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("x-api-key", "1234")
	engine.Use(limiter.RateLimitChecker())
	engine.ServeHTTP(w, req)
	assert.Equal(t, 429, w.Result().StatusCode)

	mock.ClearExpect()
	mock.ExpectSetNX("1234:-lock", 1, 5*time.Minute).SetVal(false)
	w = httptest.NewRecorder()
	_, engine = gin.CreateTestContext(w)
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("x-api-key", "1234")
	engine.Use(limiter.RateLimitChecker())
	engine.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Result().StatusCode)

	mock.ClearExpect()
	mock.ExpectSetNX("1234:-lock", 1, 5*time.Minute).SetErr(errors.New("err set nx"))
	w = httptest.NewRecorder()
	_, engine = gin.CreateTestContext(w)
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("x-api-key", "1234")
	engine.Use(limiter.RateLimitChecker())
	engine.ServeHTTP(w, req)
	assert.Equal(t, 500, w.Result().StatusCode)
}
