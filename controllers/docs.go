package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Docs documents controller
func Docs(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "docs/index.html", gin.H{})
}
