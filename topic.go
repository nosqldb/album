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

func topicsHandler(handler *Handler, conditions bson.M, sortBy string, url string, subActive string) {
	page, err := getPage(handler.Request)

	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	var nodes []Node
	c := handler.DB.C(NODES)
	c.Find(bson.M{"topiccount": bson.M{"$gt": 0}}).Sort("-topiccount").All(&nodes)

	var status Status
	c = handler.DB.C(STATUS)
	c.Find(nil).One(&status)

	c = handler.DB.C(TOPICS)

	var topTopics []Topic

	if page == 1 {
		c.Find(bson.M{"is_top": true}).Sort(sortBy).All(&topTopics)

		var objectIds []bson.ObjectId
		for _, topic := range topTopics {
			objectIds = append(objectIds, topic.Id_)
		}
		if len(topTopics) > 0 {
			conditions["_id"] = bson.M{"$not": bson.M{"$in": objectIds}}
		}
	}

	pagination := NewPagination(c.Find(conditions).Sort(sortBy), url, PerPage)

	var topics []Topic

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "页码错误", "页码错误", "error")
		return
	}

	query.(*mgo.Query).All(&topics)

	var links []Link
	c = handler.DB.C(LINKS)
	c.Find(bson.M{"is_on_home": true}).All(&links)

	topics = append(topTopics, topics...)

	handler.renderTemplate("index.html", BASE, map[string]interface{}{
		"nodes":         nodes,
		"status":        status,
		"topics":        topics,
		"links":         links,
		"pagination":    pagination,
		"page":          page,
		"active":        "topic",
		"subActive":     subActive,
	})
}

// URL: /
// 网站首页,列出按回帖时间倒序排列的第一页
func indexHandler(handler *Handler) {
	topicsHandler(handler, bson.M{}, "-latestrepliedat", "/", "latestReply")
}

// URL: /topics/latest
// 最新发布的主题，按照发布时间倒序排列
func latestTopicsHandler(handler *Handler) {
	topicsHandler(handler, bson.M{}, "-createdat", "/topics/latest", "latestCreate")
}

// URL: /topics/no_reply
// 无人回复的主题，按照发布时间倒序排列
func noReplyTopicsHandler(handler *Handler) {
	topicsHandler(handler, bson.M{"commentcount": 0}, "-createdat", "/topics/no_reply", "noReply")
}

// URL: /write
// 发布主题
func newTopicHandler(handler *Handler) {
	nodeId := mux.Vars(handler.Request)["node"]

	var nodes []Node
	c := handler.DB.C(NODES)
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
			user, _ := currentUser(handler)

			c = handler.DB.C(TOPICS)

			id_ := bson.NewObjectId()

			now := time.Now()

			nodeId := bson.ObjectIdHex(form.Value("node"))
			err := c.Insert(&Topic{
				    Id_:             id_,
				    NodeId:          nodeId,
					Title:     form.Value("title"),
					Markdown:  form.Value("editormd-markdown-doc"),
					Html:      template.HTML(form.Value("editormd-html-code")),
					CreatedBy: user.Id_,
					CreatedAt: now,
				    LatestRepliedAt: now,
			})

			if err != nil {
				fmt.Println("newTopicHandler:", err.Error())
				return
			}

			// 增加Node.TopicCount
			c = handler.DB.C(NODES)
			c.Update(bson.M{"_id": nodeId}, bson.M{"$inc": bson.M{"topiccount": 1}})

			c = handler.DB.C(STATUS)

			c.Update(nil, bson.M{"$inc": bson.M{"topiccount": 1}})

			http.Redirect(handler.ResponseWriter, handler.Request, "/p/"+id_.Hex(), http.StatusFound)
			return
		}
	}

	handler.renderTemplate("topic/form.html", BASE, map[string]interface{}{
		"form":   form,
		"title":  "发布",
		"action": "/write",
		"active": "topic",
	})
}

