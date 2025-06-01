package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	secret := os.Getenv("JWT_SECRET")
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No bearer token"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Вот тут — разрешить HS512!
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			// Можно проверить строго:
			if token.Method.Alg() != "HS512" {
				return nil, fmt.Errorf("Only HS512 is allowed")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims"})
			return
		}

		var isAdmin bool
		if rawRoles, exists := claims["roles"]; exists {
			switch roles := rawRoles.(type) {
			case []interface{}:
				for _, r := range roles {
					if s, ok := r.(string); ok && s == "ADMIN" {
						isAdmin = true
						break
					}
				}
			case []string:
				for _, s := range roles {
					if s == "ADMIN" {
						isAdmin = true
						break
					}
				}
			case string:
				if roles == "ADMIN" {
					isAdmin = true
				}
			}
		}

		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access only"})
			return
		}

		// Всё ок — пропускаем дальше
		c.Set("user_id", claims["sub"])
		c.Next()
	}
}
