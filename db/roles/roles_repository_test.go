package roles

import (
	"testing"
	"github.com/jmoiron/sqlx"
	_"github.com/lib/pq"
	"github.com/aaronaaeng/chat.connor.fun/db/users"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/aaronaaeng/chat.connor.fun/model"
	_"github.com/lib/pq"
)

const (
	testDbHost = "localhost"
	testDbPort = 5432
	testDbName = "connorfuntest"
)

var testDb *sqlx.DB

func TestMain(m *testing.M) {
	//Init the connorfuntest db and reconnect to the new db
	db, err := sqlx.Open("postgres", "postgresql://localhost:5432?sslmode=disable")
	if err != nil {
		panic("failed to establish db connection")
	}

	_, err = db.Exec("DROP DATABASE IF EXISTS connorfuntest")
	if err != nil {
		panic(err) //Something went horribly wrong
	}

	_, err = db.Exec("CREATE DATABASE connorfuntest;")
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
	_, err:= db.Query(`SELECT
			pg_terminate_backend (pg_stat_activity.pid)
			FROM
			pg_stat_activity
			WHERE
			pg_stat_activity.datname = 'connorfuntest'`)

	if err != nil {
		panic("FAILED TO KILL BG CONNECTIONS")
	}

	_, err = db.Exec("DROP DATABASE connorfuntest;")
	if err != nil {
		panic("FAILED TO DROP TEST DB: " + err.Error())
	}
}

func testRowCountEquals(t *testing.T, expected int) {
	rows, err := testDb.Queryx("SELECT COUNT(*) FROM user_roles")
	assert.NoError(t, err)

	assert.True(t, rows.Next())
	var count int
	err = rows.Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, expected, count)
}

func initTables() {
	_, _ = users.Init(testDb) //these must be inited in the right order
	_, _ = Init(testDb)
}

func cleanUpTables(t *testing.T) {
	_, err := testDb.Exec("DROP TABLE user_roles")
	assert.NoError(t, err)
	_, err = testDb.Exec("DROP TABLE users")
}

func TestInit(t *testing.T) {
	_, _ = users.Init(testDb)
	_, err := Init(testDb)

	assert.NoError(t, err)
	assert.NotEmpty(t, Repo)

	_, err = testDb.Exec("DROP TABLE user_roles")
	assert.NoError(t, err)
	_, _ = testDb.Exec("DROP TABLE users")
}

func TestRepository_AddRole(t *testing.T) {
	initTables()
	err := Repo.AddRole(123, "foobarrole") //not valid uid
	assert.Error(t, err)

	testRowCountEquals(t, 0)

	user, err := users.Repo.Create(model.User{Username: "test", Secret: "test"})

	validId := user.Id

	err = Repo.AddRole(validId, "foobarrole")
	assert.NoError(t, err)

	testRowCountEquals(t, 1)

	cleanUpTables(t)
}

func TestRepository_GetUserRoles(t *testing.T) {
	initTables()

	user1 := model.User{Username: "user1", Secret: "test"}
	user2 := model.User{Username: "user2", Secret: "test"}
	user3 := model.User{Username: "user3", Secret: "test"}

	validUser1, _ := users.Repo.Create(user1)
	validUser2, _ := users.Repo.Create(user2)
	validUser3, _ := users.Repo.Create(user3)

	Repo.AddRole(validUser1.Id, "role1")
	Repo.AddRole(validUser1.Id, "role2")
	Repo.AddRole(validUser1.Id, "role3")

	Repo.AddRole(validUser2.Id, "role4")
	Repo.AddRole(validUser2.Id, "role5")

	user1Roles, err := Repo.GetUserRoles(validUser1.Id)
	assert.NoError(t, err)
	user2Roles, err := Repo.GetUserRoles(validUser2.Id)
	assert.NoError(t, err)
	user3Roles, err := Repo.GetUserRoles(validUser3.Id)
	assert.NoError(t, err)

	assert.Len(t, user1Roles, 3)
	assert.Len(t, user2Roles, 2)
	assert.Len(t, user3Roles, 0)

	//This test could be improved in the future

	cleanUpTables(t)
}