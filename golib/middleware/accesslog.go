package middleware

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/myksc/ksc-base/golib/base"
	"github.com/myksc/ksc-base/golib/utils"
	"github.com/myksc/ksc-base/golib/zlog"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	printRequestLen  = 10240
	printResponseLen = 10240
)

var (
	// 暂不需要，后续考虑看是否需要支持用户配置
	mcpackReqUris []string
	ignoreReqUris []string
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	s = strings.Replace(s, "\n", "", -1)
	if w.body != nil {
		w.body.WriteString(s)
	}
	return w.ResponseWriter.WriteString(s)
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	if w.body != nil {
		//idx := len(b)
		// gin render json 后后面会多一个换行符
		//if b[idx-1] == '\n' {
		//	b = b[:idx-1]
		//}
		w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// access日志打印
func AccessLog() gin.HandlerFunc {
	// 当前模块名
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		// 请求url
		path := c.Request.URL.Path
		// 请求报文
		var requestBody []byte
		if c.Request.Body != nil {
			var err error
			requestBody, err = c.GetRawData()
			if err != nil {
				zlog.Warnf(c, "get http request body error: %s", err.Error())
			}
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody))
		}

		blw := new(bodyLogWriter)
		if printResponseLen <= 0 {
			blw = &bodyLogWriter{body: nil, ResponseWriter: c.Writer}
		} else {
			blw = &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		}
		c.Writer = blw

		c.Set(zlog.ContextKeyUri, path)
		_ = zlog.GetLogID(c)
		_ = zlog.GetRequestID(c)
		// 处理请求
		c.Next()

		response := ""
		if blw.body != nil {
			if len(blw.body.String()) <= printResponseLen {
				response = blw.body.String()
			} else {
				response = blw.body.String()[:printResponseLen]
			}
		}

		bodyStr := ""
		flag := false
		// macpack的请求，以二进制输出日志
		for _, val := range mcpackReqUris {
			if strings.Contains(path, val) {
				bodyStr = fmt.Sprintf("%v", requestBody)
				flag = true
				break
			}
		}
		if !flag {
			// 不打印RequestBody的请求
			for _, val := range ignoreReqUris {
				if strings.Contains(path, val) {
					bodyStr = ""
					flag = true
					break
				}
			}
		}
		if !flag {
			bodyStr = string(requestBody)
		}

		if c.Request.URL.RawQuery != "" {
			bodyStr += "&" + c.Request.URL.RawQuery
		}

		if len(bodyStr) > printRequestLen {
			bodyStr = bodyStr[:printRequestLen]
		}

		// 结束时间
		end := time.Now()

		// 用户自定义notice
		var customerFields []zlog.Field
		for k, v := range zlog.GetCustomerKeyValue(c) {
			customerFields = append(customerFields, zlog.Reflect(k, v))
		}

		// 固定notice
		commonFields := []zlog.Field{
			zlog.String("cuid", getReqValueByKey(c, "cuid")),
			zlog.String("device", getReqValueByKey(c, "device")),
			zlog.String("channel", getReqValueByKey(c, "channel")),
			zlog.String("os", getReqValueByKey(c, "os")),
			zlog.String("vc", getReqValueByKey(c, "vc")),
			zlog.String("vcname", getReqValueByKey(c, "vcname")),
			zlog.String("userid", getReqValueByKey(c, "userid")),
			zlog.String("uri", c.Request.RequestURI),
			zlog.String("host", c.Request.Host),
			zlog.String("method", c.Request.Method),
			zlog.String("httpProto", c.Request.Proto),
			zlog.String("handle", c.HandlerName()),
			zlog.String("userAgent", c.Request.UserAgent()),
			zlog.String("refer", c.Request.Referer()),
			zlog.String("clientIp", utils.GetClientIp(c)),
			zlog.String("cookie", getCookie(c)),
			zlog.String("requestStartTime", utils.GetFormatRequestTime(start)),
			zlog.String("requestEndTime", utils.GetFormatRequestTime(end)),
			zlog.Float64("cost", utils.GetRequestCost(start, end)),
			zlog.String("requestParam", bodyStr),
			zlog.Int("responseStatus", c.Writer.Status()),
			zlog.String("response", response),
		}

		commonFields = append(commonFields, customerFields...)
		zlog.InfoLogger(c, "notice", commonFields...)
	}
}

// 从request body中解析特定字段作为notice key打印
func getReqValueByKey(ctx *gin.Context, k string) string {
	if vs, exist := ctx.Request.Form[k]; exist && len(vs) > 0 {
		return vs[0]
	}
	return ""
}

func getCookie(ctx *gin.Context) string {
	cStr := ""
	for _, c := range ctx.Request.Cookies() {
		cStr += fmt.Sprintf("%s=%s&", c.Name, c.Value)
	}
	return strings.TrimRight(cStr, "&")
}

// access 添加kv打印
func AddNotice(k string, v interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		zlog.AddNotice(c, k, v)
		c.Next()
	}
}

func LoggerBeforeRun(ctx *gin.Context) {
	fields := []zlog.Field{
		zlog.String("handle", ctx.HandlerName()),
		zlog.String("type", ctx.ContentType()),
	}

	zlog.InfoLogger(ctx, "start", fields...)
}

func LoggerAfterRun(ctx *gin.Context) {
	_, err := ctx.GetRawData();
	if err != nil {
		err = errors.Cause(err)
		base.StackLogger(ctx, err)
	}

	// 用户自定义notice
	notices := zlog.GetCustomerKeyValue(ctx)

	var fields []zlog.Field
	for k, v := range notices {
		fields = append(fields, zlog.Reflect(k, v))
	}

	fields = append(fields,
		zlog.String("handle", ctx.HandlerName()),
		zlog.String("type", ctx.ContentType()),
		zlog.String("error", fmt.Sprintf("%+v", err)),
	)

	zlog.InfoLogger(ctx, "end", fields...)
}

