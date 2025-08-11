package uiapi

import (
	clusterapi "NeuroController/interfaces/cluster_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/ingress/list/all
func HandleGetAllIngresses(c *gin.Context) {
	ings, err := clusterapi.GetAllIngresses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ings)
}

// GET /uiapi/ingress/list/by-namespace/:ns
func HandleGetIngressesByNamespace(c *gin.Context) {
	ns := c.Param("ns")
	ings, err := clusterapi.GetIngressesByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ings)
}

// GET /uiapi/ingress/detail/:ns/:name
func HandleGetIngressByName(c *gin.Context) {
	ns := c.Param("ns")
	name := c.Param("name")
	ing, err := clusterapi.GetIngressByName(c.Request.Context(), ns, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ing)
}

// GET /uiapi/ingress/list/ready
func HandleGetReadyIngresses(c *gin.Context) {
	ings, err := clusterapi.GetReadyIngresses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ings)
}