package protoimpl

import real "google.golang.org/protobuf/runtime/protoimpl"

type (
	DescBuilder        = real.DescBuilder
	EnforceVersion     = real.EnforceVersion
	EnumInfo           = real.EnumInfo
	ExtensionFields    = real.ExtensionFields
	ExtensionInfo      = real.ExtensionInfo
	LazyUnmarshalInfo  = real.LazyUnmarshalInfo
	MessageInfo        = real.MessageInfo
	MessageState       = real.MessageState
	Pointer            = real.Pointer
	RaceDetectHookData = real.RaceDetectHookData
	SizeCache          = real.SizeCache
	TypeBuilder        = real.TypeBuilder
	UnknownFields      = real.UnknownFields
	WeakFields         = real.WeakFields
)

const (
	MaxVersion = real.MaxVersion
	GenVersion = real.GenVersion
	MinVersion = real.MinVersion
)

const UnsafeEnabled = real.UnsafeEnabled

var X = real.X
