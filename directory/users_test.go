package directory

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

// User used to call User endpoint.
const user = "erick"

var (
	listJSON = `
    [
		{
			"coreId": "coreId1",
			"fullName": "fullName1",
            "status": "A",
            "id": "mmid1"
		},
		{
			"coreId": "string2",
			"fullName": "fullName2"
			"status": "status2"
			"id": "mmid2"
		} 
    ]
    `

	listEmptyJSON = `[]`

	userJSON = `	
		{
			"coreId" : "aeg095",
            "fullName": "Erick Guevara",
            "status": "A",
            "id": "erick"
		}
	`

	employeeDoesNotExist = `
	{
		"error": {
			"errors": [
  			{
  				"domain": "global",
				"reason": "badRequest",
				"message": "Employee does not exists."
			}
		],
		"code": 400,
		"message": "Employee does not exists."
		}
	}
	`
)

func TestUsers_Get(t *testing.T) {
	setup()
	defer teardown()

	url := fmt.Sprintf("/employee/%v", user)
	mux.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, userJSON)
	})

	fields := "coreId,fullName,id,status"
	opt := &UsersOptions{Fields: &fields}
	user, _, err := client.Users.Get(context.Background(), user, opt)
	if err != nil {
		t.Errorf("Get() Attributes.List returned error: %v", err)
	}

	expected := &User{CoreID: "aeg095", FullName: "Erick Guevara", Status: "A", ID: "erick"}
	if !reflect.DeepEqual(user, expected) {
		t.Errorf("Get() Tags.List returned %+v, expected %+v", user, expected)
	}

}

func TestUsers_Get_emptyUser(t *testing.T) {
	setup()
	defer teardown()

	opt := &UsersOptions{}
	_, _, err := client.Users.Get(context.Background(), "", opt)
	if err == nil {
		t.Errorf("Get() Expected param can not be empty %v", err)
	}
}

func TestUsers_Get_badBody(t *testing.T) {
	setup()
	defer teardown()

	opt := &UsersOptions{}
	_, resp, err := client.Users.Get(context.Background(), user, opt)

	// Check that response is error on nil request body
	if err == nil {
		t.Error("Get() Expected Request body error.")
	}

	// Check that response status code is http.StatusNotFound.
	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("Get() Expected Status code got %v, want %v", got, want)
	}
}

func TestUsers_Get_employeeDoesNotExist(t *testing.T) {

	setup()
	defer teardown()

	url := fmt.Sprintf("/employee/%v", "employee_does_not_exist")

	mux.HandleFunc(url, func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, employeeDoesNotExist)
	})

	opt := &UsersOptions{}
	_, resp, err := client.Users.Get(context.Background(), "employee_does_not_exist", opt)

	if err == nil {
		t.Errorf("Get() Expected http error on request, %v", err)
	}

	// Check that response status code is http.StatusNotFound.
	if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("Get() Expected Status code got %v, want %v", got, want)

	}

}
