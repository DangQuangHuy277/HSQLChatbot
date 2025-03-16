package middleware

import (
	"HNLP/be/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

func Authenticate(s auth.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := ctx.GetHeader("Authorization")
		if token == "" || !strings.HasPrefix(token, "Bearer ") {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		}

		parsedToken, err := s.ValidateAndParseToken(token[7:])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		}

		if time.Now().After(time.Unix(int64(claims["exp"].(float64)), 0)) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		}

		ctx.Set("userId", claims["id"].(float64))
		ctx.Set("userCode", claims["username"].(string))
		ctx.Set("userRole", claims["role"].(string))

		ctx.Next()
	}
}

func HasAnyRole(role ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userRole, ok := ctx.Get("userRole")
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing role"})
		}

		// Bypass all check if it is admin
		if userRole == "admin" {
			ctx.Next()
			return
		}

		for _, r := range role {
			if r == userRole {
				ctx.Next()
				return
			}
		}

		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}
