package gohan

import (
	"github.com/farkhanisturkia/gohan/pkg/connect"
	"github.com/farkhanisturkia/gohan/pkg/internal/cache"
	"github.com/farkhanisturkia/gohan/pkg/internal/config"
	"github.com/farkhanisturkia/gohan/pkg/internal/database"
	gohanHttp "github.com/farkhanisturkia/gohan/pkg/internal/http"
	"github.com/farkhanisturkia/gohan/pkg/internal/mail"
	"github.com/farkhanisturkia/gohan/pkg/internal/security"
	"github.com/farkhanisturkia/gohan/pkg/protobuf/proto"
	"github.com/farkhanisturkia/gohan/pkg/utils"

	"github.com/redis/go-redis/v9"
)

// Framework Meta
const Version = "v1.11.5"

// Config
type Env = config.Env

var InitEnv = config.InitEnv
var GetEnv = config.GetEnv

// Database
type DB = database.DB
type Tx = database.Tx
type RawQuery = database.RawQuery
type Pagination = database.Pagination

var GetConn = database.GetConn

var ErrRecordNotFound = database.ErrRecordNotFound

var CachedData = cache.CachedData
var CacheForget = cache.CacheForget
var IsNotFound = cache.IsNotFound

// Redis
var Redis *redis.Client

var InitRedis = database.BindRedis(&Redis)

// HTTP Request & Response
var BindJSON = gohanHttp.BindJSON
var JSON = gohanHttp.JSON
var Marshal = gohanHttp.Marshal
var Unmarshal = gohanHttp.Unmarshal
var Error = gohanHttp.Error
var Param = gohanHttp.Param

// HTTP Client (External API Requests)
type HTTPClient = gohanHttp.HTTPClient

var NewHTTPClient = gohanHttp.NewHTTPClient
var FetchJSON = gohanHttp.FetchJSON
var PostJSON = gohanHttp.PostJSON

// HTTP Router & Server
type Router = gohanHttp.Router

var CORSMiddleware = gohanHttp.CORSMiddleware
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

// gRpc
type (
	AnyRequest           = connect.AnyRequest
	AnyResponse          = connect.AnyResponse
	ClientOption         = connect.ClientOption
	Code                 = connect.Code
	ConnectError         = connect.Error
	ConnectHTTPClient    = connect.HTTPClient
	Handler              = connect.Handler
	HandlerOption        = connect.HandlerOption
	IdempotencyLevel     = connect.IdempotencyLevel
	Interceptor          = connect.Interceptor
	Option               = connect.Option
	UnaryFunc            = connect.UnaryFunc
	UnaryInterceptorFunc = connect.UnaryInterceptorFunc

	Request[T any]                      = connect.Request[T]
	Response[T any]                     = connect.Response[T]
	Client[Req, Res any]                = connect.Client[Req, Res]
	ServerStream[Res any]               = connect.ServerStream[Res]
	ClientStream[Req any]               = connect.ClientStream[Req]
	BidiStream[Req, Res any]            = connect.BidiStream[Req, Res]
	ServerStreamForClient[Res any]      = connect.ServerStreamForClient[Res]
	ClientStreamForClient[Req, Res any] = connect.ClientStreamForClient[Req, Res]
	BidiStreamForClient[Req, Res any]   = connect.BidiStreamForClient[Req, Res]
)

const (
	IsAtLeastVersion1_13_0 = connect.IsAtLeastVersion1_13_0

	CodeCanceled           = connect.CodeCanceled
	CodeUnknown            = connect.CodeUnknown
	CodeInvalidArgument    = connect.CodeInvalidArgument
	CodeDeadlineExceeded   = connect.CodeDeadlineExceeded
	CodeNotFound           = connect.CodeNotFound
	CodeAlreadyExists      = connect.CodeAlreadyExists
	CodePermissionDenied   = connect.CodePermissionDenied
	CodeResourceExhausted  = connect.CodeResourceExhausted
	CodeFailedPrecondition = connect.CodeFailedPrecondition
	CodeAborted            = connect.CodeAborted
	CodeOutOfRange         = connect.CodeOutOfRange
	CodeUnimplemented      = connect.CodeUnimplemented
	CodeInternal           = connect.CodeInternal
	CodeUnavailable        = connect.CodeUnavailable
	CodeDataLoss           = connect.CodeDataLoss
	CodeUnauthenticated    = connect.CodeUnauthenticated
)

const (
	IdempotencyUnknown       = connect.IdempotencyUnknown
	IdempotencyNoSideEffects = connect.IdempotencyNoSideEffects
	IdempotencyIdempotent    = connect.IdempotencyIdempotent
)

var NewError = connect.NewError

func NewRequest[T any](message *T) *Request[T] {
	return connect.NewRequest(message)
}

func NewResponse[T any](message *T) *Response[T] {
	return connect.NewResponse(message)
}

var (
	WithClientOptions  = connect.WithClientOptions
	WithHandlerOptions = connect.WithHandlerOptions
	WithIdempotency    = connect.WithIdempotency
	WithSchema         = connect.WithSchema
	WithInterceptors   = connect.WithInterceptors
)

type ProtoMessage = proto.Message

var (
	ProtoMarshal   = proto.Marshal
	ProtoUnmarshal = proto.Unmarshal
	ProtoClone     = proto.Clone
	ProtoEqual     = proto.Equal
	ProtoReset     = proto.Reset
)
