/*
主题
*/

package g

import (
	"fmt"
	"html/template"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"github.com/jimmykuu/wtforms"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//  用于测试
var testParam func() = func() {}

func albumsHandler(handler *Handler, conditions bson.M, sortBy string, url string, subActive string) {
	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	var node []Node
	c := handler.DB.C(NODE)
	c.Find(bson.M{"albumcount": bson.M{"$gt": 0}}).Sort("-albumcount").All(&node)

	var status Status
	c = handler.DB.C(STATUS)
	c.Find(nil).One(&status)

	c = handler.DB.C(ALBUM)

	var topAlbums []Album

	if page == 1 {
		c.Find(bson.M{"is_top": true}).Sort(sortBy).All(&topAlbums)

		var objectIds []bson.ObjectId
		for _, album := range topAlbums {
			objectIds = append(objectIds, album.Id_)
		}
		if len(topAlbums) > 0 {
			conditions["_id"] = bson.M{"$not": bson.M{"$in": objectIds}}
		}
	}

	pagination := NewPagination(c.Find(conditions).Sort(sortBy), url, PerPage)

	var albums []Album

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	query.(*mgo.Query).All(&albums)

	albums = append(topAlbums, albums...)

	handler.renderTemplate("index.html", BASE, map[string]interface{}{
		"nodes":         node,
		"status":        status,
		"albums":     albums,
		"pagination":    pagination,
		"page":          page,
		"active":        "album",
		"subActive":     subActive,
	})
}

// URL: /
// 网站首页,列出按回帖时间倒序排列的第一页
func indexHandler(handler *Handler) {
	albumsHandler(handler, bson.M{}, "-latestrepliedat", "/", "latestReply")
}

// URL: /albums/latest
// 最新发布的主题，按照发布时间倒序排列
func latestAlbumsHandler(handler *Handler) {
	albumsHandler(handler, bson.M{}, "-createdat", "/albums/latest", "latestCreate")
}

// URL: /albums/no_reply
// 无人回复的主题，按照发布时间倒序排列
func noReplyAlbumsHandler(handler *Handler) {
	albumsHandler(handler, bson.M{"commentcount": 0}, "-createdat", "/albums/no_reply", "noReply")
}

// URL: /p
// 发布主题
func newAlbumHandler(handler *Handler) {


	nodeId := mux.Vars(handler.Request)["node"]

	var nodes []Node
	c := handler.DB.C(NODE)
	c.Find(nil).All(&nodes)

	var choices = []wtforms.Choice{wtforms.Choice{Value:"", Label:"选择节点"}} // 第一个选项为选择节点

	for _, node := range nodes {
		choices = append(choices, wtforms.Choice{Value: node.Id_.Hex(), Label: node.Name})
	}

	form := wtforms.NewForm(
		wtforms.NewSelectField("node", "节点", choices, nodeId, &wtforms.Required{}),
		wtforms.NewTextArea("title", "标题", "", &wtforms.Required{}),
		wtforms.NewTextArea("editormd-markdown-doc", "内容", ""),
		wtforms.NewTextArea("editormd-html-code", "HTML", ""),
	)

	if handler.Request.Method == "POST" {	
		if form.Validate(handler.Request) {
			formFile, formHeader, err := handler.Request.FormFile("file")
	        if err != nil {
	        	fmt.Println("newAlbumHandler:", err.Error())
	        }
	        fileSize := formFile.(Sizer).Size()
	        // 检查是否是jpg或png文件
	        uploadFileType := formHeader.Header["Content-Type"][0]
	        url, err := uploadImageToQiniu(formFile, fileSize, uploadFileType)
	        fmt.Println(url)
			
			user, _ := currentUser(handler)

			c = handler.DB.C(ALBUM)

			id_ := bson.NewObjectId()

			now := time.Now()

			nodeId := bson.ObjectIdHex(form.Value("node"))
			err = c.Insert(&Album{
				    Id_:             id_,
				    NodeId:          nodeId,
					Title:     form.Value("title"),
					Photo: url,
					Markdown:  form.Value("editormd-markdown-doc"),
					Html:      template.HTML(form.Value("editormd-html-code")),
					CreatedBy: user.Id_,
					CreatedAt: now,
				    LatestRepliedAt: now,
			})

			if err != nil {
				fmt.Println("newAlbumHandler:", err.Error())
				return
			}

			// 增加Node.AlbumCount
			c = handler.DB.C(NODE)
			c.Update(bson.M{"_id": nodeId}, bson.M{"$inc": bson.M{"albumcount": 1}})

			c = handler.DB.C(STATUS)

			c.Update(nil, bson.M{"$inc": bson.M{"albumcount": 1}})

			http.Redirect(handler.ResponseWriter, handler.Request, "/p/"+id_.Hex(), http.StatusFound)
			return
		}
	}

	handler.renderTemplate("album/form.html", BASE, map[string]interface{}{
		"form":   form,
		"title":  "发布",
		"action": "/p",
		"active": "album",
	})
}

// URL: /p/{albumId}/edit
// 编辑主题
func editAlbumHandler(handler *Handler) {
	user, _ := currentUser(handler)

	albumId := bson.ObjectIdHex(mux.Vars(handler.Request)["albumId"])

	c := handler.DB.C(ALBUM)
	var album Album
	err := c.Find(bson.M{"_id": albumId}).One(&album)

	if err != nil {
		message(handler, "没有该主题", "没有该主题,不能编辑", "error")
		return
	}

	if !album.CanEdit(user.Username, handler.DB) {
		message(handler, "没有该权限", "对不起,你没有权限编辑该主题", "error")
		return
	}

	var nodes []Node
	c = handler.DB.C(NODE)
	c.Find(nil).All(&nodes)

	var choices = []wtforms.Choice{wtforms.Choice{}} // 第一个选项为空

	for _, node := range nodes {
		choices = append(choices, wtforms.Choice{Value: node.Id_.Hex(), Label: node.Name})
	}

	form := wtforms.NewForm(
		wtforms.NewSelectField("node", "节点", choices, album.NodeId.Hex(), &wtforms.Required{}),
		wtforms.NewTextArea("title", "标题", album.Title, &wtforms.Required{}),
		wtforms.NewTextArea("editormd-markdown-doc", "内容", album.Markdown),
		wtforms.NewTextArea("editormd-html-code", "html", ""),
	)

	if handler.Request.Method == "POST" {
		if form.Validate(handler.Request) {
			nodeId := bson.ObjectIdHex(form.Value("node"))
			c = handler.DB.C(ALBUM)
			c.Update(bson.M{"_id": album.Id_}, bson.M{"$set": bson.M{
				"nodeid":            nodeId,
				"title":     form.Value("title"),
				"markdown":  form.Value("editormd-markdown-doc"),
				"html":      template.HTML(form.Value("editormd-html-code")),
				"updatedat": time.Now(),
				"updatedby": user.Id_.Hex(),
			}})

			// 如果两次的节点不同,更新节点的主题数量
			if album.NodeId != nodeId {
				c = handler.DB.C(NODE)
				c.Update(bson.M{"_id": album.NodeId}, bson.M{"$inc": bson.M{"albumcount": -1}})
				c.Update(bson.M{"_id": nodeId}, bson.M{"$inc": bson.M{"albumcount": 1}})
			}

			http.Redirect(handler.ResponseWriter, handler.Request, "/p/"+album.Id_.Hex(), http.StatusFound)
			return
		}
	}

	handler.renderTemplate("album/form.html", BASE, map[string]interface{}{
		"form":   form,
		"title":  "编辑",
		"action": "/p/" + albumId + "/edit",
		"active": "album",
	})
}

// URL: /p/{albumId}
// 根据主题的ID,显示主题的信息及回复
func showAlbumHandler(handler *Handler) {
	testParam()
	vars := mux.Vars(handler.Request)
	albumId := vars["albumId"]
	c := handler.DB.C(ALBUM)
	cusers := handler.DB.C(USER)
	album := Album{}

	if !bson.IsObjectIdHex(albumId) {
		http.NotFound(handler.ResponseWriter, handler.Request)
		return
	}

	err := c.Find(bson.M{"_id": bson.ObjectIdHex(albumId)}).One(&album)

	if err != nil {
		logger.Println(err)
		http.NotFound(handler.ResponseWriter, handler.Request)
		return
	}

	c.UpdateId(bson.ObjectIdHex(albumId), bson.M{"$inc": bson.M{"hits": 1}})

	user, has := currentUser(handler)

	//去除新消息的提醒
	if has {
		replies := user.RecentReplies
		ats := user.RecentAts
		pos := -1

		for k, v := range replies {
			if v.AlbumId == albumId {
				pos = k
				break
			}
		}

		//数组的删除不是这么删的,早知如此就应该存链表了

		if pos != -1 {
			if pos == len(replies)-1 {
				replies = replies[:pos]
			} else {
				replies = append(replies[:pos], replies[pos+1:]...)
			}
			cusers.Update(bson.M{"_id": user.Id_}, bson.M{"$set": bson.M{"recentreplies": replies}})

		}

		pos = -1

		for k, v := range ats {
			if v.AlbumId == albumId {
				pos = k
				break
			}
		}

		if pos != -1 {
			if pos == len(ats)-1 {
				ats = ats[:pos]
			} else {
				ats = append(ats[:pos], ats[pos+1:]...)
			}

			cusers.Update(bson.M{"_id": user.Id_}, bson.M{"$set": bson.M{"recentats": ats}})
		}
	}

	handler.renderTemplate("album/show.html", BASE, map[string]interface{}{
		"album":  album,
		"active": "album",
	})
}

// URL: /node/{node}
// 列出节点下所有的主题
func albumInNodeHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	nodeId := vars["node"]
	c := handler.DB.C(NODE)

	node := Node{}
	err := c.Find(bson.M{"id": nodeId}).One(&node)

	if err != nil {
		message(handler, "没有此节点", "请联系管理员创建此节点", "error")
		return
	}

	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	c = handler.DB.C(ALBUM)

	pagination := NewPagination(c.Find(bson.M{"nodeid": node.Id_}).Sort("-latestrepliedat"), "/node/" + nodeId, PerPage)

	var albums []Album

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "没有找到页面", "没有找到页面", "error")
		return
	}

	query.(*mgo.Query).All(&albums)

	handler.renderTemplate("/album/list.html", BASE, map[string]interface{}{
		"albums": albums,
		"node":   node,
		"pagination": pagination,
		"page": page,
		"active": "album",
	})
}

