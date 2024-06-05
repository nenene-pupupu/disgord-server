package controller

import (
	"crypto/ecdsa"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"time"

	"disgord/ent/user"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"golang.org/x/crypto/bcrypt"
)

// SignUp godoc
//
//	@Tags		auth
//	@Summary	sign up and create a new user
//	@Param		body	body		controller.SignUp.Body	true	"Request body"
//	@Success	201		{object}	ent.User
//	@Failure	409		"username already exists"
//	@Router		/auth/sign-up [post]
func (*Controller) SignUp(c *gin.Context) {
	type Body struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		DisplayName string `json:"displayName" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	user, err := client.User.
		Create().
		SetUsername(body.Username).
		SetPassword(hashPassword(body.Password)).
		SetDisplayName(body.DisplayName).
		SetProfileColorIndex(generateProfileColorIndex(body.Username, 4)).
		Save(ctx)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message": "username already exists",
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func generateProfileColorIndex(username string, size uint8) uint8 {
	h := fnv.New32a()
	h.Write([]byte(username))
	hash := h.Sum32()
	return uint8(hash)%size + 1
}

type Token struct {
	AccessToken string `json:"accessToken" binding:"required"`
}

// SignIn godoc
//
//	@Description	Set "Authorization" header with the "Bearer ${accessToken}" to authenticate requests.
//	@Tags			auth
//	@Summary		sign in and receive an access token
//	@Param			body	body		controller.SignIn.Body	true	"Request body"
//	@Success		200		{object}	controller.Token
//	@Failure		401		"invalid username or password"
//	@Failure		404		"user not found"
//	@Router			/auth/sign-in [post]
func (*Controller) SignIn(c *gin.Context) {
	type Body struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	var body Body
	if err := c.Bind(&body); err != nil {
		return
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	user, err := tx.User.
		Query().
		Where(user.Username(body.Username)).
		Only(ctx)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "user not found",
		})
		return
	}

	if !verifyPassword(user.Password, body.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "invalid username or password",
		})
		return
	}

	accessToken, refreshToken, err := issueToken(user.ID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = user.Update().
		SetRefreshToken(refreshToken).
		Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("refreshToken", refreshToken, 60*60*24*14, "/", "", true, true)

	c.JSON(http.StatusOK, Token{
		AccessToken: accessToken,
	})
}

// Refresh godoc
//
//	@Tags		auth
//	@Summary	refresh an access token
//	@Success	200	{object}	controller.Token
//	@Failure	401	"unauthorized"
//	@Failure	404	"cannot find user"
//	@Router		/auth/refresh [post]
func (*Controller) Refresh(c *gin.Context) {
	userID, err := extractUserID(c.Request, cookieExtractor)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	tx, err := client.Tx(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer tx.Rollback()

	user, err := tx.User.Get(ctx, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "cannot find user",
		})
		return
	}

	refreshToken, _ := c.Cookie("refreshToken")
	if user.RefreshToken != refreshToken {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "unauthorized",
		})
		return
	}

	accessToken, refreshToken, err := issueToken(userID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	_, err = user.Update().
		SetRefreshToken(refreshToken).
		Save(ctx)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.Status(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("refreshToken", refreshToken, 60*60*24*14, "/", "", true, true)

	c.JSON(http.StatusOK, Token{
		AccessToken: accessToken,
	})
}

// SignOut godoc
//
//	@Tags		auth
//	@Summary	sign out and revoke the refresh token
//	@Success	200
//	@Router		/auth/sign-out [post]
func (*Controller) SignOut(c *gin.Context) {
	userID, err := extractUserID(
		c.Request,
		&request.MultiExtractor{
			request.OAuth2Extractor,
			cookieExtractor,
		},
	)
	if err == nil {
		client.User.
			UpdateOneID(userID).
			ClearRefreshToken().
			Save(ctx)

		disconnect(userID)
	}

	c.SetSameSite(http.SameSiteNoneMode)
	c.SetCookie("refreshToken", "", -1, "/", "", true, true)

	c.Status(http.StatusOK)
}

var key *ecdsa.PrivateKey

func init() {
	b, err := os.ReadFile("disgord.pem")
	if err != nil {
		log.Fatal(err)
	}

	key, err = jwt.ParseECPrivateKeyFromPEM(b)
	if err != nil {
		log.Fatal(err)
	}
}

type Claims struct {
	UserID int `json:"userId"`
	jwt.RegisteredClaims
}

func (*Controller) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := extractUserID(c.Request, request.OAuth2Extractor)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		c.Set("userID", userID)

		c.Next()
	}
}

func getCurrentUserID(c *gin.Context) int {
	userID, _ := c.Get("userID")
	return userID.(int)
}

func issueToken(userID int) (string, string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute * 30))
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(key)
	if err != nil {
		return "", "", err
	}

	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 14))
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(key)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	return string(hash)
}

func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var cookieExtractor = &CookieExtractor{}

type CookieExtractor struct {
	request.Extractor
}

func (e *CookieExtractor) ExtractToken(req *http.Request) (string, error) {
	cookie, err := req.Cookie("refreshToken")
	if err != nil {
		return "", request.ErrNoTokenInRequest
	}

	return cookie.Value, nil
}

func extractUserID(req *http.Request, extractor request.Extractor) (int, error) {
	token, err := request.ParseFromRequest(
		req,
		extractor,
		func(*jwt.Token) (interface{}, error) { return key.Public(), nil },
		request.WithClaims(&Claims{}),
		request.WithParser(jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodES256.Name}),
			jwt.WithIssuedAt(),
		)),
	)
	if err != nil {
		return 0, err
	}

	return token.Claims.(*Claims).UserID, nil
}
