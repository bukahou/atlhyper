package uiapi

import (
	clusterapi "NeuroController/internal/interfaces/cluster_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/namespace/list
func HandleGetAllNamespaces(c *gin.Context) {
	namespaces, err := clusterapi.GetAllNamespaces(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, namespaces)
}