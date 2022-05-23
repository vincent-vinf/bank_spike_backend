package jwtx

import (
	"bank_spike_backend/internal/db"
	"bank_spike_backend/internal/util/config"
	"errors"
	"log"
	"strconv"
	"time"

	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

const (
	IdentityKey = "id"
	appRealm    = "bankSpike"
)

var (
	authMiddleware *jwt.GinJWTMiddleware
)

type loginForm struct {
	Phone  string `form:"phone" json:"phone" binding:"required"`
	Passwd string `form:"passwd" json:"passwd" binding:"required"`
}

type RegisterForm struct {
	Username string `form:"username" json:"username" binding:"required"`
	Phone    string `form:"phone" json:"phone" binding:"required"`
	Passwd   string `form:"passwd" json:"passwd" binding:"required"`
}

// TokenUserInfo 结构体中的数据将会编码进token
type TokenUserInfo struct {
	ID string
}

func GetAuthMiddleware(isAdmin bool) (*jwt.GinJWTMiddleware, error) {
	cfg := config.GetConfig()
	if isAdmin {
		return getAuthMiddleware([]byte(cfg.AdminJWT.Secret), cfg.AdminJWT.Timeout, cfg.AdminJWT.MaxRefresh, isAdmin)
	} else {
		return getAuthMiddleware([]byte(cfg.JWT.Secret), cfg.JWT.Timeout, cfg.JWT.MaxRefresh, isAdmin)
	}
}

func getAuthMiddleware(secret []byte, timeout, maxRefresh time.Duration, isAdmin bool) (*jwt.GinJWTMiddleware, error) {
	var err error

	authMiddleware, err = jwt.New(&jwt.GinJWTMiddleware{
		Realm:       appRealm,
		Key:         secret,
		Timeout:     timeout,
		MaxRefresh:  maxRefresh,
		IdentityKey: IdentityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			if v, ok := data.(*TokenUserInfo); ok {
				return jwt.MapClaims{
					IdentityKey: v.ID,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &TokenUserInfo{
				ID: claims[IdentityKey].(string),
			}
		},
		Authenticator: func(c *gin.Context) (interface{}, error) {
			var loginInfo loginForm
			if err := c.ShouldBind(&loginInfo); err != nil {
				return "", jwt.ErrMissingLoginValues
			}
			phone := loginInfo.Phone
			passwd := loginInfo.Passwd

			if phone == "" || passwd == "" {
				return nil, jwt.ErrFailedAuthentication
			}
			id, err := db.Login(phone, passwd, isAdmin)
			if err != nil {
				log.Println(err)
				return nil, jwt.ErrFailedAuthentication
			}
			u := &TokenUserInfo{
				ID: strconv.Itoa(id),
			}
			return u, nil
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"message": message,
			})
		},
		TokenLookup: "header: Authorization, query: token, cookie: jwt",

		TokenHeadName: "Bearer",

		TimeFunc: time.Now,
	})
	if err != nil {
		return nil, err
	}
	err = authMiddleware.MiddlewareInit()
	if err != nil {
		return nil, errors.New("authMiddleware.MiddlewareInit() Error:" + err.Error())
	}
	return authMiddleware, nil
}

func GenerateToken(id int) string {
	token, _, err := authMiddleware.TokenGenerator(
		&TokenUserInfo{
			ID: strconv.Itoa(id),
		},
	)
	if err != nil {
		return ""
	}
	return token
}

func IsValidToken(token string) bool {
	j, err := authMiddleware.ParseTokenString(token)
	if err != nil {
		return false
	}
	return j.Valid
}