// URL: /p/{topicId}/edit
// 编辑主题
func editTopicHandler(handler *Handler) {
	user, _ := currentUser(handler)

	topicId := bson.ObjectIdHex(mux.Vars(handler.Request)["topicId"])

	c := handler.DB.C(TOPICS)
	var topic Topic
	err := c.Find(bson.M{"_id": topicId}).One(&topic)

	if err != nil {
		message(handler, "没有该主题", "没有该主题,不能编辑", "error")
		return
	}

	if !topic.CanEdit(user.Username, handler.DB) {
		message(handler, "没有该权限", "对不起,你没有权限编辑该主题", "error")
		return
	}

	var nodes []Node
	c = handler.DB.C(NODES)
	c.Find(nil).All(&nodes)

	var choices = []wtforms.Choice{wtforms.Choice{}} // 第一个选项为空

	for _, node := range nodes {
		choices = append(choices, wtforms.Choice{Value: node.Id_.Hex(), Label: node.Name})
	}

	form := wtforms.NewForm(
		wtforms.NewSelectField("node", "节点", choices, topic.NodeId.Hex(), &wtforms.Required{}),
		wtforms.NewTextArea("title", "标题", topic.Title, &wtforms.Required{}),
		wtforms.NewTextArea("editormd-markdown-doc", "内容", topic.Markdown),
		wtforms.NewTextArea("editormd-html-code", "html", ""),
	)

	if handler.Request.Method == "POST" {
		if form.Validate(handler.Request) {
			nodeId := bson.ObjectIdHex(form.Value("node"))
			c = handler.DB.C(TOPICS)
			c.Update(bson.M{"_id": topic.Id_}, bson.M{"$set": bson.M{
				"nodeid":            nodeId,
				"title":     form.Value("title"),
				"markdown":  form.Value("editormd-markdown-doc"),
				"html":      template.HTML(form.Value("editormd-html-code")),
				"updatedat": time.Now(),
				"updatedby": user.Id_.Hex(),
			}})

			// 如果两次的节点不同,更新节点的主题数量
			if topic.NodeId != nodeId {
				c = handler.DB.C(NODES)
				c.Update(bson.M{"_id": topic.NodeId}, bson.M{"$inc": bson.M{"topiccount": -1}})
				c.Update(bson.M{"_id": nodeId}, bson.M{"$inc": bson.M{"topiccount": 1}})
			}

			http.Redirect(handler.ResponseWriter, handler.Request, "/p/"+topic.Id_.Hex(), http.StatusFound)
			return
		}
	}

	handler.renderTemplate("topic/form.html", BASE, map[string]interface{}{
		"form":   form,
		"title":  "编辑",
		"action": "/p/" + topicId + "/edit",
		"active": "topic",
	})
}

