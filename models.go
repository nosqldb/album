/*
和MongoDB对应的struct
*/

package g

import (
	"errors"
	"fmt"
	"html/template"
	"time"
	"github.com/gorilla/sessions"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	. "github.com/nosqldb/G/crypto"
)

const (
	DefaultAvatar = "gopher_teal.jpg"

	ADS                 = "ads"
	BOOKS               = "books"
	COMMENTS            = "comments"
	TOPICS              = "topics"
	NODES               = "nodes"
	LINKS               = "links"
	SITE_CATEGORIES     = "sitecategories"
	SITES               = "sites"
	STATUS              = "status"
	USERS               = "users"
	CODE                = "code"

	GITHUB_COM = "github.com"
)

var colors = []string{"#FFCC66", "#66CCFF", "#6666FF", "#FF8000", "#0080FF", "#008040", "#008080"}

//主题id和评论id，用于定位到专门的评论
type At struct {
	User      string
	TopicId string
	CommentId string
}

//主题id和主题标题
type Reply struct {
	TopicId  string
	TopicTitle string
}

//收藏的话题
type CollectTopic struct {
	TopicId       string
	TimeCollected time.Time
}

func (ct *CollectTopic) Topic(db *mgo.Database) *Topic {
	c := db.C(TOPICS)
	var topic Topic
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(ct.TopicId)}).One(&topic)
	if err != nil {
		panic(err)
		return nil
	}
	return &topic
}

// 用户
type User struct {
	Id_             bson.ObjectId `bson:"_id"`
	Username        string        //如果关联社区帐号，默认使用社区的用户名
	Password        string
	Email           string //如果关联社区帐号,默认使用社区的邮箱
	Avatar          string
	Website         string
	Location        string
	Tagline         string
	Bio             string
	Twitter         string
	Weibo           string    // 微博
	GitHubUsername  string    // GitHub 用户名
	JoinedAt        time.Time // 加入时间
	Follow          []string
	Fans            []string
	RecentReplies   []Reply        //存储的是最近回复的主题的objectid.hex
	RecentAts       []At           //存储的是最近评论被AT的主题的objectid.hex
	TopicsCollected []CollectTopic //用户收藏的topic数组
	IsSuperuser     bool           // 是否是超级用户
	IsActive        bool
	ValidateCode    string
	ResetCode       string
	Index           int    // 第几个加入社区
	AccountRef      string //帐号关联的社区
	IdRef           string //关联社区的帐号
	LinkRef         string //关联社区的主页链接
	OrgRef          string //关联社区的组织或者公司
	PictureRef      string //关联社区的头像链接
	Provider        string //关联社区名称,比如 github.com
}

// 增加最近被@
func (u *User) AtBy(c *mgo.Collection, username, TopicIdStr, commentIdStr string) error {
	if username == "" || TopicIdStr == "" || commentIdStr == "" {
		return errors.New("string parameters can not be empty string")
	}

	if len(u.RecentAts) == 0 {
		var user User
		err := c.Find(bson.M{"username": u.Username}).One(&user)
		if err != nil {
			return err
		}
		u = &user
	}

	u.RecentAts = append(u.RecentAts, At{username, TopicIdStr, commentIdStr})
	err := c.Update(bson.M{"username": u.Username}, bson.M{"$set": bson.M{"recentats": u.RecentAts}})
	if err != nil {
		return err
	}
	return nil
}

// 是否是默认头像
func (u *User) IsDefaultAvatar(avatar string) bool {
	filename := u.Avatar
	if filename == "" {
		filename = DefaultAvatar
	}

	return filename == avatar
}

// 插入github注册的用户
func (u *User) GetGithubValues(session *sessions.Session) {
	u.Website = session.Values[GITHUB_LINK].(string)
	u.GitHubUsername = session.Values[GITHUB_ID].(string)
	u.AccountRef = session.Values[GITHUB_NAME].(string)
	u.IdRef = session.Values[GITHUB_ID].(string)
	u.LinkRef = session.Values[GITHUB_LINK].(string)
	u.OrgRef = session.Values[GITHUB_ORG].(string)
	u.PictureRef = session.Values[GITHUB_PICTURE].(string)
	u.Provider = session.Values[GITHUB_PROVIDER].(string)

}

// 检查密码是否正确
func (u User) CheckPassword(password string) bool {
	return ComparePwd(password, u.Password)
}

// 删除通过session传的默认信息
func deleteGithubValues(session *sessions.Session) {
	// 删除session传过来的默认信息
	delete(session.Values, GITHUB_EMAIL)
	delete(session.Values, GITHUB_ID)
	delete(session.Values, GITHUB_LINK)
	delete(session.Values, GITHUB_NAME)
	delete(session.Values, GITHUB_ORG)
	delete(session.Values, GITHUB_PICTURE)
	delete(session.Values, GITHUB_PROVIDER)
}

// 头像的图片地址
func (u *User) AvatarImgSrc(size int) string {
	// 如果没有设置头像，用默认头像
	if u.Avatar == "" {
		return fmt.Sprintf("http://identicon.relucks.org/%s?size=%d", u.Username, size)
	}

	return fmt.Sprintf("http://7fvflv.com1.z0.glb.clouddn.com/avatar/%s?imageView2/2/w/%d/h/%d/q/100", u.Avatar, size, size)
}

// 用户发表的最近10个主题
func (u *User) LatestTopics(db *mgo.Database) *[]Topic {
	c := db.C("topics")
	var topics []Topic

	c.Find(bson.M{"createdby": u.Id_}).Sort("-createdat").Limit(10).All(&topics)

	return &topics
}

