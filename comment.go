package g

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
)

// URL: /comment/{albumId}
// 评论，不同内容共用一个评论方法
func commentHandler(handler *Handler) {
	if handler.Request.Method != "POST" {
		return
	}

	user, _ := currentUser(handler)
	albumIdStr := handler.param("albumId")
	albumId := bson.ObjectIdHex(albumIdStr)

	var temp map[string]interface{}
	c := handler.DB.C(ALBUM)
	c.Find(bson.M{"_id": albumId}).One(&temp)

	var contentCreator bson.ObjectId
	contentCreator = temp["createdby"].(bson.ObjectId)

	url := "/p/" + albumIdStr

	c.Update(bson.M{"_id": albumId}, bson.M{"$inc": bson.M{"commentcount": 1}})

	markdown := handler.Request.FormValue("editormd-markdown-doc")
	html := handler.Request.FormValue("editormd-html-code")

	Id_ := bson.NewObjectId()
	now := time.Now()

	c = handler.DB.C(COMMENT)
	c.Insert(&Comment{
		Id_:       Id_,
		AlbumId:   albumId,
		Markdown:  markdown,
		Html:      template.HTML(html),
		CreatedBy: user.Id_,
		CreatedAt: now,
	})
	
	// 修改最后回复用户Id和时间
	c = handler.DB.C(ALBUM)
	c.Update(bson.M{"_id": albumId}, bson.M{"$set": bson.M{"latestreplierid": user.Id_.Hex(), "latestrepliedat": now}})

	// 修改中的回复数量
	c = handler.DB.C(STATUS)
	c.Update(nil, bson.M{"$inc": bson.M{"replycount": 1}})
	// 修改对应用户的最近at.
	c = handler.DB.C(USER)
	usernames := findAts(markdown)
	for _, name := range usernames {
		u, err := getUserByName(c, name)
		if err != nil {
			logger.Println(err)
			continue
		}
		if user.Username != u.Username {
			u.AtBy(c, user.Username, albumIdStr, Id_.Hex())
		}
	}

	//  修改用户的最近回复
	//  该最近回复提醒通过url被点击的时候会被disactive
	//  更新最近的评论
	//  自己的评论就不提示了
	tempTitle := temp["title"].(string)

	if contentCreator.Hex() != user.Id_.Hex() {
		var recentreplies []Reply
		var Creater User
		err := c.Find(bson.M{"_id": contentCreator}).One(&Creater)
		if err != nil {
			fmt.Println(err)
		}
		recentreplies = Creater.RecentReplies
		//添加最近评论所在的主题id
		duplicate := false
		for _, v := range recentreplies {
			if albumIdStr == v.AlbumId {
				duplicate = true
			}
		}
		//如果回复的主题有最近回复的话就不添加进去，因为在同一主题下就能看到
		if !duplicate {
			recentreplies = append(recentreplies, Reply{albumIdStr, tempTitle})

			if err = c.Update(bson.M{"_id": contentCreator}, bson.M{"$set": bson.M{"recentreplies": recentreplies}}); err != nil {
				fmt.Println(err)
			}
		}
	}

	http.Redirect(handler.ResponseWriter, handler.Request, url, http.StatusFound)
}

// delete at by ajax.
func deleteAt(handler *Handler) {

}

// URL: /comment/{commentId}/delete
// 删除评论
func deleteCommentHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	var commentId string = vars["commentId"]

	c := handler.DB.C(COMMENT)
	var comment Comment
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(commentId)}).One(&comment)

	if err != nil {
		message(handler, "评论不存在", "该评论不存在", "error")
		return
	}

	c.Remove(bson.M{"_id": comment.Id_})

	c = handler.DB.C(ALBUM)
	c.Update(bson.M{"_id": comment.AlbumId}, bson.M{"$inc": bson.M{"commentcount": -1}})
	
	var album Album
	c.Find(bson.M{"_id": comment.AlbumId}).One(&album)
	if album.LatestReplierId == comment.CreatedBy.Hex() {
		if album.CommentCount == 0 {
			// 如果删除后没有回复，设置最后回复id为空，最后回复时间为创建时间
			c.Update(bson.M{"_id": album.Id_}, bson.M{"$set": bson.M{"latestreplierid": "", "latestrepliedat": album.CreatedAt}})
		} else {
			// 如果删除的是该主题最后一个回复，设置主题的最新回复id，和时间
			var latestComment Comment
			c = handler.DB.C(COMMENT)
			c.Find(bson.M{"albumid": album.Id_}).Sort("-createdat").Limit(1).One(&latestComment)

			c = handler.DB.C(ALBUM)
			c.Update(bson.M{"_id": album.Id_}, bson.M{"$set": bson.M{"latestreplierid": latestComment.CreatedBy.Hex(), "latestrepliedat": latestComment.CreatedAt}})
		}
	}

	// 修改中的回复数量
	c = handler.DB.C(STATUS)
	c.Update(nil, bson.M{"$inc": bson.M{"replycount": -1}})
	
    url := "/p/" + comment.AlbumId.Hex()
	
	http.Redirect(handler.ResponseWriter, handler.Request, url, http.StatusFound)
}

// URL: /comment/:id.json
// 获取comment的内容
func commentJsonHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	var id string = vars["id"]

	c := handler.DB.C(COMMENT)
	var comment Comment
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&comment)

	if err != nil {
		return
	}

	data := map[string]string{
		"markdown": comment.Markdown,
	}

	handler.renderJson(data)
}

// URL: /commeint/:id/edit
// 编辑comment
func editCommentHandler(handler *Handler) {
	if handler.Request.Method != "POST" {
		return
	}
	vars := mux.Vars(handler.Request)
	var id string = vars["id"]

	c := handler.DB.C(COMMENT)

	user, _ := currentUser(handler)

	comment := Comment{}

	c.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&comment)

	if !comment.CanDeleteOrEdit(user.Username, handler.DB) {
		return
	}

	markdown := handler.Request.FormValue("editormd-edit-markdown-doc")
	html := handler.Request.FormValue("editormd-edit-html-code")

	c.Update(bson.M{"_id": bson.ObjectIdHex(id)}, bson.M{"$set": bson.M{
		"markdown":  markdown,
		"html":      template.HTML(html),
		"updatedby": user.Id_.Hex(),
		"updatedat": time.Now(),
	}})

	var temp map[string]interface{}
	c = handler.DB.C(ALBUM)
	c.Find(bson.M{"_id": comment.AlbumId}).One(&temp)

	url := "/p/" + comment.AlbumId.Hex()

	http.Redirect(handler.ResponseWriter, handler.Request, url, http.StatusFound)
}
