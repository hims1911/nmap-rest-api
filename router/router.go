package router

import (
	apiv1 "nmap-rest-api/api/v1"
	"nmap-rest-api/middleware"

	_ "nmap-rest-api/docs" // or replace with your actual module name

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Logger())
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Use(otelgin.Middleware("nmap-api"))
	r.POST("/scan", apiv1.HandleScanRequest)
	r.GET("/results/:host", apiv1.GetScanResults)
	r.GET("/scan/status/:scan_id", apiv1.GetScanStatus)
	return r
}
