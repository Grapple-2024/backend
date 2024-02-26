package main

import (
	"fmt"

	"github.com/Grapple-2024/backend/cognito"
	"github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

type Handler struct {
	CognitoClient *cognito.Client
}

func main() {
	cc, err := cognito.NewClient("us-west-1",
		cognito.WithClientID("40s9oop5e9srair8mljupn000j"),
		cognito.WithClientSecret("1fifmgpshit01l5eqppj95o1kjt2v16n32kaunve5ntv2n938ei9"),
	)
	if err != nil {
		panic(err)
	}

	h := Handler{CognitoClient: cc}
	// if err := h.Register(); err != nil {
	// 	panic(err)
	// }

	if err := h.Login("jordan", "<your-password>"); err != nil {
		panic(err)
	}
}
func (h *Handler) Login(username, password string) error {
	auth, err := h.CognitoClient.Login(username, password, "", "")
	if err != nil {
		return err
	}
	log.Info().Msgf("Auth: %+v, err: %v", auth, err)

	log.Info().Msgf("Token: %v", *auth.IdToken)
	return nil
}

func (h *Handler) Register() error {
	email := "jordan@dionysustechnologygroup.com"
	username := "jordan"
	password := "<your-password>"
	phoneNumber := "+19498705588"
	birthDate := "10/28/1997"
	picture := "123123"
	givenName := "Jordan"
	familyName := "Levin"
	if err := h.CognitoClient.SignUp(email, username, password, phoneNumber, birthDate, picture, givenName, familyName); err != nil {
		return err
	}
	return nil
}

// validateJWT takes a token string and validates it
func (h *Handler) ValidateJWT(tokenString string) error {
	// authHeader := c.Request.Header.Get(headers.Authorization)
	// if len(authHeader) <= 1 {
	// 	c.JSON(http.StatusFound, map[string]string{"error": "invalid or corrupt Authorization header"})
	// 	c.Abort()
	// 	return
	// }
	// tokenString := strings.TrimSpace(strings.Split(authHeader, "Bearer")[1])

	regionID := "us-west-1"
	userPoolID := "us-west-1_HT5oR6AwO"
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return err
	}

	// Parse the JWT.
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return fmt.Errorf("error parsing jwt: %v", err)
	}

	// Check if the token is valid.
	if !token.Valid {
		return fmt.Errorf("token is not valid")
	}

	return nil
}
