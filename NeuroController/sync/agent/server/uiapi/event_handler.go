package uiapi

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/event/list/all
func HandleGetAllEvents(c *gin.Context) {
	events, err := uiapi.GetAllEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// GET /uiapi/event/list/by-namespace/:ns
func HandleGetEventsByNamespace(c *gin.Context) {
	ns := c.Param("ns")
	events, err := uiapi.GetEventsByNamespace(c.Request.Context(), ns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// GET /uiapi/event/list/by-object/:ns/:kind/:name
func HandleGetEventsByObject(c *gin.Context) {
	ns := c.Param("ns")
	kind := c.Param("kind")
	name := c.Param("name")
	events, err := uiapi.GetEventsByInvolvedObject(c.Request.Context(), ns, kind, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// GET /uiapi/event/stats/type-count
func HandleGetEventTypeCounts(c *gin.Context) {
	counts, err := uiapi.GetEventTypeCounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, counts)
}