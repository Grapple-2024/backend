package cognito

import (
	"crypto/hmac"
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cip "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/rs/zerolog/log"
)

const (
	flowRefreshToken     = "REFRESH_TOKEN_AUTH"
	flowUsernamePassword = "USER_PASSWORD_AUTH"
)

type Client struct {
	*cip.CognitoIdentityProvider
	clientID     string
	clientSecret string
}

func WithClientID(id string) func(*Client) {
	return func(c *Client) {
		c.clientID = id
	}
}
func WithClientSecret(s string) func(*Client) {
	return func(c *Client) {
		c.clientSecret = s
	}
}

func NewClient(region string, opts ...func(*Client)) (*Client, error) {
	client := &Client{}

	// apply options to client using functional options pattern
	for _, o := range opts {
		o(client)
	}

	// Create Cognito Client
	conf := &aws.Config{Region: aws.String(region)}
	sess, err := session.NewSession(conf)
	if err != nil {
		return nil, err
	}

	c := cip.New(sess)
	client.CognitoIdentityProvider = c
	return client, nil
}

func (c *Client) SignUp(email, username, password, phoneNumber, birthDate, picture, givenName, familyName string) error {
	secretHash := c.newSecretHash(username)

	user := &cip.SignUpInput{
		Username:   aws.String(username),
		Password:   aws.String(password),
		ClientId:   aws.String(c.clientID),
		SecretHash: &secretHash,
		UserAttributes: []*cip.AttributeType{
			{
				Name:  aws.String("phone_number"),
				Value: aws.String(phoneNumber),
			},
			{
				Name:  aws.String("given_name"),
				Value: aws.String(givenName),
			},
			{
				Name:  aws.String("family_name"),
				Value: aws.String(familyName),
			},
			{
				Name:  aws.String("birthdate"),
				Value: aws.String(birthDate),
			},
			{
				Name:  aws.String("email"),
				Value: aws.String(email),
			},
			{
				Name:  aws.String("picture"),
				Value: aws.String(picture),
			},
		},
	}

	resp, err := c.CognitoIdentityProvider.SignUp(user)
	if err != nil {
		fmt.Println(err)
		// c.Redirect(http.StatusSeeOther, fmt.Sprintf("/register?message=%s", err.Error()))
		return err
	}
	log.Info().Msgf("Response: %+v", resp)
	return nil
}

func (c *Client) Login(username, password, refresh, refreshToken string) (*cip.AuthenticationResultType, error) {
	params := map[string]*string{
		"USERNAME":    aws.String(username),
		"PASSWORD":    aws.String(password),
		"SECRET_HASH": aws.String(c.newSecretHash(username)),
	}

	// Compute secret hash based on client secret.
	flow := aws.String(flowUsernamePassword)
	if refresh != "" {
		flow = aws.String(flowRefreshToken)
		params = map[string]*string{
			"REFRESH_TOKEN": aws.String(refreshToken),
		}
	}

	authTry := &cip.InitiateAuthInput{
		AuthFlow:       flow,
		AuthParameters: params,
		ClientId:       aws.String(c.clientID),
	}

	res, err := c.CognitoIdentityProvider.InitiateAuth(authTry)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("InitiateAuth: %+v", res)

	if res.AuthenticationResult != nil {
		return res.AuthenticationResult, nil
	}

	return nil, fmt.Errorf("error authenticating, response: %v", res)
}

func (c *Client) newSecretHash(username string) string {
	// create a new HMAC by defining the hash type and the key
	data := []byte(fmt.Sprintf("%s%s", username, c.clientID))
	hmac := hmac.New(sha256.New, []byte(c.clientSecret))

	// compute the HMAC
	hmac.Write([]byte(data))
	dataHmac := hmac.Sum(nil)

	return b64.StdEncoding.EncodeToString(dataHmac)
}
