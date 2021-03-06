package jwtmiddleware

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/jmoiron/sqlx"
	_"github.com/lib/pq"

	"github.com/aaronaaeng/chat.connor.fun/model"
	"github.com/labstack/echo"
	"net/http/httptest"
	"strings"
	"github.com/aaronaaeng/chat.connor.fun/controllers/auth"
	"github.com/aaronaaeng/chat.connor.fun/context"
	"github.com/aaronaaeng/chat.connor.fun/db"
	"github.com/aaronaaeng/chat.connor.fun/db/roles"
	"github.com/aaronaaeng/chat.connor.fun/db/users"
	"github.com/satori/go.uuid"
)
const (
	testDbHost = "localhost"
	testDbPort = 5432
	testDbName = "connorfuntest_jwtmiddleware"
)

var testDb *sqlx.DB

func TestMain(m *testing.M) {
	//New the connorfuntest db and reconnect to the new db
	db, err := sqlx.Open("postgres", "postgresql://localhost:5432?sslmode=disable")
	if err != nil {
		panic("failed to establish db connection")
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", testDbName))
	if err != nil {
		panic(err) //Something went horribly wrong
	}

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDbName))
	if err != nil {
		panic("Failed to create test db: " + err.Error())
	}

	testDb, err = sqlx.Open("postgres",
		fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable", testDbHost, testDbPort, testDbName))
	if err != nil {
		panic("failed to establish db connection")
	}

	m.Run()

	//clean up the connorfuntestdb
	testDb.Close()
	cleanUpTestDb(db)

	db.Close()
}

func cleanUpTestDb(db *sqlx.DB) {
	_, err:= db.Query(fmt.Sprintf(`SELECT
			pg_terminate_backend (pg_stat_activity.pid)
			FROM
			pg_stat_activity
			WHERE
			pg_stat_activity.datname = '%s';`, testDbName))

	if err != nil {
		panic("FAILED TO KILL BG CONNECTIONS")
	}

	_, err = db.Exec(fmt.Sprintf("DROP DATABASE %s;", testDbName))
	if err != nil {
		panic("FAILED TO DROP TEST DB: " + err.Error())
	}
}

func initTables() (db.UserRepository, db.RolesRepository){
	usersRepo, _ := users.New(testDb) //these must be inited in the right order
	rolesRepo, _ := roles.New(testDb)
	return usersRepo, rolesRepo
}

func cleanUpTables(t *testing.T) {
	_, err := testDb.Exec("DROP TABLE user_roles")
	assert.NoError(t, err)
	_, err = testDb.Exec("DROP TABLE users")
}


const (
	testJsonRoleData = `
		{
		  "anonUser": {
			"parent": "NONE",
			"override": "NONE",
			"permissions": [
			  {"path": "/foo/bar",  "verbs": "c"}
			]
		  },

		  "normalUser": {
			"parent": "anon_user",
			"override": "NONE",
			"permissions": [
			  {"path": "/foo/bar",  "verbs": "crud"},
			  {"path": "/foo/foo",  "verbs": "crud"}
			]
		  }
		}
	`
)



func TestDoAuthorization_WithAuth_Fail(t *testing.T) {
	usersRepo, rolesRepo := initTables()
	assert.NoError(t, model.InitRoleMap([]byte(testJsonRoleData)))
	user := &model.User{Id: uuid.NewV4(), Username: "test", Secret: "test"}
 	err := usersRepo.Add(user)
 	assert.NoError(t, err)

 	assert.NoError(t, rolesRepo.Add(user.Id, model.RoleNormal))

 	e := echo.New()
	req := httptest.NewRequest("POST", "/bar", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := &context.AuthorizedContextImpl{
		Context: e.NewContext(req, rec),
	}

	failHandler := func(c echo.Context) error {
		assert.Fail(t, "Handler was called")
		return nil
	}

	claims := auth.Claims{
		User: *user,
		Permissions: []model.Permission{},
	}

	doAuthorization(failHandler, &claims, c, rolesRepo)

	cleanUpTables(t)
}

func TestDoAuthorization_WithAuth(t *testing.T) {
	usersRepo, rolesRepo := initTables()
	assert.NoError(t, model.InitRoleMap([]byte(testJsonRoleData)))
	user := &model.User{Id: uuid.NewV4(), Username: "test", Secret: "test"}
	err := usersRepo.Add(user)
	assert.NoError(t, err)

	assert.NoError(t, rolesRepo.Add(user.Id, model.RoleNormal))

	e := echo.New()
	req := httptest.NewRequest("POST", "/foo/foo", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := &context.AuthorizedContextImpl{
		Context: e.NewContext(req, rec),
	}

	shouldBeTrue := false
	failHandler := func(c echo.Context) error {
		ac := c.(context.AuthorizedContext)
		assert.True(t, ac.AccessCode().CanCreate())
		assert.NotNil(t, ac.Requestor())
		shouldBeTrue = true
		return nil
	}

	claims := auth.Claims{
		User: *user,
		Permissions: []model.Permission{},
	}

	doAuthorization(failHandler, &claims, c, rolesRepo)

	assert.True(t, shouldBeTrue, "handler was not called")

	cleanUpTables(t)
}

func TestDoAuthorization_NoAuth(t *testing.T) {
	_, rolesRepo := initTables()
	assert.NoError(t, model.InitRoleMap([]byte(testJsonRoleData)))

	e := echo.New()
	req := httptest.NewRequest("POST", "/foo/bar", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := &context.AuthorizedContextImpl{
		Context: e.NewContext(req, rec),
	}

	shouldBeTrue := false
	failHandler := func(c echo.Context) error {
		ac := c.(context.AuthorizedContext)
		assert.True(t, ac.AccessCode().CanCreate())
		assert.Empty(t, ac.Requestor())
		shouldBeTrue = true
		return nil
	}

	doAuthorization(failHandler, nil, c, rolesRepo)

	assert.True(t, shouldBeTrue, "handler was not called")

	cleanUpTables(t)
}

func TestDoAuthorization_NoAuth_Fail(t *testing.T) {
	_, rolesRepo := initTables()
	assert.NoError(t, model.InitRoleMap([]byte(testJsonRoleData)))

	e := echo.New()
	req := httptest.NewRequest("POST", "/foo/foo", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := &context.AuthorizedContextImpl{
		Context: e.NewContext(req, rec),
	}

	failHandler := func(c echo.Context) error {
		assert.Fail(t, "Handler was called")
		return nil
	}

	doAuthorization(failHandler, nil, c, rolesRepo)

	cleanUpTables(t)
}