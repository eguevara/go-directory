package directory

import (
	"context"
	"fmt"
)

// UsersService is an interface for interfacing with the UsersService
// endpoints of the directory APi.
//
// See: https://mm-directory.appspot.com/_ah/api/mm/v1/employee/erick
type UsersService interface {
	Get(context.Context, string, *UsersOptions) (*User, *Response, error)
}

// UsersServiceOp handles communication with the Users related
// methods of the directory API.
type UsersServiceOp struct {
	client *Client
}

var _ UsersService = &UsersServiceOp{}

// User represents a directory User resource.
type User struct {
	CoreID   string `json:"coreId"`
	FullName string `json:"fullName"`
	Status   string `json:"status"`
	ID       string `json:"id"`
}

// UsersOptions specifies the optional parameters to the UserService.Get()
type UsersOptions struct {
	Fields *string `url:"fields,omitempty"`
}

// Get will call User service with mmID param.
func (u *UsersServiceOp) Get(ctx context.Context, mmID string, opt *UsersOptions) (*User, *Response, error) {
	if mmID == "" {
		return nil, nil, fmt.Errorf("mmID can not be empty")
	}

	url := fmt.Sprintf("employee/%v", mmID)
	url, err := addOptions(url, opt)

	req, err := u.client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(User)
	resp, err := u.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, err
}
