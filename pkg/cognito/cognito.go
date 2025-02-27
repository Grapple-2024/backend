package cognito

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	b64 "encoding/base64"
	"errors"
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

func (c *Client) DeleteGroup(ctx context.Context, name string) (*cip.DeleteGroupOutput, error) {
	return c.Client.DeleteGroup(ctx, &cip.DeleteGroupInput{
		GroupName:  aws.String(name),
		UserPoolId: aws.String(c.userPoolID),
	}, nil)
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
			// Enhanced error logging
			var notFoundErr *types.ResourceNotFoundException
			if errors.As(err, &notFoundErr) {
				log.Error().Str("userPoolId", c.userPoolID).Msg("User pool does not exist")
			} else {
				log.Error().Err(err).Str("userPoolId", c.userPoolID).Msg("Failed to list groups")
			}
			return nil, err
		}

		for _, group := range page.Groups {
			// Log each group's details properly
			out.Groups = append(out.Groups, group)
		}
	}

	return out, nil
}
func (c *Client) ListGroupsForUser(ctx context.Context, cognitoSubID string) (*cip.AdminListGroupsForUserOutput, error) {
	resp, err := c.Client.AdminGetUser(ctx, &cip.AdminGetUserInput{
		Username:   &cognitoSubID,
		UserPoolId: &c.userPoolID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find cognito user with Username %s in pool %s", cognitoSubID, c.userPoolID)
	}

	listGroupsResp, err := c.Client.AdminListGroupsForUser(ctx, &cip.AdminListGroupsForUserInput{
		UserPoolId: aws.String(c.userPoolID),
		Username:   resp.Username,
	})
	if err != nil {
		return nil, err
	}

	return listGroupsResp, nil
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

func DeleteCognitoGroupsForGym(ctx context.Context, cc *Client, gymID string) error {
	// Generate the three group names
	groupTypes := []string{"owners", "coaches", "students"}

	for _, groupType := range groupTypes {
		groupName := fmt.Sprintf("gym::%s::%s", gymID, groupType)

		// Delete the Cognito Group
		_, err := cc.DeleteGroup(ctx, groupName)
		if err != nil {
			log.Warn().Msgf("Could not delete group %s: %v", groupName, err.Error())
			return err
		}
	}

	return nil
}
