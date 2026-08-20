package gohan

import (
	"github.com/farkhanisturkia/gohan/internal/config"
	"github.com/farkhanisturkia/gohan/internal/database"
	gohanHttp "github.com/farkhanisturkia/gohan/internal/http"
	"github.com/farkhanisturkia/gohan/internal/security"
	"github.com/farkhanisturkia/gohan/utils"
	"github.com/farkhanisturkia/gohan/internal/mail"
)

// Config
type Env = config.Env

var InitEnv = config.InitEnv
var GetEnv = config.GetEnv

// Database
type DB = database.DB
type Tx = database.Tx
type RawQuery = database.RawQuery

var GetConn = database.GetConn

// HTTP Request & Response
var BindJSON = gohanHttp.BindJSON
var JSON = gohanHttp.JSON
var Error = gohanHttp.Error
var Param = gohanHttp.Param

// HTTP Client (External API Requests)
type HTTPClient = gohanHttp.HTTPClient

var NewHTTPClient = gohanHttp.NewHTTPClient
var FetchJSON = gohanHttp.FetchJSON
var PostJSON = gohanHttp.PostJSON

// HTTP Router & Server
type Router = gohanHttp.Router

var SetRoute = gohanHttp.SetRoute
var Get = gohanHttp.Get
var Post = gohanHttp.Post
var Put = gohanHttp.Put
var Patch = gohanHttp.Patch
var Delete = gohanHttp.Delete
var Serve = gohanHttp.Serve
var Use = gohanHttp.Use
var TimeoutMiddleware = gohanHttp.TimeoutMiddleware

// Security
type JWTClaims = security.JWTClaims

var HashPassword = security.HashPassword
var CheckPasswordHash = security.CheckPasswordHash
var GenerateRandomToken = security.GenerateRandomToken
var HashToken = security.HashToken
var GenerateJWT = security.GenerateJWT
var ValidateJWT = security.ValidateJWT

// Utils
var GetClientIP = utils.GetClientIP
var ParseTokenName = utils.ParseTokenName

// Mail Export
var SendEmail = mail.SendEmail
var InitMailWorker = mail.InitMailWorker
var QueueEmail = mail.QueueEmail