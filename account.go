/*
处理用户相关的操作,注册,登录,验证,等等
*/
package g

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	. "github.com/nosqldb/album/crypto"
	"github.com/nosqldb/album/email"
	"github.com/pborman/uuid"
	"github.com/dchest/captcha"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jimmykuu/wtforms"
	qiniuIo "github.com/qiniu/api.v6/io"
	"github.com/qiniu/api.v6/rs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var defaultAvatars = []string{
	"gopher_aqua.jpg",
	"gopher_boy.jpg",
	"gopher_brown.jpg",
	"gopher_gentlemen.jpg",
	"gopher_girl.jpg",
	"gopher_strawberry_bg.jpg",
	"gopher_strawberry.jpg",
	"gopher_teal.jpg",
}


// 生成users.json字符串
func generateUsersJson(db *mgo.Database) {
	var users []User
	c := db.C(USER)
	err := c.Find(nil).All(&users)
	if err != nil {
		panic(err)
	}
	var usernames []string
	for _, user := range users {
		usernames = append(usernames, user.Username)
	}

	b, err := json.Marshal(usernames)
	if err != nil {
		panic(err)
	}
	usersJson = b
}

// 返回当前用户
func currentUser(handler *Handler) (*User, bool) {
	r := handler.Request

	session, _ := store.Get(r, "user")

	username, ok := session.Values["username"]

	if !ok {
		return nil, false
	}

	username = username.(string)

	user := User{}

	c := handler.DB.C(USER)

	// 检查用户名
	err := c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		return nil, false
	}

	return &user, true
}


