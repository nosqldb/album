// 提供 RESTful API 服务

package g

import (
	"gopkg.in/mgo.v2/bson"
)

func apiAlbumsHandler(handler *Handler) {
	c := handler.DB.C(ALBUM)

	var albums []Album

	c.Find(bson.M{}).Sort("-latestrepliedat").Limit(20).All(&albums)

	result := []map[string]interface{}{}
	for _, album := range albums {
		creater := album.Creater(handler.DB)

		result = append(result, map[string]interface{}{
			"id":            album.Id_.Hex(),
			"title":         album.Title,
			"markdown":      album.Markdown,
			"html":          album.Html,
			"comment_count": album.CommentCount,
			"created_by": map[string]interface{}{
				"id":       creater.Id_.Hex(),
				"username": creater.Username,
				"avatar":   creater.AvatarImgSrc(48),
			},
			"created_at":        album.CreatedAt.Format("2006-01-02 15:04:05"),
			"latest_replied_at": album.LatestRepliedAt.Format("2006-01-02 15:04:05"),
		})
	}

	handler.renderJson(result)
}
