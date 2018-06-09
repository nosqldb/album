/*
URL和Handler的Mapping
*/

package g

import (
	"net/http"
	"time"
	"gopkg.in/mgo.v2"
)

// NewHandler返回含有请求上下文的Handler.
func NewHandler(w http.ResponseWriter, r *http.Request) *Handler {
	session, err := mgo.Dial(Config.DB)
	if err != nil {
		panic(err)
	}

	session.SetMode(mgo.Monotonic, true)
	return &Handler{
		ResponseWriter: w,
		Request:        r,
		StartTime:      time.Now(),
		Session:        session,
		DB:             session.DB(Config.DBNAME),
	}
}

// Redirect是重定向的简便方法.
func (h Handler) Redirect(urlStr string) {
	http.Redirect(h.ResponseWriter, h.Request, urlStr, http.StatusFound)
}

// HandlerFunc 是自定义的请求处理函数,接受*Handler作为参数.
type HandlerFunc func(*Handler)

// Route 是代表对应请求的路由规则以及权限的结构体.
type Route struct {
	URL         string
	Permission  PerType
	HandlerFunc HandlerFunc
}

func fileHandler(w http.ResponseWriter, req *http.Request) {
	url := req.Method + " " + req.URL.Path
	logger.Println(url)
	filePath := req.URL.Path[1:]
	http.ServeFile(w, req, filePath)
}

// 路由规则
var routes = []Route{
	{"/", Everyone, indexHandler},
	{"/about", Everyone, staticHandler("about.html")},
	{"/faq", Everyone, staticHandler("faq.html")},
	{"/timeline", Everyone, staticHandler("timeline.html")},
	{"/search", Everyone, searchHandler},
	{"/users.json", Everyone, usersJsonHandler},

	{"/admin", Administrator, adminHandler},
	{"/admin/nodes", Administrator, adminListNodesHandler},
	{"/admin/node/new", Administrator, adminNewNodeHandler},
	{"/admin/users", Administrator, adminListUsersHandler},
	{"/admin/user/{userId}/activate", Administrator, adminActivateUserHandler},

	{"/admin/top/albums", Administrator, listTopAlbumsHandler},
	{"/admin/album/{id:[0-9a-f]{24}}/cancel/top", Administrator, cancelTopAlbumHandler},
	{"/admin/album/{id:[0-9a-f]{24}}/set/top", Administrator, setTopAlbumHandler},

	{"/signup", Everyone, signupHandler},
	{"/signin", Everyone, signinHandler},
	{"/signout", Authenticated, signoutHandler},
	{"/activate/{code}", Everyone, activateHandler},
	{"/forgot_password", Everyone, forgotPasswordHandler},
	{"/reset/{code}", Everyone, resetPasswordHandler},

	{"/setting", Authenticated, userCenterHandler},
	{"/setting/change_avatar", Authenticated, changeAvatarHandler},
	{"/setting/upload_avatar", Authenticated, uploadAvatarHandler},
	{"/setting/choose_avatar", Authenticated, chooseAvatarHandler},
	{"/setting/get_gravatar", Authenticated, setAvatarFromGravatar},
	{"/setting/edit_info", Authenticated, editUserInfoHandler},
	{"/setting/change_password", Authenticated, changePasswordHandler},

	{"/nodes", Everyone, nodesHandler},
	{"/node/{node}", Everyone, albumInNodeHandler},

	{"/comment/{albumId:[0-9a-f]{24}}", Authenticated, commentHandler},
	{"/comment/{commentId:[0-9a-f]{24}}/delete", Administrator, deleteCommentHandler},
	{"/comment/{id:[0-9a-f]{24}}.json", Authenticated, commentJsonHandler},
	{"/comment/{id:[0-9a-f]{24}}/edit", Authenticated, editCommentHandler},

	{"/albums/latest", Everyone, latestAlbumsHandler},
	{"/albums/no_reply", Everyone, noReplyAlbumsHandler},
	{"/p", Authenticated, newAlbumHandler},
	{"/p/{albumId:[0-9a-f]{24}}", Everyone, showAlbumHandler},
	{"/p/{albumId:[0-9a-f]{24}}/edit", Authenticated, editAlbumHandler},
	{"/p/{albumId:[0-9a-f]{24}}/collect", Authenticated, collectAlbumHandler},
	{"/p/{albumId:[0-9a-f]{24}}/delete", Administrator, deleteAlbumHandler},

	{"/user/{username}", Everyone, userInfoHandler},
	{"/user/{username}/albums", Everyone, userAlbumsHandler},
	{"/user/{username}/replies", Everyone, userRepliesHandler},
	{"/user/{username}/news", Everyone, userNewsHandler},
	{"/user/{username}/clear/{t}", Authenticated, userNewsClear},
	{"/user/{username}/collect", Everyone, userAlbumsCollectedHandler},
	{"/follow/{username}", Authenticated, followHandler},
	{"/unfollow/{username}", Authenticated, unfollowHandler},
	{"/users", Everyone, usersHandler},
	{"/users/all", Everyone, allUsersHandler},

	{"/upload/image", Authenticated, uploadImageHandler},

	
	//{"/api/v1/albums", Everyone, apiAlbumsHandler},
}
