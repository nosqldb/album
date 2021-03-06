/*
一些辅助方法
*/

package g

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"io"
	"errors"
	"github.com/pborman/uuid"
	"github.com/gorilla/sessions"
	"github.com/nosqldb/album/helpers"
	"github.com/jimmykuu/wtforms"
	qiniuIo "github.com/qiniu/api.v6/io"
	"github.com/qiniu/api.v6/rs"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	PerPage = 20
)

var (
	store       *sessions.CookieStore
	fileVersion map[string]string = make(map[string]string) // {path: version}
	utils       *Utils
	usersJson   []byte
)

type Utils struct {
}

func (u *Utils) UserInfo(username string, db *mgo.Database) template.HTML {
	c := db.C(USER)

	user := User{}
	// 检查用户名
	c.Find(bson.M{"username": username}).One(&user)

	format := `<div>
        <a href="/user/%s"><img class="gravatar img-rounded" src="%s" class="gravatar"></a>
        <h4><a href="/user/%s">%s</a><br><small>%s</small></h4>
	<div class="clearfix">
	</div>
    </div>`

	return template.HTML(fmt.Sprintf(format, username, user.AvatarImgSrc(48), username, username, user.Tagline))
}

func (u *Utils) News(username string, db *mgo.Database) template.HTML {
	c := db.C(USER)
	user := User{}
	//检查用户名
	c.Find(bson.M{"username": username}).One(&user)
	format := `<div>
		<hr>
		<a href="/user/%s/news#album">新回复 <span class="label label-pill label-default pull-right">%d</span></a>
		<br>
		<a href="/user/%s/news#at">AT<span class="label label-pill label-default pull-right">%d</span></a>
	</div>
	`
	return template.HTML(fmt.Sprintf(format, username, len(user.RecentReplies), username, len(user.RecentAts)))
}

func (u *Utils) Truncate(html template.HTML, length int) string {
	text := helpers.RemoveFormatting(string(html))
	return helpers.Truncate(text, length, "...")
}

func (u *Utils) HTML(str string) template.HTML {
	return template.HTML(str)
}

func (u *Utils) RenderInput(form wtforms.Form, fieldStr string, inputAttrs ...string) template.HTML {
	field, err := form.Field(fieldStr)
	if err != nil {
		panic(err)
	}

	errorClass := ""

	if field.HasErrors() {
		errorClass = " has-error"
	}

	format := `<fieldset class="form-group%s">
        %s
        %s
        %s
    </fieldset>`

	var inputAttrs2 []string = []string{`class="form-control"`}
	inputAttrs2 = append(inputAttrs2, inputAttrs...)

	return template.HTML(
		fmt.Sprintf(format,
			errorClass,
			field.RenderLabel(),
			field.RenderInput(inputAttrs2...),
			field.RenderErrors()))
}

func (u *Utils) RenderInputH(form wtforms.Form, fieldStr string, labelWidth, inputWidth int, inputAttrs ...string) template.HTML {
	field, err := form.Field(fieldStr)
	if err != nil {
		panic(err)
	}

	errorClass := ""

	if field.HasErrors() {
		errorClass = " has-error"
	}
	format := `<fieldset class="form-group%s">
        %s
        <fieldset class="col-lg-%d">
            %s%s
        </fieldset>
    </fieldset>`
	labelClass := fmt.Sprintf(`class="col-lg-%d control-label"`, labelWidth)

	var inputAttrs2 []string = []string{`class="form-control"`}
	inputAttrs2 = append(inputAttrs2, inputAttrs...)

	return template.HTML(
		fmt.Sprintf(format,
			errorClass,
			field.RenderLabel(labelClass),
			inputWidth,
			field.RenderInput(inputAttrs2...),
			field.RenderErrors(),
		))
}

func (u *Utils) AssertUser(i interface{}) *User {
	v, _ := i.(User)
	return &v
}

func (u *Utils) AssertNode(i interface{}) *Node {
	v, _ := i.(Node)
	return &v
}

func (u *Utils) AssertAlbum(i interface{}) *Album {
	v, _ := i.(Album)
	return &v
}

func message(handler *Handler, title string, message string, class string) {
	handler.renderTemplate("message.html", BASE, map[string]interface{}{"title": title, "message": template.HTML(message), "class": class})
}

// 获取链接的页码，默认"?p=1"这种类型
func Page(r *http.Request) (int, error) {
	p := r.FormValue("p")
	page := 1

	if p != "" {
		var err error
		page, err = strconv.Atoi(p)

		if err != nil {
			return 0, err
		}
	}

	return page, nil
}

