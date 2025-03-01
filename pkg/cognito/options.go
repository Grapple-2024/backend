package cognito

func WithUserPool(userPoolID string) func(*Client) {
	return func(c *Client) {
		c.userPoolID = userPoolID
	}
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
