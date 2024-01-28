package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Grapple-2024/backend/cognito"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/go-http-utils/headers"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	CognitoClient *cognito.Client
}

func (h *Handler) Login(c *gin.Context) {
	c.Request.ParseForm()
	form := c.Request.Form
	username := form.Get("username")
	password := form.Get("password")
	refresh := form.Get("refresh")
	refreshToken := form.Get("refreshToken")

	log.Info().Msgf("Parsed form data: %v", form)

	auth, err := h.CognitoClient.Login(username, password, refresh, refreshToken)
	if err != nil {
		c.JSON(http.StatusFound, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, map[string]string{
		"access_token":  *auth.AccessToken,
		"refresh_token": *auth.RefreshToken,
	})
}

func (h *Handler) Protected(c *gin.Context) {

	c.JSON(http.StatusFound, "Hello world")
}

func (h *Handler) Register(c *gin.Context) {

	c.Request.ParseForm()
	form := c.Request.Form
	email := form.Get("email")

	username := form.Get("username")
	password := form.Get("password")
	phoneNumber := form.Get("phone_number")
	givenName := form.Get("given_name")
	familyName := form.Get("family_name")
	picture := form.Get("picture")
	birthDate := form.Get("birthdate")

	log.Info().Msgf("Parsed form data: %v, user identity: %s %s %s %s", form, givenName, familyName, picture, birthDate)

	if err := h.CognitoClient.SignUp(email, username, password, phoneNumber, birthDate, picture, givenName, familyName); err != nil {
		c.JSON(http.StatusFound, map[string]string{"error": err.Error()})
		return
	}

	c.JSON(http.StatusFound, fmt.Sprintf("/otp?username=%s", username))
}

// validateJWT takes a token string and validates it
func (h *Handler) ValidateJWT(c *gin.Context) {
	authHeader := c.Request.Header.Get(headers.Authorization)
	if len(authHeader) <= 1 {
		c.JSON(http.StatusFound, map[string]string{"error": "invalid or corrupt Authorization header"})
		c.Abort()
		return
	}
	tokenString := strings.TrimSpace(strings.Split(authHeader, "Bearer")[1])

	regionID := "us-west-1"
	userPoolID := "us-west-1_OzzDUVG4n"
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusFound, map[string]string{"error": err.Error()})
		return
	}

	// Parse the JWT.
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusFound, map[string]string{"error": err.Error()})
		return
	}

	// Check if the token is valid.
	if !token.Valid {
		c.AbortWithStatusJSON(http.StatusFound, map[string]string{"error": err.Error()})
		return
	}

	log.Info().Msgf("Token is valid!\n%+v\n", token)

}
