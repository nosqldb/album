/*
和MongoDB对应的struct
*/

package g

import (
	"errors"
	"fmt"
	"html/template"
	"time"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	. "github.com/nosqldb/album/crypto"
)

const (
	DefaultAvatar = "gopher_teal.jpg"

	COMMENT            = "comment"
	ALBUM              = "album"
	NODE               = "node"
	STATUS              = "status"
	USER               = "user"
	CODE                = "code"

)

var colors = []string{"#FFCC66", "#66CCFF", "#6666FF", "#FF8000", "#0080FF", "#008040", "#008080"}

//主题id和评论id，用于定位到专门的评论
type At struct {
	User      string
	AlbumId string
	CommentId string
}

//主题id和主题标题
type Reply struct {
	AlbumId  string
	AlbumTitle string
}

//收藏的话题
type CollectAlbum struct {
	AlbumId       string
	TimeCollected time.Time
}

func (ct *CollectAlbum) Album(db *mgo.Database) *Album {
	c := db.C(ALBUM)
	var album Album
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(ct.AlbumId)}).One(&album)
	if err != nil {
		panic(err)
		return nil
	}
	return &album
}

// 用户
type User struct {
	Id_             bson.ObjectId `bson:"_id"`
	Username        string        //如果关联社区帐号，默认使用社区的用户名
	Password        string
	Email           string //如果关联社区帐号,默认使用社区的邮箱
	Avatar          string
	Website         string
	Tagline         string
	Bio             string
	Weibo           string    // 微博
	JoinedAt        time.Time // 加入时间
	Follow          []string
	Fans            []string
	RecentReplies   []Reply        //存储的是最近回复的主题的objectid.hex
	RecentAts       []At           //存储的是最近评论被AT的主题的objectid.hex
	AlbumsCollected []CollectAlbum //用户收藏的album数组
	IsSuperuser     bool           // 是否是超级用户
	IsActive        bool
	ValidateCode    string
	ResetCode       string
	Index           int    // 第几个加入社区
}

// 增加最近被@
func (u *User) AtBy(c *mgo.Collection, username, AlbumIdStr, commentIdStr string) error {
	if username == "" || AlbumIdStr == "" || commentIdStr == "" {
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

	u.RecentAts = append(u.RecentAts, At{username, AlbumIdStr, commentIdStr})
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


// 检查密码是否正确
func (u User) CheckPassword(password string) bool {
	return ComparePwd(password, u.Password)
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
func (u *User) LatestAlbums(db *mgo.Database) *[]Album {
	c := db.C("album")
	var album []Album

	c.Find(bson.M{"createdby": u.Id_}).Sort("-createdat").Limit(10).All(&album)

	return &album
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
	AlbumCount  int
}

// 主题
type Album struct {
	Id_             bson.ObjectId `bson:"_id"`
	NodeId          bson.ObjectId
	Title        string
	Photo        string
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

func (t *Album) Creater(db *mgo.Database) *User {
	t_ := db.C(USER)
	user := User{}
	t_.Find(bson.M{"_id": t.CreatedBy}).One(&user)

	return &user
}

// 是否有权编辑主题
func (t *Album) CanEdit(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C(USER)
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	if user.IsSuperuser {
		return true
	}

	return t.CreatedBy == user.Id_
}

func (t *Album) CanDelete(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C("users")
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	return user.IsSuperuser
}

func (c *Album) Updater(db *mgo.Database) *User {
	if c.UpdatedBy == "" {
		return nil
	}

	t_ := db.C(USER)
	user := User{}
	t_.Find(bson.M{"_id": bson.ObjectIdHex(c.UpdatedBy)}).One(&user)

	return &user
}

func (t *Album) Comments(db *mgo.Database) *[]Comment {
	t_ := db.C("comments")
	var comments []Comment

	t_.Find(bson.M{"albumid": t.Id_}).Sort("createdat").All(&comments)

	return &comments
}
// 主题所属节点
func (t *Album) Node(db *mgo.Database) *Node {
	c := db.C("nodes")
	node := Node{}
	c.Find(bson.M{"_id": t.NodeId}).One(&node)

	return &node
}

// 只能收藏未收藏过的主题
func (t *Album) CanCollect(username string, db *mgo.Database) bool {
	var user User
	t_ := db.C(USER)
	err := t_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	has := false
	for _, v := range user.AlbumsCollected {
		if v.AlbumId == t.Id_.Hex() {
			has = true
		}
	}
	return !has
}
// 主题链接
func (t *Album) Link(id bson.ObjectId) string {
	return "http://nosqldb.org/p/" + id.Hex()

}

//格式化日期
func (t *Album) Format(tm time.Time) string {
	return tm.Format(time.RFC822)
}

// 主题的最近的一个回复
func (t *Album) LatestReplier(db *mgo.Database) *User {
	if t.LatestReplierId == "" {
		return nil
	}

	c := db.C(USER)
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
	AlbumCount int
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
	AlbumId   bson.ObjectId
	Markdown  string
	Html      template.HTML
	CreatedBy bson.ObjectId
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
}

// 评论人
func (c *Comment) Creater(db *mgo.Database) *User {
	c_ := db.C(USER)
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
	c_ := db.C(USER)
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	return user.IsSuperuser
}

// 主题
func (c *Comment) Album(db *mgo.Database) *Album {
	// 内容
	var album Album
	c_ := db.C(ALBUM)
	c_.Find(bson.M{"_id": c.AlbumId}).One(&album)
	return &album
}
