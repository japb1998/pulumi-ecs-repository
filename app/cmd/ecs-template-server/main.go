package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("health", func(ctx *gin.Context) {
		ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
			"healthy": true,
		})
	})

	if err := r.Run(":80"); err != nil {
		log.Fatal(err)
	}
}
