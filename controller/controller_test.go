package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gotest.tools/v3/assert"
)

func TestRateLimiterController(t *testing.T) {
	w := httptest.NewRecorder()
	// var status int
	// var body interface{}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	// c.JSON = func(stat int, object interface{}) {
	// 	status = stat
	// 	body = object
	// }
	RateLimiterController(c)
	assert.Equal(t, 200, w.Code)
}