// 检查一个string元素是否在数组里面
func stringInArray(a []string, x string) bool {
	sort.Strings(a)
	index := sort.SearchStrings(a, x)

	if index == 0 {
		if a[0] == x {
			return true
		}

		return false
	} else if index > len(a)-1 {
		return false
	}

	return true
}

func staticHandler(templateFile string) HandlerFunc {
	return func(handler *Handler) {
		handler.renderTemplate(templateFile, BASE, map[string]interface{}{})
	}
}

func getPage(r *http.Request) (page int, err error) {
	p := r.FormValue("p")
	page = 1

	if p != "" {
		page, err = strconv.Atoi(p)

		if err != nil {
			return
		}
	}

	return
}

//  提取评论中被at的用户名
func findAts(content string) []string {
	allAts := regexp.MustCompile(`@(\S*) `).FindAllStringSubmatch(content, -1)
	var users []string
	for _, v := range allAts {
		users = append(users, v[1])
	}
	return users
}

func searchHandler(handler *Handler) {
	p := handler.Request.FormValue("p")
	page := 1

	if p != "" {
		var err error
		page, err = strconv.Atoi(p)

		if err != nil {
			message(handler, "页码错误", "页码错误", "error")
			return
		}
	}

	q := handler.Request.FormValue("q")

	keywords := strings.Split(q, " ")

	var noSpaceKeywords []string

	for _, keyword := range keywords {
		temp := strings.TrimSpace(keyword)
		if temp != "" {
			noSpaceKeywords = append(noSpaceKeywords, temp)
		}
	}

	var titleConditions []bson.M
	var markdownConditions []bson.M

	for _, keyword := range noSpaceKeywords {
		titleConditions = append(titleConditions, bson.M{"title": bson.M{"$regex": bson.RegEx{keyword, "i"}}})
		markdownConditions = append(markdownConditions, bson.M{"markdown": bson.M{"$regex": bson.RegEx{keyword, "i"}}})
	}

	c := handler.DB.C(ALBUM)

	var pagination *Pagination

	if len(noSpaceKeywords) == 0 {
		pagination = NewPagination(c.Find(bson.M{}).Sort("-latestrepliedat"), "/search?"+q, PerPage)
	} else {
		pagination = NewPagination(c.Find(bson.M{"$and": []bson.M{
			bson.M{},
			bson.M{"$or": []bson.M{
				bson.M{"$and": titleConditions},
				bson.M{"$and": markdownConditions},
			},
			},
		}}).Sort("-latestrepliedat"), "/search?q="+q, PerPage)
	}

	var albums []Album

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	query.(*mgo.Query).All(&albums)

	if err != nil {
		println(err.Error())
	}

	handler.renderTemplate("search.html", BASE, map[string]interface{}{
		"q":          q,
		"albums":     albums,
		"pagination": pagination,
		"page":       page,
		"active":     "album",
	})
}

// URL: /upload/image
// 编辑器上传图片，接收后上传到七牛
func uploadImageHandler(handler *Handler) {
	file, header, err := handler.Request.FormFile("editormd-image-file")
	if err != nil {
		panic(err)
		return
	}
	defer file.Close()

	// 检查是否是jpg或png文件
	uploadFileType := header.Header["Content-Type"][0]

	filenameExtension := ""
	if uploadFileType == "image/jpeg" {
		filenameExtension = ".jpg"
	} else if uploadFileType == "image/png" {
		filenameExtension = ".png"
	} else if uploadFileType == "image/gif" {
		filenameExtension = ".gif"
	}

	if filenameExtension == "" {
		handler.renderJson(map[string]interface{}{
			"success": 0,
			"message": "不支持的文件格式，请上传 jpg/png/gif 图片",
		})
		return
	}

	// 上传到七牛
	// 文件名：32位uuid+后缀组成
	filename := strings.Replace(uuid.NewUUID().String(), "-", "", -1) + filenameExtension
	key := filename

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
		header.Size,
		nil,
	)

	if err != nil {
		panic(err)

		handler.renderJson(map[string]interface{}{
			"success": 0,
			"message": "图片上传到七牛失败",
		})

		return
	}

	handler.renderJson(map[string]interface{}{
		"success": 1,
		"url":     Config.QiniuDomain + key,
	})
}

// 上传到七牛，并返回文件名
func uploadImageToQiniu(file io.ReadCloser, size int64, contentType string) (filename string, err error) {
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

	key := filename

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

	return Config.QiniuDomain + filename, nil
}
