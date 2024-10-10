package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func RateLimiterController(ctx *gin.Context) {
	log.Debug().Msg("HANDLING REQUEST")
	ctx.JSON(http.StatusOK, map[string]string{"message": "ok"})
}
