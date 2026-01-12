package alert

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


func HandleAlertSlackPreview(c *gin.Context) {
	stub := BuildAlertGroupFromEvents()
	c.JSON(http.StatusOK, gin.H{
		"display": stub.Display,
		"title":   stub.Title,
		"data":    stub.Data,
	})
}
