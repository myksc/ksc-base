package middleware

import (
	"git.zuoyebang.cc/pkg/golib/v2/utils/metadata"
	"github.com/gin-gonic/gin"
)

func Metadata() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		UseMetadata(ctx)
		ctx.Next()
	}
}

func UseMetadata(ctx *gin.Context) {
	if _, ok := metadata.CtxFromGinContext(ctx); !ok {
		metadata.GinCtxWithCtx(ctx, metadata.NewContext4Gin())
	}
}
