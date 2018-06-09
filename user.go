/*
会员
*/

package g

import (
	"encoding/json"
	"net/http"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const (
	at    = "at"
	reply = "reply"
)

func returnJson(w http.ResponseWriter, input interface{}) {
	js, err := json.Marshal(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// 显示最新加入的会员
// URL: /users
func usersHandler(handler *Handler) {
	c := handler.DB.C(USER)
	var newestUsers []User
	c.Find(nil).Sort("-joinedat").Limit(40).All(&newestUsers)

	usersCount, _ := c.Find(nil).Count()

	handler.renderTemplate("user/index.html", BASE, map[string]interface{}{
		"newestUsers": newestUsers,
		"usersCount":  usersCount,
		"active":        "users",
	})
}

// 显示所有会员
// URL: /users/all
func allUsersHandler(handler *Handler) {
	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	c := handler.DB.C(USER)

	pagination := NewPagination(c.Find(nil).Sort("joinedat"), "/users/all", 40)

	var users []User

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	query.(*mgo.Query).All(&users)

	handler.renderTemplate("user/list.html", BASE, map[string]interface{}{
		"users":    users,
		"active":     "users",
		"pagination": pagination,
		"page":       page,
	})
}

// URL: /user/{username}
// 显示用户信息
func userInfoHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	username := vars["username"]
	c := handler.DB.C(USER)

	user := User{}

	err := c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		message(handler, "会员未找到", "会员未找到", "error")
		return
	}

	handler.renderTemplate("account/info.html", BASE, map[string]interface{}{
		"user":   user,
		"active": "users",
	})
}

// URL: /user/{username}/collect/
// 用户收集的album
func userAlbumsCollectedHandler(handler *Handler) {
	page, err := getPage(handler.Request)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
	}
	vars := mux.Vars(handler.Request)
	username := vars["username"]
	c := handler.DB.C(USER)
	user := User{}
	err = c.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		message(handler, "会员未找到", "会员未找到", "error")
	}
	pagination := NewPagination(user.AlbumsCollected, "/user/"+username+"/collect", 3)
	collects, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
	}
	handler.renderTemplate("account/collects.html", BASE, map[string]interface{}{
		"user":       user,
		"collects":   collects,
		"pagination": pagination,
		"page":       page,
		"active":     "users",
	})
}

// URL: /user/{username}/albums
// 用户发表的所有主题
func userAlbumsHandler(handler *Handler) {
	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	vars := mux.Vars(handler.Request)
	username := vars["username"]
	c := handler.DB.C(USER)

	user := User{}
	err = c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		message(handler, "会员未找到", "会员未找到", "error")
		return
	}

	c = handler.DB.C(ALBUM)

	pagination := NewPagination(c.Find(bson.M{"createdby": user.Id_}).Sort("-latestrepliedat"), "/user/"+username+"/albums", PerPage)

	var albums []Album

	query, err := pagination.Page(page)

	if err != nil {
		message(handler, "没有找到页面", "没有找到页面", "error")
		return
	}

	query.(*mgo.Query).All(&albums)

	handler.renderTemplate("account/albums.html", BASE, map[string]interface{}{
		"user":       user,
		"albums":     albums,
		"pagination": pagination,
		"page":       page,
		"active":     "users",
	})
}

// /user/{username}/replies
// 用户的所有回复
func userRepliesHandler(handler *Handler) {
	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	vars := mux.Vars(handler.Request)
	username := vars["username"]
	c := handler.DB.C(USER)

	user := User{}
	err = c.Find(bson.M{"username": username}).One(&user)

	if err != nil {
		message(handler, "会员未找到", "会员未找到", "error")
		return
	}

	if err != nil {
		message(handler, "没有找到页面", "没有找到页面", "error")
		return
	}

	var replies []Comment

	c = handler.DB.C(COMMENT)

	pagination := NewPagination(c.Find(bson.M{"createdby": user.Id_}).Sort("-createdat"), "/user/"+username+"/replies", PerPage)

	query, err := pagination.Page(page)

	query.(*mgo.Query).All(&replies)

	handler.renderTemplate("account/replies.html", BASE, map[string]interface{}{
		"user":       user,
		"pagination": pagination,
		"page":       page,
		"replies":    replies,
		"active":     "users",
	})
}

// URL: /user/{username}/clear/{t}
func userNewsClear(handler *Handler) {
	vars := mux.Vars(handler.Request)
	username := vars["username"]
	t := vars["t"]
	res := map[string]interface{}{}
	user, ok := currentUser(handler)
	if ok {
		if user.Username == username {
			var user User
			c := handler.DB.C(USER)
			c.Find(bson.M{"username": username}).One(&user)
			if t == at {
				user.RecentAts = user.RecentAts[:0]
				c.Update(bson.M{"username": username}, bson.M{"$set": bson.M{"recentats": user.RecentAts}})
				res["status"] = true
			} else if t == reply {
				user.RecentReplies = user.RecentReplies[:0]
				c.Update(bson.M{"username": username}, bson.M{"$set": bson.M{"recentreplies": user.RecentReplies}})
				res["status"] = true
			} else {
				res["status"] = false
				res["error"] = "Wrong Type"
			}

		} else {
			res["status"] = false
			res["error"] = "Need authentication"
		}
	} else {
		res["status"] = false
		res["error"] = "No such User"
	}
	returnJson(handler.ResponseWriter, res)
}

// URL: /user/{username}/comments
func userNewsHandler(handler *Handler) {
	page, err := getPage(handler.Request)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	vars := mux.Vars(handler.Request)
	username := vars["username"]
	c := handler.DB.C(USER)
	user := User{}
	err = c.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		message(handler, "会员未找到", "会员未找到", "error")
		return
	}

	handler.renderTemplate("account/news.html", BASE, map[string]interface{}{
		"user":     user,
		"page":     page,
		"comments": user.RecentReplies,
		"ats":      user.RecentAts,

		"active": "users",
	})
}

// URL: /user/{username}/comments
func userAtsHandler(handler *Handler) {
	return
}
