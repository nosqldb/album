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
	{"/link", Everyone, linksHandler},
	{"/search", Everyone, searchHandler},
	{"/users.json", Everyone, usersJsonHandler},

	{"/admin", Administrator, adminHandler},
	{"/admin/nodes", Administrator, adminListNodesHandler},
	{"/admin/node/new", Administrator, adminNewNodeHandler},
	{"/admin/site_categories", Administrator, adminListSiteCategoriesHandler},
	{"/admin/site_category/new", Administrator, adminNewSiteCategoryHandler},
	{"/admin/users", Administrator, adminListUsersHandler},
	{"/admin/user/{userId}/activate", Administrator, adminActivateUserHandler},
	{"/admin/links", Administrator, adminListLinksHandler},
	{"/admin/link/new", Administrator, adminNewLinkHandler},
	{"/admin/link/{linkId}/edit", Administrator, adminEditLinkHandler},
	{"/admin/link/{linkId}/delete", Administrator, adminDeleteLinkHandler},
	{"/admin/ads", Administrator, adminListAdsHandler},
	{"/admin/ad/new", Administrator, adminNewAdHandler},
	{"/admin/ad/{id:[0-9a-f]{24}}/delete", Administrator, adminDeleteAdHandler},
	{"/admin/ad/{id:[0-9a-f]{24}}/edit", Administrator, adminEditAdHandler},
	{"/admin/book/new", Administrator, newBookHandler},
	{"/admin/books", Administrator, listBooksHandler},
	{"/admin/book/{id}/edit", Administrator, editBookHandler},
	{"/admin/book/{id}/delete", Administrator, deleteBookHandler},
	{"/admin/top/topics", Administrator, listTopTopicsHandler},
	{"/admin/topic/{id:[0-9a-f]{24}}/cancel/top", Administrator, cancelTopTopicHandler},
	{"/admin/topic/{id:[0-9a-f]{24}}/set/top", Administrator, setTopTopicHandler},

	{"/auth/signup", Everyone, authSignupHandler},
	{"/auth/login", Everyone, authLoginHandler},
	{"/signup", Everyone, signupHandler},
	{"/signin", Everyone, signinHandler},
	{"/signout", Authenticated, signoutHandler},
	{"/activate/{code}", Everyone, activateHandler},
	{"/forgot_password", Everyone, forgotPasswordHandler},
	{"/reset/{code}", Everyone, resetPasswordHandler},

	{"/user_center", Authenticated, userCenterHandler},
	{"/user_center/change_avatar", Authenticated, changeAvatarHandler},
	{"/user_center/upload_avatar", Authenticated, uploadAvatarHandler},
	{"/user_center/choose_avatar", Authenticated, chooseAvatarHandler},
	{"/user_center/get_gravatar", Authenticated, setAvatarFromGravatar},
	{"/user_center/edit_info", Authenticated, editUserInfoHandler},
	{"/user_center/change_password", Authenticated, changePasswordHandler},

	{"/nodes", Everyone, nodesHandler},
	{"/node/{node}", Everyone, topicInNodeHandler},

	{"/comment/{topicId:[0-9a-f]{24}}", Authenticated, commentHandler},
	{"/comment/{commentId:[0-9a-f]{24}}/delete", Administrator, deleteCommentHandler},
	{"/comment/{id:[0-9a-f]{24}}.json", Authenticated, commentJsonHandler},
	{"/comment/{id:[0-9a-f]{24}}/edit", Authenticated, editCommentHandler},

	{"/topics/latest", Everyone, latestTopicsHandler},
	{"/topics/no_reply", Everyone, noReplyTopicsHandler},
	{"/topic/new", Authenticated, newTopicHandler},
	{"/new/{node}", Authenticated, newTopicHandler},
	{"/p/{topicId:[0-9a-f]{24}}", Everyone, showTopicHandler},
	{"/p/{topicId:[0-9a-f]{24}}/edit", Authenticated, editTopicHandler},
	{"/p/{topicId:[0-9a-f]{24}}/collect", Authenticated, collectTopicHandler},
	{"/p/{topicId:[0-9a-f]{24}}/delete", Administrator, deleteTopicHandler},

	{"/user/{username}", Everyone, userInfoHandler},
	{"/user/{username}/topics", Everyone, userTopicsHandler},
	{"/user/{username}/replies", Everyone, userRepliesHandler},
	{"/user/{username}/news", Everyone, userNewsHandler},
	{"/user/{username}/clear/{t}", Authenticated, userNewsClear},
	{"/user/{username}/collect", Everyone, userTopicsCollectedHandler},
	{"/follow/{username}", Authenticated, followHandler},
	{"/unfollow/{username}", Authenticated, unfollowHandler},
	{"/users", Everyone, usersHandler},
	{"/users/all", Everyone, allUsersHandler},
	{"/users/city/{cityName}", Everyone, usersInTheSameCityHandler},

	{"/sites", Everyone, sitesHandler},
	{"/site/new", Authenticated, newSiteHandler},
	{"/site/{siteId:[0-9a-f]{24}}/edit", Authenticated, editSiteHandler},
	{"/site/{siteId:[0-9a-f]{24}}/delete", Administrator, deleteSiteHandler},

	{"/books", Everyone, booksHandler},
	{"/book/{id}", Everyone, showBookHandler},

	{"/download", Everyone, downloadHandler},

	{"/upload/image", Authenticated, uploadImageHandler},

	{"/api/v1/topics", Everyone, apiTopicsHandler},
}
