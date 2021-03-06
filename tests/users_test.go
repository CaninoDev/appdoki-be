package tests

import (
	"appdoki-be/app/repositories"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
)

func testUsers(t *testing.T) {
	t.Run("expect GET /users to return an empty list of users", emptyListUsers)

	seeds.SeedUsers()
	defer seeds.TruncateUsers()

	t.Run("expect GET /users to return a list of users", listUsers)

	t.Run("expect GET /users/{id} to return a user", getUser)
	t.Run("expect GET /users/{id} to return 404 when not found", getNonexistentUser)

	t.Run("expect GET /users/{id}/beers to return a user's beer summary", getUserBeers)
	t.Run("expect GET /users/{id}/beers to return 404 when not found", getNonexistentUserBeers)

	t.Run("expect POST /users/{id}/beers/{beers} to return 400 when beers is invalid", giveBadBeers)
	t.Run("expect POST /users/{id}/beers/{beers} to return 400 when beers is negative", giveNegativeBeers)
	t.Run("expect POST /users/{id}/beers/{beers} to return 403 when giving beers to self", giveYourselfBeers)
	t.Run("expect POST /users/{id}/beers/{beers} to return 404 when receiving user not found", giveNonexistentUserBeers)
	t.Run("expect POST /users/{id}/beers/{beers} to return 204 when successful", giveBeers)
}

func getUsers(t *testing.T) []repositories.User {
	res, err := http.Get(apiURL + "/users")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status code, got %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("expected to be able to read response body")
	}

	var response []repositories.User
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.Fatal("expected to be able to parse response body")
	}

	return response
}

func emptyListUsers(t *testing.T) {
	users := getUsers(t)

	if len(users) != 0 {
		t.Fatalf("expected to get 0 users, got %d", len(users))
	}
}

func listUsers(t *testing.T) {
	users := getUsers(t)

	if len(users) != 5 {
		t.Fatalf("expected to get 5 users, got %d", len(users))
	}
}

func getUser(t *testing.T) {
	res, err := http.Get(apiURL + "/users/2")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status code, got %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("expected to be able to read response body")
	}

	var user repositories.User
	err = json.Unmarshal(body, &user)
	if err != nil {
		t.Fatal("expected to be able to parse response body")
	}

	if user.ID != "2" {
		t.Fatalf("expected user with ID 2, got %s", user.ID)
	}
}

func getNonexistentUser(t *testing.T) {
	res, err := http.Get(apiURL + "/users/thisuserdoesnotexist")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 status code, got %d", res.StatusCode)
	}
}

func getUserBeers(t *testing.T) {
	res, err := http.Get(apiURL + "/users/2/beers")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status code, got %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("expected to be able to read response body")
	}

	var beers repositories.UserBeerLog
	err = json.Unmarshal(body, &beers)
	if err != nil {
		t.Fatal("expected to be able to parse response body")
	}

	if beers.Given != 0 || beers.Received != 0 {
		t.Fatalf("expected 0 given 0 received, got %d given %d received", beers.Given, beers.Received)
	}
}

func getNonexistentUserBeers(t *testing.T) {
	res, err := http.Get(apiURL + "/users/287f6gsdfsd76fgsdfgs76d5/beers")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 status code, got %d", res.StatusCode)
	}
}

func giveBadBeers(t *testing.T) {
	res, err := http.Post(apiURL + "/users/2/beers/fifty", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 status code, got %d", res.StatusCode)
	}
}

func giveNegativeBeers(t *testing.T) {
	res, err := http.Post(apiURL + "/users/2/beers/-12", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 status code, got %d", res.StatusCode)
	}
}

func giveYourselfBeers(t *testing.T) {
	res, err := http.Post(apiURL + "/users/1/beers/10", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 status code, got %d", res.StatusCode)
	}
}

func giveNonexistentUserBeers(t *testing.T) {
	res, err := http.Post(apiURL + "/users/287f6gsdfsd76fgsdfgs76d5/beers/10", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 status code, got %d", res.StatusCode)
	}
}

func giveBeers(t *testing.T) {
	res, err := http.Post(apiURL + "/users/2/beers/10", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 404 status code, got %d", res.StatusCode)
	}

	res, err = http.Get(apiURL + "/users/2/beers")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status code, got %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal("expected to be able to read response body")
	}

	var beers repositories.UserBeerLog
	err = json.Unmarshal(body, &beers)
	if err != nil {
		t.Fatal("expected to be able to parse response body")
	}

	if beers.Given != 0 || beers.Received != 10 {
		t.Fatalf("expected 0 given 10 received, got %d given %d received", beers.Given, beers.Received)
	}
}