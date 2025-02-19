package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
)

const (
	flowRefreshToken     = "REFRESH_TOKEN_AUTH"
	flowUsernamePassword = "USER_PASSWORD_AUTH"
)

type Client struct {
	*cip.Client

	clientID     string
	clientSecret string
	userPoolID   string
}

// Token represents the AWS Cognito user token
type Token struct {
	Username   string   `mapstructure:"cognito:username"`
	Email      string   `mapstructure:"email"`
	Roles      []string `mapstructure:"cognito:roles"`
	Groups     []string `mapstructure:"cognito:groups"`
	GivenName  string   `mapstructure:"given_name"`
	FamilyName string   `mapstructure:"family_name"`

	Sub string `mapstructure:"sub"`
}

func NewClient(region string, opts ...func(*Client)) (*Client, error) {
	c := &Client{}

	// apply options to client using functional options pattern
	for _, o := range opts {
		o(c)
	}

	// Create Cognito Client
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	cognitoClient := cip.NewFromConfig(cfg)
	c.Client = cognitoClient
	return c, nil
}

// CreateGroup creates a new group in Cognito.
func (c *Client) CreateGroup(ctx context.Context, groupName string) error {
	_, err := c.Client.CreateGroup(ctx, &cip.CreateGroupInput{
		GroupName:  aws.String(groupName),
		UserPoolId: aws.String(c.userPoolID),
	})
	return err
}

// AddUserToGroup adds a user to a group in Cognito.
func (c *Client) AddUserToGroup(ctx context.Context, username, groupName string) error {
	_, err := c.Client.AdminAddUserToGroup(ctx, &cip.AdminAddUserToGroupInput{
		GroupName:  aws.String(groupName),
		UserPoolId: aws.String(c.userPoolID),
		Username:   aws.String(username),
	})
	return err
}

// RemoveUserFromGroup removes a user to a group in Cognito.
func (c *Client) RemoveUserFromGroup(ctx context.Context, username, groupName string) error {
	_, err := c.Client.AdminRemoveUserFromGroup(ctx, &cip.AdminRemoveUserFromGroupInput{
		GroupName:  aws.String(groupName),
		UserPoolId: aws.String(c.userPoolID),
		Username:   aws.String(username),
	})

	return err
}

func (c *Client) ListGroups(ctx context.Context) (*cip.ListGroupsOutput, error) {
	input := &cip.ListGroupsInput{
		UserPoolId: aws.String(c.userPoolID),
	}
	out := &cip.ListGroupsOutput{}

	paginator := cip.NewListGroupsPaginator(c.Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, group := range page.Groups {
			out.Groups = append(out.Groups, group)
		}
	}

	return out, nil
}

func (c *Client) ListGroupsForUser(ctx context.Context, cognitoSubID string) (*cip.AdminListGroupsForUserOutput, error) {
	log.Info().Msgf("Listing groups for user: %v, cognito pool id: %v", cognitoSubID, c.userPoolID)

	resp, err := c.Client.AdminGetUser(ctx, &cip.AdminGetUserInput{
		Username:   &cognitoSubID,
		UserPoolId: &c.userPoolID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find cognito user with Username %s in pool %s", cognitoSubID, c.userPoolID)
	}

	log.Info().Msgf("AdminGetUser output: %+v\n\n", resp)
	return c.Client.AdminListGroupsForUser(ctx, &cip.AdminListGroupsForUserInput{
		UserPoolId: aws.String(c.userPoolID),
		Username:   resp.Username,
	})
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

func (c *Client) Login(ctx context.Context, username, password, refresh, refreshToken string) (*types.AuthenticationResultType, error) {
	params := map[string]string{
		"USERNAME":    username,
		"PASSWORD":    password,
		"SECRET_HASH": c.newSecretHash(username), // Assuming newSecretHash is defined
	}

	var authFlow types.AuthFlowType
	if refresh != "" {
		authFlow = types.AuthFlowTypeRefreshTokenAuth
		params = map[string]string{
			"REFRESH_TOKEN": refreshToken,
		}
	} else {
		authFlow = types.AuthFlowTypeUserPasswordAuth
	}

	authTry := &cip.InitiateAuthInput{
		AuthFlow:       authFlow,
		AuthParameters: params,
		ClientId:       aws.String(c.clientID),
	}

	res, err := c.Client.InitiateAuth(ctx, authTry)
	if err != nil {
		return nil, err
	}

	if res.AuthenticationResult != nil {
		return res.AuthenticationResult, nil
	}

	return nil, fmt.Errorf("error authenticating, response: %v", res)
}