// URL: /p/{topicId}
// 根据主题的ID,显示主题的信息及回复
func showTopicHandler(handler *Handler) {
	testParam()
	vars := mux.Vars(handler.Request)
	topicId := vars["topicId"]
	c := handler.DB.C(TOPICS)
	cusers := handler.DB.C(USERS)
	topic := Topic{}

	if !bson.IsObjectIdHex(topicId) {
		http.NotFound(handler.ResponseWriter, handler.Request)
		return
	}

	err := c.Find(bson.M{"_id": bson.ObjectIdHex(topicId)}).One(&topic)

	if err != nil {
		logger.Println(err)
		http.NotFound(handler.ResponseWriter, handler.Request)
		return
	}

	c.UpdateId(bson.ObjectIdHex(topicId), bson.M{"$inc": bson.M{"hits": 1}})

	user, has := currentUser(handler)

	//去除新消息的提醒
	if has {
		replies := user.RecentReplies
		ats := user.RecentAts
		pos := -1

		for k, v := range replies {
			if v.TopicId == topicId {
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
			if v.TopicId == topicId {
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

	handler.renderTemplate("topic/show.html", BASE, map[string]interface{}{
		"topic":  topic,
		"active": "topic",
	})
}

// URL: /node/{node}
// 列出节点下所有的主题
func topicInNodeHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	nodeId := vars["node"]
	c := handler.DB.C(NODES)

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

	c = handler.DB.C(TOPICS)

	pagination := NewPagination(c.Find(bson.M{"nodeid": node.Id_}).Sort("-latestrepliedat"), "/", 20)

	var topics []Topic

	query, err := pagination.Page(page)
	if err != nil {
		message(handler, "没有找到页面", "没有找到页面", "error")
		return
	}

	query.(*mgo.Query).All(&topics)

	handler.renderTemplate("/topic/list.html", BASE, map[string]interface{}{
		"topics": topics,
		"node":   node,
		"active": "topic",
	})
}

// URL: /p/{topicId}/collect/
// 将主题收藏至当前用户的收藏夹
func collectTopicHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	topicId := vars["topicId"]
	t := time.Now()
	user, _ := currentUser(handler)
	for _, v := range user.TopicsCollected {
		if v.TopicId == topicId {
			return
		}
	}
	user.TopicsCollected = append(user.TopicsCollected, CollectTopic{topicId, t})
	c := handler.DB.C(USERS)
	c.UpdateId(user.Id_, bson.M{"$set": bson.M{"topicscollected": user.TopicsCollected}})
	http.Redirect(handler.ResponseWriter, handler.Request, "/user/"+user.Username+"/collect?p=1", http.StatusFound)
}

// URL: /p/{topicId}/delete
// 删除主题
func deleteTopicHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	topicId := bson.ObjectIdHex(vars["topicId"])

	c := handler.DB.C(TOPICS)

	topic := Topic{}

	err := c.Find(bson.M{"_id": topicId}).One(&topic)
	
	if err != nil {
		fmt.Println("deleteTopic:", err.Error())
		return
	}

	// Node统计数减一
	c = handler.DB.C(NODES)
	c.Update(bson.M{"_id": topic.NodeId}, bson.M{"$inc": bson.M{"topiccount": -1}})

	c = handler.DB.C(STATUS)
	// 统计的主题数减一，减去统计的回复数减去该主题的回复数
	c.Update(nil, bson.M{"$inc": bson.M{"topiccount": -1, "replycount": -topic.CommentCount}})

	//删除评论
	c = handler.DB.C(COMMENTS)
	if topic.CommentCount > 0 {
		c.Remove(bson.M{"topicid": topic.Id_})
	}

	// 删除Topic记录
	c = handler.DB.C(TOPICS)
	c.Remove(bson.M{"_id": topic.Id_})
	
	http.Redirect(handler.ResponseWriter, handler.Request, "/", http.StatusFound)
}

// 列出置顶的主题
func listTopTopicsHandler(handler *Handler) {
	var topics []Topic
	c := handler.DB.C(TOPICS)
	c.Find(bson.M{"is_top": true}).All(&topics)

	handler.renderTemplate("/topic/top_list.html", ADMIN, map[string]interface{}{
		"topics": topics,
	})
}

// 设置置顶
func setTopTopicHandler(handler *Handler) {
	topicId := bson.ObjectIdHex(mux.Vars(handler.Request)["id"])
	c := handler.DB.C(TOPICS)
	c.Update(bson.M{"_id": topicId}, bson.M{"$set": bson.M{"is_top": true}})
	handler.Redirect("/p/" + topicId.Hex())
}

// 取消置顶
func cancelTopTopicHandler(handler *Handler) {
	vars := mux.Vars(handler.Request)
	topicId := bson.ObjectIdHex(vars["id"])

	c := handler.DB.C(TOPICS)
	c.Update(bson.M{"_id": topicId}, bson.M{"$set": bson.M{"is_top": false}})
	handler.Redirect("/admin/top/topics")
}