// URL: /signup
// 处理用户注册,要求输入用户名,密码和邮箱
func signupHandler(handler *Handler) {
	// 如果已经登录了，跳转到首页
	_, has := currentUser(handler)
	if has {
		handler.Redirect("/")
	}

	var username string
	var email string

	form := wtforms.NewForm(
		wtforms.NewTextField("username", "用户名", username, wtforms.Required{}, wtforms.Regexp{Expr: `^[a-zA-Z0-9_\p{Han}]{3,16}$`, Message: "请使用a-z, A-Z, 0-9以及下划线或中文, 长度3-16之间"}),
		wtforms.NewPasswordField("password", "密码", wtforms.Required{}),
		wtforms.NewTextField("email", "电子邮件", email, wtforms.Required{}, wtforms.Email{}),
		wtforms.NewTextField("captcha", "验证码", "", wtforms.Required{}),
		wtforms.NewHiddenField("captchaId", ""),
	)

	if handler.Request.Method == "POST" {
		if form.Validate(handler.Request) {
			// 检查验证码
			if !captcha.VerifyString(form.Value("captchaId"), form.Value("captcha")) {
				form.AddError("captcha", "验证码错误")
				fmt.Println("captcha")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signup.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			c := handler.DB.C(USER)

			result := User{}

			// 检查用户名
			err := c.Find(bson.M{"username": form.Value("username")}).One(&result)
			if err == nil {
				form.AddError("username", "该用户名已经被注册")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signup.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			// 检查邮箱
			err = c.Find(bson.M{"email": form.Value("email")}).One(&result)

			if err == nil {
				form.AddError("email", "电子邮件地址已经被注册")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signup.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			c2 := handler.DB.C(STATUS)
			var status Status
			c2.Find(nil).One(&status)

			id := bson.NewObjectId()
			username := form.Value("username")
			validateCode := strings.Replace(uuid.NewUUID().String(), "-", "", -1)
			index := status.UserIndex + 1
			u := &User{
				Id_:          id,
				Username:     username,
				Password:     GenPwd(form.Value("password")),
				Avatar:       "", // defaultAvatars[rand.Intn(len(defaultAvatars))],
				Email:        form.Value("email"),
				ValidateCode: validateCode,
				IsActive:     true,
				JoinedAt:     time.Now(),
				Index:        index,
			}

			err = c.Insert(u)
			if err != nil {
				logger.Println(err)
				return
			}

			c2.Update(nil, bson.M{"$inc": bson.M{"userindex": 1, "usercount": 1}})

			// 重新生成users.json字符串
			generateUsersJson(handler.DB)

			// 注册成功后设成登录状态
			session, _ := store.Get(handler.Request, "user")
			session.Values["username"] = username
			session.Save(handler.Request, handler.ResponseWriter)

			// 跳到修改用户信息页面
			handler.redirect("/setting/edit_info", http.StatusFound)
			return
		}
	}
	form.SetValue("captcha", "")
	handler.renderTemplate("account/signup.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
}

// URL: /activate/{code}
// 用户根据邮件中的链接进行验证,根据code找到是否有对应的用户,如果有,修改User.IsActive为true
func activateHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	code := vars["code"]

	var user User

	c := handler.DB.C(USER)

	err := c.Find(bson.M{"validatecode": code}).One(&user)

	if err != nil {
		message(handler, "没有该验证码", "请检查连接是否正确", "error")
		return
	}

	c.Update(bson.M{"_id": user.Id_}, bson.M{"$set": bson.M{"isactive": true, "validatecode": ""}})

	c = handler.DB.C(STATUS)
	var status Status
	c.Find(nil).One(&status)
	c.Update(bson.M{"_id": status.Id_}, bson.M{"$inc": bson.M{"usercount": 1}})

	message(handler, "通过验证", `恭喜你通过验证,请 <a href="/signin">登录</a>.`, "success")
}

// URL: /signin
// 处理用户登录,如果登录成功,设置Cookie
func signinHandler(handler *Handler) {
	// 如果已经登录了，跳转到首页
	_, has := currentUser(handler)
	if has {
		handler.Redirect("/")
	}

	next := handler.Request.FormValue("next")

	form := wtforms.NewForm(
		wtforms.NewHiddenField("next", next),
		wtforms.NewTextField("username", "用户名", "", &wtforms.Required{}),
		wtforms.NewPasswordField("password", "密码", &wtforms.Required{}),
		wtforms.NewTextField("captcha", "验证码", "", wtforms.Required{}),
		wtforms.NewHiddenField("captchaId", ""),
	)

	if handler.Request.Method == "POST" {
		if form.Validate(handler.Request) {
			// 检查验证码
			if !captcha.VerifyString(form.Value("captchaId"), form.Value("captcha")) {
				form.AddError("captcha", "验证码错误")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signin.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			c := handler.DB.C(USER)
			user := User{}

			err := c.Find(bson.M{"username": form.Value("username")}).One(&user)

			if err != nil {
				form.AddError("username", "该用户不存在")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signin.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			if !user.IsActive {
				form.AddError("username", "邮箱没有经过验证,如果没有收到邮件,请联系管理员")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signin.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			if !user.CheckPassword(form.Value("password")) {
				form.AddError("password", "密码和用户名不匹配")
				form.SetValue("captcha", "")

				handler.renderTemplate("account/signin.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
				return
			}

			session, _ := store.Get(handler.Request, "user")
			session.Values["username"] = user.Username
			session.Save(handler.Request, handler.ResponseWriter)

			if form.Value("next") == "" {
				http.Redirect(handler.ResponseWriter, handler.Request, "/", http.StatusFound)
			} else {
				http.Redirect(handler.ResponseWriter, handler.Request, next, http.StatusFound)
			}

			return
		}
	}

	form.SetValue("captcha", "")
	handler.renderTemplate("account/signin.html", BASE, map[string]interface{}{"form": form, "captchaId": captcha.New()})
}

// URL: /signout
// 用户登出,清除Cookie
func signoutHandler(handler *Handler) {
	session, _ := store.Get(handler.Request, "user")
	session.Options = &sessions.Options{MaxAge: -1}
	session.Save(handler.Request, handler.ResponseWriter)
	handler.renderTemplate("account/signout.html", BASE, map[string]interface{}{"signout": true})
}

func followHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	username := vars["username"]

	currUser, _ := currentUser(handler)

	//不能关注自己
	if currUser.Username == username {
		message(handler, "提示", "不能关注自己", "error")
		return
	}

	user := User{}
	c := handler.DB.C(USER)
	err := c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		message(handler, "关注的会员未找到", "关注的会员未找到", "error")
		return
	}

	if user.IsFollowedBy(currUser.Username) {
		message(handler, "你已经关注该会员", "你已经关注该会员", "error")
		return
	}
	c.Update(bson.M{"_id": user.Id_}, bson.M{"$push": bson.M{"fans": currUser.Username}})
	c.Update(bson.M{"_id": currUser.Id_}, bson.M{"$push": bson.M{"follow": user.Username}})

	http.Redirect(handler.ResponseWriter, handler.Request, "/user/"+user.Username, http.StatusFound)
}

func unfollowHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	username := vars["username"]

	currUser, _ := currentUser(handler)

	//不能取消关注自己
	if currUser.Username == username {
		message(handler, "提示", "不能对自己进行操作", "error")
		return
	}

	user := User{}
	c := handler.DB.C(USER)
	err := c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		message(handler, "没有该会员", "没有该会员", "error")
		return
	}

	if !user.IsFollowedBy(currUser.Username) {
		message(handler, "不能取消关注", "该会员不是你的粉丝,不能取消关注", "error")
		return
	}

	c.Update(bson.M{"_id": user.Id_}, bson.M{"$pull": bson.M{"fans": currUser.Username}})
	c.Update(bson.M{"_id": currUser.Id_}, bson.M{"$pull": bson.M{"follow": user.Username}})

	http.Redirect(handler.ResponseWriter, handler.Request, "/user/"+user.Username, http.StatusFound)
}

// URL: /forgot_password
// 忘记密码,输入用户名和邮箱,如果匹配,发出邮件
func forgotPasswordHandler(handler *Handler) {
	form := wtforms.NewForm(
		wtforms.NewTextField("username", "用户名", "", wtforms.Required{}),
		wtforms.NewTextField("email", "电子邮件", "", wtforms.Email{}),
	)

	if handler.Request.Method == "POST" {
		if form.Validate(handler.Request) {
			var user User
			c := handler.DB.C(USER)
			err := c.Find(bson.M{"username": form.Value("username")}).One(&user)
			if err != nil {
				form.AddError("username", "没有该用户")
			} else if user.Email != form.Value("email") {
				form.AddError("username", "用户名和邮件不匹配")
			} else {
				message2 := `Hi %s,<br>
我们的系统收到一个请求，说你希望通过电子邮件重新设置你在 nosqldb.org 的密码。你可以点击下面的链接开始重设密码：

<a href="%s/reset/%s">%s/reset/%s</a><br>

如果这个请求不是由你发起的，那没问题，你不用担心，你可以安全地忽略这封邮件。

如果你有任何疑问，可以回复<a href="mailto:nosqldb@163.com">nosqldb@163.com</a>向我提问。`
				code := strings.Replace(uuid.NewUUID().String(), "-", "", -1)
				c.Update(bson.M{"_id": user.Id_}, bson.M{"$set": bson.M{"resetcode": code}})
				message2 = fmt.Sprintf(message2, user.Username, Config.Host, code, Config.Host, code)
				email.SendMail(
					"[nosqldb.org]重设密码",
					message2,
					Config.FromEmail,
					[]string{user.Email},
					email.SmtpConfig{
						Username: Config.SmtpUsername,
						Password: Config.SmtpPassword,
						Host:     Config.SmtpHost,
						Addr:     Config.SmtpAddr,
					},
					true,
				)
				message(handler, "通过电子邮件重设密码", "一封包含了重设密码指令的邮件已经发送到你的注册邮箱，按照邮件中的提示，即可重设你的密码。", "success")
				return
			}
		}
	}

	handler.renderTemplate("account/forgot_password.html", BASE, map[string]interface{}{"form": form})
}

// URL: /reset/{code}
// 用户点击邮件中的链接,根据code找到对应的用户,设置新密码,修改完成后清除code
func resetPasswordHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	code := vars["code"]

	var user User
	c := handler.DB.C(USER)
	err := c.Find(bson.M{"resetcode": code}).One(&user)

	if err != nil {
		message(handler, "重设密码", `无效的重设密码标记,可能你已经重新设置过了或者链接已经失效,请通过<a href="/forgot_password">忘记密码</a>进行重设密码`, "error")
		return
	}

	form := wtforms.NewForm(
		wtforms.NewPasswordField("new_password", "新密码", wtforms.Required{}),
		wtforms.NewPasswordField("confirm_password", "确认新密码", wtforms.Required{}),
	)

	if handler.Request.Method == "POST" && form.Validate(handler.Request) {
		if form.Value("new_password") == form.Value("confirm_password") {
			c.Update(
				bson.M{"_id": user.Id_},
				bson.M{
					"$set": bson.M{
						"password":  GenPwd(form.Value("new_password")),
						"resetcode": "",
					},
				},
			)
			message(handler, "重设密码成功", `密码重设成功,你现在可以 <a href="/signin" class="btn btn-primary">登录</a> 了`, "success")
			return
		} else {
			form.AddError("confirm_password", "密码不匹配")
		}
	}

	handler.renderTemplate("account/reset_password.html", BASE, map[string]interface{}{"form": form, "code": code, "account": user.Username})
}

type Sizer interface {
	Size() int64
}

// 上传到七牛，并返回文件名
func uploadAvatarToQiniu(file io.ReadCloser, size int64, contentType string) (filename string, err error) {
	isValidateType := false
	for _, imgType := range []string{"image/png", "image/jpeg"} {
		if imgType == contentType {
			isValidateType = true
			break
		}
	}

	if !isValidateType {
		return "", errors.New("文件类型错误")
	}

	filenameExtension := ".jpg"
	if contentType == "image/png" {
		filenameExtension = ".png"
	}

	// 文件名：32位uuid，不带减号和后缀组成
	filename = strings.Replace(uuid.NewUUID().String(), "-", "", -1) + filenameExtension

	key := "avatar/" + filename

	ret := new(qiniuIo.PutRet)

	var policy = rs.PutPolicy{
		Scope: Config.QiniuBucket,
	}

	err = qiniuIo.Put2(
		nil,
		ret,
		policy.Token(nil),
		key,
		file,
		size,
		nil,
	)

	if err != nil {
		return "", err
	}

	return filename, nil
}

//  URL: /users.json
// 获取所有用户的json列表
func usersJsonHandler(handler *Handler) {
	handler.ResponseWriter.Write(usersJson)
}
