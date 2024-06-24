// MIT License
//
// Copyright (c) 2024 chaunsin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

package api

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"os"
	"time"

	"github.com/chaunsin/netease-cloud-music/pkg/cookie"
	"github.com/chaunsin/netease-cloud-music/pkg/crypto"
	"github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/chaunsin/netease-cloud-music/pkg/utils"

	"github.com/cheggaaa/pb/v3"
	"github.com/go-resty/resty/v2"
	"github.com/google/brotli/go/cbrotli"
)

type Config struct {
	Debug   bool                       `json:"debug" yaml:"debug"`
	Timeout time.Duration              `json:"timeout" yaml:"timeout"`
	Retry   int                        `json:"retry" yaml:"retry"`
	Cookie  cookie.PersistentJarConfig `json:"cookie" yaml:"cookie"`
	// Agent   *Agent                     `json:"agent" yaml:"agent"`
}

func (c *Config) Validate() error {
	if c.Retry < 0 {
		return errors.New("retry is < 0")
	}
	if c.Timeout < 0 {
		return errors.New("timeout is < 0")
	}
	return nil
}

type Client struct {
	cfg    *Config
	cli    *resty.Client
	cookie *cookie.PersistentJar
	l      *log.Logger
	// agent  *Agent
}

func New(cfg *Config) *Client {
	client, err := NewClient(cfg, log.Default)
	if err != nil {
		panic(err)
	}
	return client
}

func NewClient(cfg *Config, l *log.Logger) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	var opts = []cookie.PersistentJarOption{
		cookie.WithSyncInterval(cfg.Cookie.Interval),
	}
	if cfg.Cookie.Filepath != "" {
		opts = append(opts, cookie.WithFilePath(cfg.Cookie.Filepath))
	}
	if opt := cfg.Cookie.Options; opt != nil && opt.PublicSuffixList != nil {
		opts = append(opts, cookie.WithPublicSuffixList(cfg.Cookie.PublicSuffixList))
	}
	jar, err := cookie.NewPersistentJar(opts...)
	if err != nil {
		return nil, fmt.Errorf("NewPersistentJar: %w", err)
	}

	cli := resty.New()
	cli.SetRetryCount(cfg.Retry)
	cli.SetTimeout(cfg.Timeout)
	cli.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	cli.SetDebug(cfg.Debug)
	cli.SetCookieJar(jar)
	cli.OnAfterResponse(contentEncoding)
	// cli.OnAfterResponse(dump)
	// cli.OnBeforeRequest(encrypt)
	// cli.SetLogger(l)
	// cli.AddRetryHook(func(resp *resty.Response, err error) {
	// 	l.Warnf("URL:%s,RetryCount:%d,RequestBody:%+v StatusCode:%d,ResponseBody:%s CusumeTime:%s Err:%s",
	// 		resp.Request.URL, resp.Request.Attempt, resp.Request.Body, resp.StatusCode(), resp.Body(), resp.Time(), err)
	// })

	c := Client{
		cfg:    cfg,
		cli:    cli,
		cookie: jar,
		l:      l,
		// agent:  NewAgent(),
	}
	return &c, nil
}

func (c *Client) Ping(ctx context.Context) error {
	return nil
}

func (c *Client) Close(ctx context.Context) error {
	c.cli.SetCloseConnection(true)
	return c.cookie.Close(ctx)
}

// Cookie 根据url和cookie name获取cookie
func (c *Client) Cookie(url, name string) (http.Cookie, bool) {
	uri, err := neturl.Parse(url)
	if err != nil {
		log.Warn("cookie parse(%v) err: ", url, err)
		return http.Cookie{}, false
	}
	for _, c := range c.cookie.Cookies(uri) {
		if c.Name == name {
			return *c, true
		}
	}
	return http.Cookie{}, false
}

// Cookies 获取当前所有cookies
func (c *Client) Cookies() []*http.Cookie {
	if c.cli != nil {
		return c.cli.R().Cookies
	}
	return make([]*http.Cookie, 0)
}

// GetCSRF 获取csrf 一般用于weapi接口中使用
func (c *Client) GetCSRF(url string) (string, bool) {
	uri, err := neturl.Parse(url)
	if err != nil {
		log.Warn("GetCSRF parse(%v) err: %s", url, err)
		return "", false
	}
	for _, c := range c.cookie.Cookies(uri) {
		if c.Name == "__csrf_token" && c.Value != "" {
			return c.Value, true
		}
		if c.Name == "__csrf" && c.Value != "" {
			return c.Value, true
		}
	}
	return "", false
}

// NeedLogin 是否需要登录
func (c *Client) NeedLogin(ctx context.Context) bool {
	var need = true
	u, _ := neturl.Parse("https://music.163.com")
	for _, c := range c.cli.GetClient().Jar.Cookies(u) {
		if c.Name == "MUSIC_U" && c.Expires.Before(time.Now()) {
			need = false
			break
		}
	}
	return need
}

// UserInfo 获取用户信息
func (c *Client) UserInfo() interface{} {
	// todo:
	return nil
}

func (c *Client) NewRequest() *resty.Request {
	return c.cli.NewRequest()
}

