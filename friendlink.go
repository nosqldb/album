package g

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/jimmykuu/wtforms"
	"gopkg.in/mgo.v2/bson"
)

// URL: /admin/links
// 友情链接列表
func adminListLinksHandler(handler *Handler) {
	c := handler.DB.C(LINKS)
	var links []Link
	c.Find(nil).All(&links)

	handler.renderTemplate("admin/links.html", ADMIN, map[string]interface{}{
		"links": links,
	})
}

// ULR: /admin/link/new
// 增加友链
func adminNewLinkHandler(handler *Handler) {
	defer dps.Persist()

	form := wtforms.NewForm(
		wtforms.NewTextField("name", "名称", "", wtforms.Required{}),
		wtforms.NewTextField("url", "URL", "", wtforms.Required{}, wtforms.URL{}),
		wtforms.NewTextField("description", "描述", "", wtforms.Required{}),
		wtforms.NewTextField("logo", "Logo", ""),
	)

	if handler.Request.Method == "POST" {
		if !form.Validate(handler.Request) {
			handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
				"form":  form,
				"isNew": true,
			})
			return
		}

		c := handler.DB.C(LINKS)
		var link Link
		err := c.Find(bson.M{"url": form.Value("url")}).One(&link)

		if err == nil {
			form.AddError("url", "该URL已经有了")
			handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
				"form":  form,
				"isNew": true,
			})
			return
		}

		err = c.Insert(&Link{
			Id_:         bson.NewObjectId(),
			Name:        form.Value("name"),
			URL:         form.Value("url"),
			Description: form.Value("description"),
			Logo:        form.Value("logo"),
			IsOnHome:    handler.Request.FormValue("is_on_home") == "on",
			IsOnBottom:  handler.Request.FormValue("is_on_bottom") == "on",
		})

		if err != nil {
			panic(err)
		}

		http.Redirect(handler.ResponseWriter, handler.Request, "/admin/links", http.StatusFound)
		return
	}

	handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
		"form":  form,
		"isNew": true,
	})
}

// URL: /admin/link/{linkId}/edit
// 编辑友情链接
func adminEditLinkHandler(handler *Handler) {
	defer dps.Persist()

	linkId := mux.Vars(handler.Request)["linkId"]

	c := handler.DB.C(LINKS)
	var link Link
	c.Find(bson.M{"_id": bson.ObjectIdHex(linkId)}).One(&link)

	form := wtforms.NewForm(
		wtforms.NewTextField("name", "名称", link.Name, wtforms.Required{}),
		wtforms.NewTextField("url", "URL", link.URL, wtforms.Required{}, wtforms.URL{}),
		wtforms.NewTextField("description", "描述", link.Description, wtforms.Required{}),
		wtforms.NewTextField("logo", "Logo", link.Logo),
	)

	if handler.Request.Method == "POST" {
		if !form.Validate(handler.Request) {
			handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
				"link": link,
				"form":         form,
				"isNew":        false,
			})
			return
		}

		err := c.Update(bson.M{"_id": link.Id_}, bson.M{"$set": bson.M{
			"name":         form.Value("name"),
			"url":          form.Value("url"),
			"description":  form.Value("description"),
			"logo":         form.Value("logo"),
			"is_on_home":   handler.Request.FormValue("is_on_home") == "on",
			"is_on_bottom": handler.Request.FormValue("is_on_bottom") == "on",
		}})

		if err != nil {
			panic(err)
		}

		http.Redirect(handler.ResponseWriter, handler.Request, "/admin/links", http.StatusFound)
		return
	}

	handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
		"link": link,
		"form":         form,
		"isNew":        false,
	})
}

// URL: /admin/link/{linkId}/delete
// 删除友情链接
func adminDeleteLinkHandler(handler *Handler) {
	linkId := mux.Vars(handler.Request)["linkId"]

	c := handler.DB.C(LINKS)
	c.RemoveId(bson.ObjectIdHex(linkId))

	handler.ResponseWriter.Write([]byte("true"))
}

// URL: /link
// 友情链接
func linksHandler(handler *Handler) {
	var links []Link
	c := handler.DB.C(LINKS)
	c.Find(nil).All(&links)
	handler.renderTemplate("links/all.html", BASE, map[string]interface{}{
		"links": links,
	})
}
