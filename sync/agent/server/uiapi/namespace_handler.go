package uiapi

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/namespace/list
func HandleGetAllNamespaces(c *gin.Context) {
	namespaces, err := uiapi.GetAllNamespaces(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, namespaces)
}