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

package weapi

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/types"
)

type RecommendSongsReq struct{}

type RecommendSongsResp struct {
	types.RespCommon[RecommendSongsRespData]
}

type RecommendSongsRespData struct {
	DailySongs []struct {
		A  interface{} `json:"a"`
		Al struct {
			Id     int           `json:"id"`
			Name   string        `json:"name"`
			Pic    int64         `json:"pic"`
			PicUrl string        `json:"picUrl"`
			PicStr string        `json:"pic_str,omitempty"`
			Tns    []interface{} `json:"tns"`
		} `json:"al"`
		Alg  string   `json:"alg"`
		Alia []string `json:"alia"`
		Ar   []struct {
			Alias []interface{} `json:"alias"`
			Id    int           `json:"id"`
			Name  string        `json:"name"`
			Tns   []interface{} `json:"tns"`
		} `json:"ar"`
		Cd                   string         `json:"cd"`
		Cf                   string         `json:"cf"`
		Copyright            int            `json:"copyright"`
		Cp                   int            `json:"cp"`
		Crbt                 interface{}    `json:"crbt"`
		DjId                 int            `json:"djId"`
		Dt                   int            `json:"dt"`
		EntertainmentTags    interface{}    `json:"entertainmentTags"`
		Fee                  int            `json:"fee"`
		Ftype                int            `json:"ftype"`
		H                    *types.Quality `json:"h"`
		Hr                   *types.Quality `json:"hr"`
		Id                   int64          `json:"id"`
		L                    *types.Quality `json:"l"`
		M                    *types.Quality `json:"m"`
		Mark                 int64          `json:"mark"`
		Mst                  int            `json:"mst"`
		Mv                   int            `json:"mv"`
		Name                 string         `json:"name"`
		No                   int            `json:"no"`
		NoCopyrightRcmd      interface{}    `json:"noCopyrightRcmd"`
		OriginCoverType      int            `json:"originCoverType"`
		OriginSongSimpleData interface{}    `json:"originSongSimpleData"`
		Pop                  float64        `json:"pop"`
		Privilege            struct {
			ChargeInfoList []struct {
				ChargeMessage interface{} `json:"chargeMessage"`
				ChargeType    int         `json:"chargeType"`
				ChargeUrl     interface{} `json:"chargeUrl"`
				Rate          int         `json:"rate"`
			} `json:"chargeInfoList"`
			Cp                 int    `json:"cp"`
			Cs                 bool   `json:"cs"`
			Dl                 int    `json:"dl"`
			DlLevel            string `json:"dlLevel"`
			DownloadMaxBrLevel string `json:"downloadMaxBrLevel"`
			DownloadMaxbr      int    `json:"downloadMaxbr"`
			Fee                int    `json:"fee"`
			Fl                 int    `json:"fl"`
			FlLevel            string `json:"flLevel"`
			Flag               int    `json:"flag"`
			FreeTrialPrivilege struct {
				CannotListenReason interface{} `json:"cannotListenReason"`
				ListenType         interface{} `json:"listenType"`
				PlayReason         interface{} `json:"playReason"`
				ResConsumable      bool        `json:"resConsumable"`
				UserConsumable     bool        `json:"userConsumable"`
			} `json:"freeTrialPrivilege"`
			Id             int64       `json:"id"`
			MaxBrLevel     string      `json:"maxBrLevel"`
			Maxbr          int         `json:"maxbr"`
			PaidBigBang    bool        `json:"paidBigBang"`
			Payed          int         `json:"payed"`
			Pc             interface{} `json:"pc"`
			Pl             int         `json:"pl"`
			PlLevel        string      `json:"plLevel"`
			PlayMaxBrLevel string      `json:"playMaxBrLevel"`
			PlayMaxbr      int         `json:"playMaxbr"`
			PreSell        bool        `json:"preSell"`
			RealPayed      int         `json:"realPayed"`
			RightSource    int         `json:"rightSource"`
			Rscl           interface{} `json:"rscl"`
			Sp             int         `json:"sp"`
			St             int         `json:"st"`
			Subp           int         `json:"subp"`
			Toast          bool        `json:"toast"`
		} `json:"privilege"`
		Pst             int            `json:"pst"`
		PublishTime     int64          `json:"publishTime"`
		Reason          *string        `json:"reason"`
		RecommendReason *string        `json:"recommendReason"`
		ResourceState   bool           `json:"resourceState"`
		Rt              *string        `json:"rt"`
		RtUrl           interface{}    `json:"rtUrl"`
		RtUrls          []interface{}  `json:"rtUrls"`
		Rtype           int            `json:"rtype"`
		Rurl            interface{}    `json:"rurl"`
		SId             int            `json:"s_id"`
		Single          int            `json:"single"`
		SongJumpInfo    interface{}    `json:"songJumpInfo"`
		Sq              *types.Quality `json:"sq"`
		St              int            `json:"st"`
		T               int            `json:"t"`
		TagPicList      interface{}    `json:"tagPicList"`
		V               int            `json:"v"`
		Version         int            `json:"version"`
	} `json:"dailySongs"`
	MvResourceInfos interface{}   `json:"mvResourceInfos"`
	OrderSongs      []interface{} `json:"orderSongs"`
	// RecommendReasons 推荐原因说明
	RecommendReasons []struct {
		Reason    string      `json:"reason"`
		ReasonId  string      `json:"reasonId"`
		SongId    int64       `json:"songId"`
		TargetUrl interface{} `json:"targetUrl"`
	} `json:"recommendReasons"`
}

// RecommendSongs 每日推荐歌曲列表
// url:
// needLogin: 未知
func (a *Api) RecommendSongs(ctx context.Context, req *RecommendSongsReq) (*RecommendSongsResp, error) {
	var (
		url   = "https://music.163.com/weapi/v3/discovery/recommend/songs"
		reply RecommendSongsResp
		opts  = api.NewOptions()
	)

	resp, err := a.client.Request(ctx, url, req, &reply, opts)
	if err != nil {
		return nil, fmt.Errorf("Request: %w", err)
	}
	_ = resp
	return &reply, nil
}
