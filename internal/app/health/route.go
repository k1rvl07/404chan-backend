package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg gin.IRoutes, ctrl *HealthController) {
	rg.GET("/health", ctrl.Check)
}