// Request 接口请求
func (c *Client) Request(ctx context.Context, method, url, cryptoType string, req, resp interface{}) (*resty.Response, error) {
	var (
		encryptData map[string]string
		err         error
		response    *resty.Response
	)

	uri, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	// todo: User-Agent

	request := c.cli.R().
		SetContext(ctx).
		SetHeader("Host", "music.163.com").
		SetHeader("Connection", "keep-alive").
		SetHeader("Accept", "*/*").
		SetHeader("Accept-Encoding", "gzip, deflate, br").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept-language", "zh-CN,zh-Hans;q=0.9").
		SetHeader("Referer", "https://music.163.com").
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) NeteaseMusicDesktop/2.3.17.1034")
	// SetHeader("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/25.1 Chrome/121.0.0.0 Mobile Safari/537.36")

	switch cryptoType {
	case "eapi":
		// todo: set common params
		// var dataHeader = http.Header{}
		// dataHeader.Add("osver", getCookie(options.cookies, "osver"))
		// dataHeader.Add("deviceId", getCookie(options.cookies, "deviceId"))
		// dataHeader.Add("appver", getCookie(options.cookies, "appver", "6.1.1"))
		// dataHeader.Add("versioncode", getCookie(options.cookies, "versioncode", "140"))
		// dataHeader.Add("mobilename", getCookie(options.cookies, "mobilename"))
		// dataHeader.Add("buildver", getCookie(options.cookies, "buildver"))
		// dataHeader.Add("resolution", getCookie(options.cookies, "resolution", "1920x1080"))
		// dataHeader.Add("__csrf", getCookie(options.cookies, "__csrf"))
		// dataHeader.Add("os", getCookie(options.cookies, "os", "android"))
		// dataHeader.Add("channel", getCookie(options.cookies, "channel"))
		// dataHeader.Add("channel", getCookie(options.cookies, "channel"))
		// dataHeader.Add("requestId", fmt.Sprintf("%d_%04d", time.Now().UnixNano()/1000000, r.Intn(1000)))
		// if c := getCookie(options.cookies, "MUSIC_U"); c != "" {
		// 	dataHeader.Add("MUSIC_U", c)
		// }
		// if c := getCookie(options.cookies, "MUSIC_A"); c != "" {
		// 	dataHeader.Add("MUSIC_A", c)
		// }
		// req.Header.Set("Cookie", "")
		// for k, v := range dataHeader {
		// 	req.AddCookie(&http.Cookie{
		// 		Name:  k,
		// 		Value: v[0],
		// 	})
		// }
		// data["header"] = dataHeader

		encryptData, err = crypto.EApiEncrypt(uri.Path, req)
		if err != nil {
			return nil, fmt.Errorf("EApiEncrypt: %w", err)
		}
	case "weapi":
		// todo: 需要替换？因为有些 https://interface.music.163.com/api 得接口也会走这个逻辑
		// reg, _ := regexp.Compile(`\w*api`)
		// url = reg.ReplaceAllString(url, "weapi")
		// url = strings.ReplaceAll(url, "api", "weapi")

		csrf, has := c.GetCSRF(url)
		if !has {
			log.Debug("get csrf token not found")
		}
		request.SetQueryParam("csrf_token", csrf)

		// // request.SetCookie(&http.Cookie{Name: "appver", Value: "2.3.17"})
		// request.SetCookie(&http.Cookie{Name: "appver", Value: "9.0.95"})
		// // request.SetCookie(&http.Cookie{Name: "os", Value: "osx"})
		// request.SetCookie(&http.Cookie{Name: "os", Value: "android"})
		// // request.SetCookie(&http.Cookie{Name: "deviceId", Value: "7A8EB581-E60B-5230-BB5B-E6DAB1FBFA62%7C5FD718A3-0602-4389-B612-EBEFAA7F108B"})
		// // request.SetCookie(&http.Cookie{Name: "WEVNSM", Value: "1.0.0"})
		// // request.SetCookie(&http.Cookie{Name: "channel", Value: "netease"})
		// // request.SetHeader("nm-gcore-status", "1")
		// request.SetHeader("appver", "9.0.95")
		// request.SetHeader("os", "android")

		encryptData, err = crypto.WeApiEncrypt(req)
		if err != nil {
			return nil, fmt.Errorf("WeApiEncrypt: %w", err)
		}
	case "linux":
		encryptData, err = crypto.LinuxApiEncrypt(req)
		if err != nil {
			return nil, fmt.Errorf("LinuxApiEncrypt: %w", err)
		}
	case "api":
		// tips: 不需要加密处理请求
	default:
		return nil, fmt.Errorf("%s crypto mode unknown", cryptoType)
	}
	log.Debug("request: %+v encrypt: %+v", req, encryptData)

	switch method {
	case http.MethodPost:
		response, err = request.SetFormData(encryptData).Post(url)
	case http.MethodGet:
		resp, err = request.Get(url)
	default:
		return nil, fmt.Errorf("%s not surpport http method", method)
	}
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	log.Debug("response: %+v", string(response.Body()))

	var decryptData []byte
	switch cryptoType {
	case "api":
		// tips: api接口返回数据是明文
		fallthrough
	case "eapi":
		// 貌似eapi接口返回数据是明文
		// decryptData, err = crypto.EApiDecrypt(string(response.Body()), "")
		// if err != nil {
		// 	return nil, fmt.Errorf("EApiDecrypt: %w", err)
		// }
		fallthrough
	case "weapi":
		// tips: weapi接口返回数据是明文
		decryptData = response.Body()
	case "linux":
		decryptData, err = crypto.LinuxApiDecrypt(string(response.Body()))
		if err != nil {
			return nil, fmt.Errorf("LinuxApiDecrypt: %w", err)
		}
	}
	log.Debug("decrypt body: %s", string(decryptData))
	if err := json.Unmarshal(decryptData, &resp); err != nil {
		return nil, err
	}
	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", response.StatusCode())
	}
	return response, nil
}

