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

var GetConn = database.GetConn

// HTTP Request & Response
var BindJSON = gohanHttp.BindJSON
var JSON = gohanHttp.JSON
var Error = gohanHttp.Error
var Param = gohanHttp.Param

// HTTP Router & Server
type Router = gohanHttp.Router

var SetRoute = gohanHttp.SetRoute
var Get = gohanHttp.Get
var Post = gohanHttp.Post
var Put = gohanHttp.Put
var Patch = gohanHttp.Patch
var Delete = gohanHttp.Delete
var Serve = gohanHttp.Serve

// Security
var HashPassword = security.HashPassword
var CheckPasswordHash = security.CheckPasswordHash
var GenerateRandomToken = security.GenerateRandomToken
var HashToken = security.HashToken

// Utils
var GetClientIP = utils.GetClientIP
var ParseTokenName = utils.ParseTokenName

// Mail Export
var SendEmail = mail.SendEmail