// URL: /p/{albumId}/collect/
// 将主题收藏至当前用户的收藏夹
func collectAlbumHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	albumId := vars["albumId"]
	t := time.Now()
	user, _ := currentUser(handler)
	for _, v := range user.AlbumsCollected {
		if v.AlbumId == albumId {
			return
		}
	}
	user.AlbumsCollected = append(user.AlbumsCollected, CollectAlbum{albumId, t})
	c := handler.DB.C(USER)
	c.UpdateId(user.Id_, bson.M{"$set": bson.M{"albumscollected": user.AlbumsCollected}})
	http.Redirect(handler.ResponseWriter, handler.Request, "/user/"+user.Username+"/collect?p=1", http.StatusFound)
}

// URL: /p/{albumId}/delete
// 删除主题
func deleteAlbumHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	albumId := bson.ObjectIdHex(vars["albumId"])

	c := handler.DB.C(ALBUM)

	album := Album{}

	err := c.Find(bson.M{"_id": albumId}).One(&album)
	
	if err != nil {
		fmt.Println("deleteAlbum:", err.Error())
		return
	}

	// Node统计数减一
	c = handler.DB.C(NODE)
	c.Update(bson.M{"_id": album.NodeId}, bson.M{"$inc": bson.M{"albumcount": -1}})

	c = handler.DB.C(STATUS)
	// 统计的主题数减一，减去统计的回复数减去该主题的回复数
	c.Update(nil, bson.M{"$inc": bson.M{"albumcount": -1, "replycount": -album.CommentCount}})

	//删除评论
	c = handler.DB.C(COMMENT)
	if album.CommentCount > 0 {
		c.Remove(bson.M{"albumid": album.Id_})
	}

	// 删除Album记录
	c = handler.DB.C(ALBUM)
	c.Remove(bson.M{"_id": album.Id_})
	
	http.Redirect(handler.ResponseWriter, handler.Request, "/", http.StatusFound)
}

// 列出置顶的主题
func listTopAlbumsHandler(handler *Handler) {
	var albums []Album
	c := handler.DB.C(ALBUM)
	c.Find(bson.M{"is_top": true}).All(&albums)

	handler.renderTemplate("/album/top_list.html", ADMIN, map[string]interface{}{
		"albums": albums,
	})
}

// 设置置顶
func setTopAlbumHandler(handler *Handler) {
	albumId := bson.ObjectIdHex(mux.Vars(handler.Request)["id"])
	c := handler.DB.C(ALBUM)
	c.Update(bson.M{"_id": albumId}, bson.M{"$set": bson.M{"is_top": true}})
	handler.Redirect("/p/" + albumId.Hex())
}

// 取消置顶
func cancelTopAlbumHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	albumId := bson.ObjectIdHex(vars["id"])

	c := handler.DB.C(ALBUM)
	c.Update(bson.M{"_id": albumId}, bson.M{"$set": bson.M{"is_top": false}})
	handler.Redirect("/admin/top/albums")
}