func (c *Client) Upload(ctx context.Context, url string, headers map[string]string, req, resp interface{}, bar *pb.ProgressBar) (*resty.Response, error) {
	file, err := os.Open(req.(string))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	md5, err := utils.MD5Hex(data)
	if err != nil {
		return nil, fmt.Errorf("MD5Hex: %v", err)
	}

	var body any
	if bar != nil {
		body = bar.NewProxyReader(bytes.NewReader(data))
	} else {
		body = bytes.NewReader(data)
	}

	response, err := c.cli.R().
		SetContext(ctx).
		SetHeaders(headers).
		SetHeader("Content-Length", fmt.Sprintf("%d", stat.Size())).
		SetHeader("Content-Type", "audio/mpeg").
		SetHeader("Content-Md5", md5).
		SetHeader("Host", "music.163.com").
		SetHeader("Connection", "keep-alive").
		SetHeader("Accept", "*/*").
		SetHeader("Referer", "https://music.163.com").
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) NeteaseMusicDesktop/2.3.17.1034").
		SetBody(body).
		// SetFile("file", "").
		Post(url)
	if err != nil {
		return nil, err
	}
	log.Debug("response: %+v", string(response.Body()))
	if err := json.Unmarshal(response.Body(), &resp); err != nil {
		return nil, err
	}
	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("http status code: %d", response.StatusCode())
	}
	return response, nil
}

func contentEncoding(c *resty.Client, resp *resty.Response) error {
	kind := resp.Header().Get("Content-Encoding")
	log.Debug("Content-Encoding:%s Uncompressed: %v", kind, resp.RawResponse.Uncompressed)
	switch kind {
	case "deflate":
		// 为何使用zlib库: https://zlib.net/zlib_faq.html#faq39
		data, err := zlib.NewReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return fmt.Errorf("zlib.NewReader: %w", err)
		}
		defer data.Close()
		bodyBytes, err := io.ReadAll(data)
		if err != nil {
			return fmt.Errorf("deflate.ReadAll: %w", err)
		}
		resp.SetBody(bodyBytes)
		// reader:=flate.NewReader(bytes.NewReader(resp.Body()))
		// defer reader.Close()
		// bodyBytes, err := io.ReadAll(reader)
		// if err != nil {
		// 	return err
		// }
		// resp.SetBody(bodyBytes)
	case "br":
		bodyBytes, err := cbrotli.Decode(resp.Body())
		if err != nil {
			return fmt.Errorf("cbrotli.Decode: %w", err)
		}
		resp.SetBody(bodyBytes)
	case "gzip":
		// tips: restry 自身已经实现gzip解压缩
		// reader, err := gzip.NewReader(bytes.NewReader(resp.Body()))
		// if err != nil {
		// 	return err
		// }
		// defer reader.Close()
		// bodyBytes, err := io.ReadAll(reader)
		// if err != nil {
		// 	return err
		// }
		// resp.SetBody(bodyBytes)
	case "":
		// 空则代表是gzip,golang底层会做相应得解压缩处理,为空得原因是,
		// 收到请求后进行解压, 同时删除 Content-Encoding: gzip请求头。
		// 如果想关闭自动解压缩,则可以设置Transport.DisableCompression=true
	default:
		return fmt.Errorf("not supported yet Content-Encoding: %s", kind)
	}
	return nil
}

func dump(c *resty.Client, resp *resty.Response) error {
	// d, err := io.ReadAll(resp.RawBody())
	// if err != nil {
	// 	return fmt.Errorf("ReadAll: %w", err)
	// }
	// log.Debug("rawbody:%s", string(d))

	resp.RawResponse.Body = io.NopCloser(bytes.NewReader(resp.Body()))
	log.Debug("############### http dump ################")

	dumpReq, err := httputil.DumpRequest(resp.Request.RawRequest, true)
	if err != nil {
		return fmt.Errorf("DumpRequest: %w", err)
	}
	log.Debug("---------------- request ----------------\n%s", string(dumpReq))

	dumpResp, err := httputil.DumpResponse(resp.RawResponse, true)
	if err != nil {
		return fmt.Errorf("DumpResponse: %w", err)
	}
	log.Debug("---------------- response ----------------\n%s\n", string(dumpResp))
	return nil
}
