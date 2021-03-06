package accountLogic

import (
	"github.com/medivhzhan/weapp"
	weappUtil "github.com/medivhzhan/weapp/util"
	"encoding/json"
	"github.com/astaxie/beego/context"
	"cherish-time-go/modules/util"
	"cherish-time-go/define/retcode"
	"cherish-time-go/models/User"
	"time"
	"cherish-time-go/define/common"
	"cherish-time-go/cache"
	"fmt"
	"cherish-time-go/models/Time"
	"github.com/astaxie/beego"
)

type AccountLogic struct {
}

type AuthData struct {
	Auth     string         `json:"auth"`
	UserId   string         `json:"userId"`
	UserInfo UserModel.User `json:"userInfo"`
}

func (this *AccountLogic) Login(c *context.Context, code, iv, encryptedData string) (authData AuthData) {
	appID := beego.AppConfig.String("weChat_mini_program_appId")
	secret := beego.AppConfig.String("weChat_mini_program_secret")
	res, err := weapp.Login(appID, secret, code)
	if err != nil {
		util.ThrowApi(c, retcode.WECHAT_LOGIN_ERR, "微信登录失败："+err.Error())
		return
	}

	//解析用户资料
	bts, err := weappUtil.CBCDecrypt(res.SessionKey, encryptedData, iv)
	if err != nil {
		util.ThrowApi(c, retcode.WECHAT_LOGIN_ERR, "微信登录失败："+err.Error())
		return
	}

	userInfo := weapp.Userinfo{}
	json.Unmarshal(bts, &userInfo)

	//查找用户是否存在
	model, err := UserModel.GetByOpenId(userInfo.OpenID)
	fmt.Println(model)
	if err != nil {
		//新建用户
		userModel, ok := UserModel.AddNew(userInfo.OpenID, userInfo.Nickname, userInfo.Gender, userInfo.City, userInfo.Province, userInfo.Country, userInfo.Avatar)
		if !ok {
			util.ThrowApi(c, retcode.WECHAT_LOGIN_ERR, "新建用户失败")
			return
		}
		model = userModel

		//添加一条记录
		date := time.Now().Format("20060102")
		_, ok = TimeModel.AddNew("安装惜时光", userModel.Id, common.TIME_TYPE_ASC, date, `["#fc9e9a", "#fed89c"]`, "记下珍贵的日子")
	} else {
		//更新用户数据
		UserModel.UpdateData(&model, userInfo.Nickname, userInfo.Gender, userInfo.City, userInfo.Province, userInfo.Country, userInfo.Avatar)
	}

	auth := util.GenShortUuid()
	authData.Auth = auth
	authData.UserId = model.Id
	authData.UserInfo = model

	json, _ := util.JsonEncode(authData)

	cache.Bm.Put(auth, json, common.AUTH_EXIST_TIME_HOUR*time.Hour)

	return
}

func (this *AccountLogic) CheckAuth(c *context.Context, auth string) (authData AuthData) {
	redis := cache.Bm.Get(auth)
	if redis == nil {
		util.ThrowApi(c, retcode.ERR_NO_LOGIN, "用户未登录")
		return
	}

	util.JsonDecode(string(redis.([]byte)), &authData)

	return authData
}
