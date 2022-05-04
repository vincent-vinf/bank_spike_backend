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

func GetAuthMiddleware() (*jwt.GinJWTMiddleware, error) {
	cfg := config.GetConfig().JWT
	var err error
	// the jwt middleware
	authMiddleware, err = jwt.New(&jwt.GinJWTMiddleware{
		Realm:       appRealm,
		Key:         []byte(cfg.Secret),
		Timeout:     cfg.Timeout,
		MaxRefresh:  cfg.MaxRefresh,
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
			id, err := db.Login(phone, passwd)
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
	token, _, err := authMiddleware.TokenGenerator(jwt.MapClaims{
		IdentityKey: strconv.Itoa(id),
	})
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
