package requestctx

import (
	"context"
	"strings"
)

type metaKey string

const requestMetaKey metaKey = "request_meta"

type Meta struct {
	IPAddress string
	UserAgent string
}

func WithMeta(ctx context.Context, ipAddress, userAgent string) context.Context {
	return context.WithValue(ctx, requestMetaKey, Meta{
		IPAddress: strings.TrimSpace(ipAddress),
		UserAgent: strings.TrimSpace(userAgent),
	})
}

func GetMeta(ctx context.Context) Meta {
	if ctx == nil {
		return Meta{}
	}
	value := ctx.Value(requestMetaKey)
	meta, ok := value.(Meta)
	if !ok {
		return Meta{}
	}
	return meta
}
