package apisix

import "github.com/gin-gonic/gin"

func InitRouter(r *gin.RouterGroup) {
	group := r.Group("/apisix")
	{
		group.Any("/admin/*path", ProxyAdminAPI)
	}
}
