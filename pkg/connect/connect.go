package connect

import (
	"context"

	real "connectrpc.com/connect"
)

type (
	AnyRequest           = real.AnyRequest
	AnyResponse          = real.AnyResponse
	ClientOption         = real.ClientOption
	Code                 = real.Code
	Error                = real.Error
	Handler              = real.Handler
	HandlerOption        = real.HandlerOption
	HTTPClient           = real.HTTPClient
	IdempotencyLevel     = real.IdempotencyLevel
	Interceptor          = real.Interceptor
	Option               = real.Option
	UnaryFunc            = real.UnaryFunc
	UnaryInterceptorFunc = real.UnaryInterceptorFunc
)

type (
	Request[T any]                      = real.Request[T]
	Response[T any]                     = real.Response[T]
	Client[Req, Res any]                = real.Client[Req, Res]
	ServerStream[Res any]               = real.ServerStream[Res]
	ClientStream[Req any]               = real.ClientStream[Req]
	BidiStream[Req, Res any]            = real.BidiStream[Req, Res]
	ServerStreamForClient[Res any]      = real.ServerStreamForClient[Res]
	ClientStreamForClient[Req, Res any] = real.ClientStreamForClient[Req, Res]
	BidiStreamForClient[Req, Res any]   = real.BidiStreamForClient[Req, Res]
)

const IsAtLeastVersion1_13_0 = real.IsAtLeastVersion1_13_0

const (
	CodeCanceled           = real.CodeCanceled
	CodeUnknown            = real.CodeUnknown
	CodeInvalidArgument    = real.CodeInvalidArgument
	CodeDeadlineExceeded   = real.CodeDeadlineExceeded
	CodeNotFound           = real.CodeNotFound
	CodeAlreadyExists      = real.CodeAlreadyExists
	CodePermissionDenied   = real.CodePermissionDenied
	CodeResourceExhausted  = real.CodeResourceExhausted
	CodeFailedPrecondition = real.CodeFailedPrecondition
	CodeAborted            = real.CodeAborted
	CodeOutOfRange         = real.CodeOutOfRange
	CodeUnimplemented      = real.CodeUnimplemented
	CodeInternal           = real.CodeInternal
	CodeUnavailable        = real.CodeUnavailable
	CodeDataLoss           = real.CodeDataLoss
	CodeUnauthenticated    = real.CodeUnauthenticated
)

const (
	IdempotencyUnknown       = real.IdempotencyUnknown
	IdempotencyNoSideEffects = real.IdempotencyNoSideEffects
	IdempotencyIdempotent    = real.IdempotencyIdempotent
)

func NewError(code Code, underlying error) *Error {
	return real.NewError(code, underlying)
}

func NewRequest[T any](message *T) *Request[T] {
	return real.NewRequest(message)
}

func NewResponse[T any](message *T) *Response[T] {
	return real.NewResponse(message)
}

func NewUnaryHandler[Req, Res any](
	procedure string,
	unary func(context.Context, *Request[Req]) (*Response[Res], error),
	options ...HandlerOption,
) *Handler {
	return real.NewUnaryHandler(procedure, unary, options...)
}

func NewUnaryHandlerSimple[Req, Res any](
	procedure string,
	unary func(context.Context, *Req) (*Res, error),
	options ...HandlerOption,
) *Handler {
	return real.NewUnaryHandlerSimple(procedure, unary, options...)
}

func NewServerStreamHandler[Req, Res any](
	procedure string,
	implementation func(context.Context, *Request[Req], *ServerStream[Res]) error,
	options ...HandlerOption,
) *Handler {
	return real.NewServerStreamHandler(procedure, implementation, options...)
}

func NewServerStreamHandlerSimple[Req, Res any](
	procedure string,
	implementation func(context.Context, *Req, *ServerStream[Res]) error,
	options ...HandlerOption,
) *Handler {
	return real.NewServerStreamHandlerSimple(procedure, implementation, options...)
}

func NewClientStreamHandler[Req, Res any](
	procedure string,
	implementation func(context.Context, *ClientStream[Req]) (*Response[Res], error),
	options ...HandlerOption,
) *Handler {
	return real.NewClientStreamHandler(procedure, implementation, options...)
}

func NewClientStreamHandlerSimple[Req, Res any](
	procedure string,
	implementation func(context.Context, *ClientStream[Req]) (*Res, error),
	options ...HandlerOption,
) *Handler {
	return real.NewClientStreamHandlerSimple(procedure, implementation, options...)
}

func NewBidiStreamHandler[Req, Res any](
	procedure string,
	implementation func(context.Context, *BidiStream[Req, Res]) error,
	options ...HandlerOption,
) *Handler {
	return real.NewBidiStreamHandler(procedure, implementation, options...)
}

func NewClient[Req, Res any](httpClient HTTPClient, url string, options ...ClientOption) *Client[Req, Res] {
	return real.NewClient[Req, Res](httpClient, url, options...)
}

func WithClientOptions(options ...ClientOption) ClientOption {
	return real.WithClientOptions(options...)
}

func WithHandlerOptions(options ...HandlerOption) HandlerOption {
	return real.WithHandlerOptions(options...)
}

func WithIdempotency(idempotencyLevel IdempotencyLevel) Option {
	return real.WithIdempotency(idempotencyLevel)
}

func WithSchema(schema any) Option {
	return real.WithSchema(schema)
}

func WithInterceptors(interceptors ...Interceptor) Option {
	return real.WithInterceptors(interceptors...)
}
