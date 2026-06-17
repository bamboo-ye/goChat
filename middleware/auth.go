package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"goChat/models"
)

const UserIDKey = "authenticated_user_id"

func ExtractToken(c *gin.Context) string {
	if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	}
	return strings.TrimSpace(c.Query("token"))
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := models.FindUserByToken(ExtractToken(c))
		if user.ID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    -1,
				"message": "登录已失效，请重新登录",
			})
			return
		}
		c.Set(UserIDKey, user.ID)
		c.Next()
	}
}

func UserID(c *gin.Context) uint {
	value, ok := c.Get(UserIDKey)
	if !ok {
		return 0
	}
	userID, _ := value.(uint)
	return userID
}
