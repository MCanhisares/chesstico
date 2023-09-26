package main

type Client struct {
	ID     string
	Secret string
	Domain string
	Public bool
	UserID string
}

func (c Client) GetDomain() string {
	return c.Domain
}

func (c Client) GetSecret() string {
	return c.Secret
}

func (c Client) GetID() string {
	return c.ID
}

func (c Client) IsPublic() bool {
	return c.Public
}

func (c Client) GetUserID() string {
	return c.UserID
}
