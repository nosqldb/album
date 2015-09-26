package gopher

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jimmykuu/wtforms"
	"gopkg.in/mgo.v2/bson"
)

// URL: /admin/links
// 友情链接列表
func adminListLinkExchangesHandler(handler *Handler) {
	c := handler.DB.C(LINKS)
	var linkExchanges []Link
	c.Find(nil).All(&linkExchanges)

	handler.renderTemplate("admin/links.html", ADMIN, map[string]interface{}{
		"linkExchanges": linkExchanges,
	})
}

// ULR: /admin/link/new
// 增加友链
func adminNewLinkExchangeHandler(handler *Handler) {
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
		var linkExchange Link
		err := c.Find(bson.M{"url": form.Value("url")}).One(&linkExchange)

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

// URL: /admin/link/{linkExchangeId}/edit
// 编辑友情链接
func adminEditLinkExchangeHandler(handler *Handler) {
	defer dps.Persist()

	linkExchangeId := mux.Vars(handler.Request)["linkExchangeId"]

	c := handler.DB.C(LINKS)
	var linkExchange Link
	c.Find(bson.M{"_id": bson.ObjectIdHex(linkExchangeId)}).One(&linkExchange)

	form := wtforms.NewForm(
		wtforms.NewTextField("name", "名称", linkExchange.Name, wtforms.Required{}),
		wtforms.NewTextField("url", "URL", linkExchange.URL, wtforms.Required{}, wtforms.URL{}),
		wtforms.NewTextField("description", "描述", linkExchange.Description, wtforms.Required{}),
		wtforms.NewTextField("logo", "Logo", linkExchange.Logo),
	)

	if handler.Request.Method == "POST" {
		if !form.Validate(handler.Request) {
			handler.renderTemplate("links/form.html", ADMIN, map[string]interface{}{
				"linkExchange": linkExchange,
				"form":         form,
				"isNew":        false,
			})
			return
		}

		err := c.Update(bson.M{"_id": linkExchange.Id_}, bson.M{"$set": bson.M{
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
		"linkExchange": linkExchange,
		"form":         form,
		"isNew":        false,
	})
}

// URL: /admin/link/{linkExchangeId}/delete
// 删除友情链接
func adminDeleteLinkExchangeHandler(handler *Handler) {
	linkExchangeId := mux.Vars(handler.Request)["linkExchangeId"]

	c := handler.DB.C(LINKS)
	c.RemoveId(bson.ObjectIdHex(linkExchangeId))

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