// 用户的最近10个回复
func (u *User) LatestReplies(db *mgo.Database) *[]Comment {
	c := db.C("comments")
	var replies []Comment

	c.Find(bson.M{"createdby": u.Id_}).Sort("-createdat").Limit(10).All(&replies)

	return &replies
}

// 是否被某人关注
func (u *User) IsFollowedBy(who string) bool {
	for _, username := range u.Fans {
		if username == who {
			return true
		}
	}

	return false
}

// 是否关注某人
func (u *User) IsFans(who string) bool {
	for _, username := range u.Follow {
		if username == who {
			return true
		}
	}

	return false
}

// getUserByName
func getUserByName(c *mgo.Collection, name string) (*User, error) {
	u := new(User)
	err := c.Find(bson.M{"username": name}).One(u)
	if err != nil {
		return nil, err
	}
	return u, nil

}

// 节点
type Node struct {
	Id_         bson.ObjectId `bson:"_id"`
	Id          string
	Name        string
	Description string
	TopicCount  int
}

// 主题
type Topic struct {
	Id_             bson.ObjectId `bson:"_id"`
	NodeId          bson.ObjectId
	Title        string
	Markdown     string
	Html         template.HTML
	CommentCount int
	Hits         int // 点击数量
	CreatedAt    time.Time
	CreatedBy    bson.ObjectId
	UpdatedAt    time.Time
	UpdatedBy    string
	LatestReplierId string
	LatestRepliedAt time.Time
	IsTop           bool `bson:"is_top"` // 置顶
}

func (t *Topic) Creater(db *mgo.Database) *User {
	t_ := db.C(USERS)
	user := User{}
	t_.Find(bson.M{"_id": t.CreatedBy}).One(&user)

	return &user
}

// 是否有权编辑主题
func (t *Topic) CanEdit(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C(USERS)
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	if user.IsSuperuser {
		return true
	}

	return t.CreatedBy == user.Id_
}

func (t *Topic) CanDelete(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C("users")
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	return user.IsSuperuser
}

func (c *Topic) Updater(db *mgo.Database) *User {
	if c.UpdatedBy == "" {
		return nil
	}

	t_ := db.C(USERS)
	user := User{}
	t_.Find(bson.M{"_id": bson.ObjectIdHex(c.UpdatedBy)}).One(&user)

	return &user
}

func (t *Topic) Comments(db *mgo.Database) *[]Comment {
	t_ := db.C("comments")
	var comments []Comment

	t_.Find(bson.M{"topicid": t.Id_}).Sort("createdat").All(&comments)

	return &comments
}
// 主题所属节点
func (t *Topic) Node(db *mgo.Database) *Node {
	c := db.C("nodes")
	node := Node{}
	c.Find(bson.M{"_id": t.NodeId}).One(&node)

	return &node
}

// 只能收藏未收藏过的主题
func (t *Topic) CanCollect(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C(USERS)
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	has := false
	for _, v := range user.TopicsCollected {
		if v.TopicId == t.Id_.Hex() {
			has = true
		}
	}
	return !has
}
// 主题链接
func (t *Topic) Link(id bson.ObjectId) string {
	return "http://nosqldb.org/p/" + id.Hex()

}

//格式化日期
func (t *Topic) Format(tm time.Time) string {
	return tm.Format(time.RFC822)
}

// 主题的最近的一个回复
func (t *Topic) LatestReplier(db *mgo.Database) *User {
	if t.LatestReplierId == "" {
		return nil
	}

	c := db.C("users")
	user := User{}

	err := c.Find(bson.M{"_id": bson.ObjectIdHex(t.LatestReplierId)}).One(&user)

	if err != nil {
		return nil
	}

	return &user
}

// 状态,MongoDB中只存储一个状态
type Status struct {
	Id_        bson.ObjectId `bson:"_id"`
	UserCount  int
	TopicCount int
	ReplyCount int
	UserIndex  int
}

// 站点分类
type SiteCategory struct {
	Id_  bson.ObjectId `bson:"_id"`
	Name string
}

// 评论
type Comment struct {
	Id_       bson.ObjectId `bson:"_id"`
	TopicId   bson.ObjectId
	Markdown  string
	Html      template.HTML
	CreatedBy bson.ObjectId
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
}

// 评论人
func (c *Comment) Creater(db *mgo.Database) *User {
	c_ := db.C("users")
	user := User{}
	c_.Find(bson.M{"_id": c.CreatedBy}).One(&user)

	return &user
}

// 是否有权删除评论，管理员和作者可删除
func (c *Comment) CanDeleteOrEdit(username string, db *mgo.Database) bool {
	if c.Creater(db).Username == username {
		return true
	}

	var user User
	c_ := db.C("users")
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	return user.IsSuperuser
}

// 主题
func (c *Comment) Topic(db *mgo.Database) *Topic {
	// 内容
	var topic Topic
	c_ := db.C("topics")
	c_.Find(bson.M{"_id": c.TopicId}).One(&topic)
	return &topic
}

type Link struct {
	Id_         bson.ObjectId `bson:"_id"`
	Name        string        `bson:"name"`
	URL         string        `bson:"url"`
	Description string        `bson:"description"`
	Logo        string        `bson:"logo"`
	IsOnHome    bool          `bson:"is_on_home"`   // 是否在首页右侧显示
	IsOnBottom  bool          `bson:"is_on_bottom"` // 是否在底部显示
}

type AD struct {
	Id_      bson.ObjectId `bson:"_id"`
	Position string        `bson:"position"`
	Name     string        `bson:"name"`
	Code     string        `bson:"code"`
	Index    int           `bons:"index"`
}
