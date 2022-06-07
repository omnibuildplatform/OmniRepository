package application

import (
	"github.com/gin-gonic/gin"
	"github.com/omnibuildplatform/omni-repository/application/controller"
	"github.com/omnibuildplatform/omni-repository/docs"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func AddRoutes(r *gin.Engine) {
	// status
	r.GET("/health", controller.AppHealth)

	docs.SwaggerInfo.BasePath = "/images"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// not found routes
	r.NoRoute(func(c *gin.Context) {
		c.Data(404, "text/plain", []byte("not found"))
	})
}
