package main

import (
	"github.com/labstack/echo"
	"html/template"
	"io"
	"github.com/jmoiron/sqlx"
	"github.com/aaronaaeng/chat.connor.fun/config"
	"github.com/aaronaaeng/chat.connor.fun/db/users"
	_"github.com/lib/pq"
	"github.com/labstack/echo/middleware"
	"github.com/aaronaaeng/chat.connor.fun/controllers"
	"github.com/aaronaaeng/chat.connor.fun/model"
	"fmt"
	"github.com/aaronaaeng/chat.connor.fun/db/roles"
	"io/ioutil"
	"github.com/aaronaaeng/chat.connor.fun/controllers/jwtmiddleware"
	"strings"
	"github.com/aaronaaeng/chat.connor.fun/controllers/chat"
	"github.com/aaronaaeng/chat.connor.fun/context"
	"github.com/aaronaaeng/chat.connor.fun/db"
	"github.com/aaronaaeng/chat.connor.fun/db/rooms"
	"github.com/aaronaaeng/chat.connor.fun/db/messages"
	"github.com/aaronaaeng/chat.connor.fun/db/verifications"
)


func createApiRoutes(api *echo.Group, hubMap *chat.HubMap, repo db.TransactionalRepository) {

	api.POST("/users", controllers.CreateUser(repo, config.Debug)).Name = "create-user"
	api.GET("/users/:id", controllers.GetUser(repo)).Name = "get-user"
	api.PUT("/users/:id", controllers.UpdateUser(repo))

	api.POST("/login", controllers.LoginUser(repo)).Name = "login-user"

	api.GET("/messages", controllers.GetMessages(repo)).Name = "get-messages"
	api.GET("/messages/:id", controllers.GetMessage(repo)).Name = "get-message"
	api.PUT("/messages/:id", controllers.UpdateMessage(repo)).Name = "update-message"

	api.GET("/rooms/nearby", controllers.GetNearbyRooms(repo)).Name = "get-nearby-rooms"
	api.GET("/rooms/:room/users", controllers.GetRoomMembers(hubMap)).Name = "get-room-members"
	api.GET("/rooms/:room", controllers.GetRoom(repo, hubMap)).Name = "get-room"
	api.POST("/rooms", controllers.CreateRoom(repo))

	api.PUT("/verifications/accountverification", controllers.VerifyUserAccount(repo))

	api.GET("/rooms/:room/ws", chat.HandleWebsocket(hubMap, repo)).Name = "join-room"
}

func addMiddlewares(e *echo.Echo, api *echo.Group, rolesRepository db.RolesRepository) {
	if !config.Debug {
		e.Pre(middleware.HTTPSNonWWWRedirect())
	}
	//this must be added first
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &context.AuthorizedContextImpl{Context: c}
			return h(cc)
		}
	})

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	api.Use(jwtmiddleware.JwtAuth(func(c echo.Context) bool {
		return strings.HasSuffix(c.Path(), "ws") //||
	}, jwtmiddleware.JWTBearerTokenExtractor, rolesRepository))

	api.Use(jwtmiddleware.JwtAuth(func(c echo.Context) bool { //websocket auth
		return !strings.HasSuffix(c.Path(), "ws")
	}, jwtmiddleware.JWTProtocolHeaderExtractor, rolesRepository))
}

func initDatabaseRepositories() (*sqlx.DB, db.UserRepository, db.RolesRepository,
		db.RoomsRepository, db.MessagesRepository, db.VerificationCodeRepository){
	database, err := sqlx.Open("postgres", config.DatabaseURL)
	if err != nil {
		panic(err)
	}
	userRepository, err := users.New(database)
	if err != nil {
		panic(err)
	}

	rolesRepository, err := roles.New(database)
	if err != nil {
		panic(err)
	}

	roomsRepository, err := rooms.New(database)
	if err != nil {
		panic(err)
	}

	messagesRepository, err := messages.New(database)
	if err != nil {
		panic(err)
	}

	verificationsRepo, err := verifications.New(database)
	if err != nil {
		panic(err)
	}

	return database, userRepository, rolesRepository, roomsRepository, messagesRepository, verificationsRepo
}

func main() {
	e := echo.New()
	e.Debug = config.Debug

	e.Static("/web", "frontend/build")
	e.Static("/static", "frontend/build/static")
	e.Static("/service-worker.js", "frontend/build/service-worker.js")
	e.GET("/", controllers.Index)
	e.GET("/*", controllers.Index)

	v1ApiGroup := e.Group("/api/v1")

	roleJsonData, err := ioutil.ReadFile("assets/roles.json")
	if err != nil {
		e.Logger.Fatal(err)
	}
	if model.InitRoleMap(roleJsonData) != nil {
		e.Logger.Fatal(fmt.Errorf("error creating roles data"))
	}

	hubMap := chat.NewHubMap()

	transactionalRepo := db.NewTransactionalRepository(initDatabaseRepositories())


	addMiddlewares(e, v1ApiGroup, transactionalRepo.Roles())
	createApiRoutes(v1ApiGroup, hubMap, transactionalRepo)

	t := &Template{
		templates: template.Must(template.ParseGlob("frontend/build/*.html")),
	}
	e.Renderer = t

	//log.SetOutput(os.Stdout)
	e.Logger.Fatal(e.Start(":4000"))
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}