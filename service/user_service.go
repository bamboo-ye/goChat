package service

import (
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"goChat/middleware"
	"goChat/models"
	"goChat/utils"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// GetUserList
// @Summary UserList
// @Tags User
// @Success 200 {string} json{"code","message"}
// @Router /user/getUserlist [get]
func GetUserList(ctx *gin.Context) {
	data := models.GetUserList()
	users := make([]gin.H, 0, len(data))
	for _, user := range data {
		users = append(users, gin.H{"id": user.ID, "name": user.Name})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "OK",
		"data":    users,
	})

}

// CreateUser
// @Summary 新增用户
// @Tags 用户模块
// @param name query string false "用户名"
// @param password query string false "密码"
// @param repassword query string false "确认密码"
// @Success 200 {string} json{"code","message"}
// @Router /user/createUser [get]
func CreateUser(c *gin.Context) {

	// user.Name = c.Query("name")
	// password := c.Query("password")
	// repassword := c.Query("repassword")
	user := models.UserBasic{}
	user.Name = c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	repassword := c.Request.FormValue("Identity")
	salt := fmt.Sprintf("%06d", rand.Int31())

	data := models.FindUserByName(user.Name)
	if user.Name == "" || password == "" || repassword == "" {
		c.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "用户名或密码不能为空！",
			"data":    user,
		})
		return
	}
	if data.Name != "" {
		c.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "用户名已注册！",
			"data":    user,
		})
		return
	}
	if password != repassword {
		c.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "两次密码不一致！",
			"data":    user,
		})
		return
	}
	//user.PassWord = password
	user.Password = utils.MakePassword(password, salt)
	user.Salt = salt
	user.LoginTime = time.Now()
	user.LoginOutTime = time.Now()
	user.HeatBeat = time.Now()
	models.CreateUser(user)
	c.JSON(200, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "新增用户成功！",
		"data":    gin.H{"id": user.ID, "name": user.Name},
	})
}

// GetUserList
// @Summary 所有用户
// @Tags 用户模块
// @param name query string false "用户名"
// @param password query string false "密码"
// @Success 200 {string} json{"code","message"}
// @Router /user/findUserByNameAndPwd [post]
func FindUserByNameAndPwd(c *gin.Context) {
	data := models.UserBasic{}

	//name := c.Query("name")
	//password := c.Query("password")
	name := c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	user := models.FindUserByName(name)
	if user.Name == "" {
		c.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "该用户不存在",
			"data":    data,
		})
		return
	}

	flag := utils.ValidPassword(password, user.Salt, user.Password)
	if !flag {
		c.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "密码不正确",
			"data":    data,
		})
		return
	}
	if utils.IsLegacyPassword(user.Password) {
		if err := models.UpgradePassword(user.ID, password); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "升级密码失败"})
			return
		}
	}
	token, err := models.IssueToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": -1, "message": "生成登录凭证失败"})
		return
	}

	c.JSON(200, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "登录成功",
		"data": gin.H{
			"ID":       user.ID,
			"Name":     user.Name,
			"Identity": token,
		},
	})
}

func FindByID(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Request.FormValue("userId"))

	//	name := c.Request.FormValue("name")
	data := models.FindByID(uint(userId))
	utils.RespOK(c.Writer, data, "ok")
}

// DeleteUser
// @Summary DeleteUser
// @Tag User
// @param id query string false "id"
// @Success 200 {string} json{"code","message"}
// @Router /user/deleteUser [get]
func DeleteUser(ctx *gin.Context) {
	user := models.UserBasic{}
	user.ID = middleware.UserID(ctx)

	models.DeleteUser(user)
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "用户删除成功",
		"data":    user,
	})

}

// UpdateUser
// @Summary UpdateUser
// @Tag User
// @param id formData string false "id"
// @param name formData string false "name"
// @param password formData string false "password"
// @param phone formData string false "phone"
// @param email formData string false "email"
// @Success 200 {string} json{"code","message"}
// @Router /user/updateUser [post]
func UpdateUser(ctx *gin.Context) {
	user := models.UserBasic{}
	user.ID = middleware.UserID(ctx)
	user.Name = ctx.PostForm("name")
	user.Password = ctx.PostForm("password")
	user.Phone = ctx.PostForm("phone")
	user.Email = ctx.PostForm("email")
	_, err := govalidator.ValidateStruct(user)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusOK, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "修改不合法",
			"data":    user,
		})
		return
	}
	data := models.FindUserByPhone(user.Phone)
	if data.Phone != "" {
		ctx.JSON(200, gin.H{
			"code":    -1, //  0成功   -1失败
			"message": "该手机号已注册用户！",
			"data":    user,
		})
		return
	}

	models.UpdateUser(user)
	ctx.JSON(http.StatusOK, gin.H{
		"code":    0, //  0成功   -1失败
		"message": "用户修改成功",
		"data":    user,
	})

}

//
//// 防止跨域站点伪造请求
//var upGrader = websocket.Upgrader{
//	CheckOrigin: func(r *http.Request) bool {
//		return true
//	},
//}
//
//func SendMsg(c *gin.Context) {
//	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//	defer func(ws *websocket.Conn) {
//		err = ws.Close()
//		if err != nil {
//			fmt.Println(err)
//		}
//	}(ws)
//	MsgHandler(c, ws)
//}
//func MsgHandler(c *gin.Context, ws *websocket.Conn) {
//	for {
//		msg, err := utils.Subscribe(c, utils.PublishKey)
//		if err != nil {
//			fmt.Println(" MsgHandler 发送失败", err)
//		}
//
//		tm := time.Now().Format("2006-01-02 15:04:05")
//		m := fmt.Sprintf("[ws][%s]:%s", tm, msg)
//		err = ws.WriteMessage(1, []byte(m))
//		if err != nil {
//			log.Fatalln(err)
//		}
//	}
//}

func SendUserMsg(c *gin.Context) {
	models.Chat(c.Writer, c.Request, int64(middleware.UserID(c)))
}

func RedisMsg(c *gin.Context) {
	userIdA := int(middleware.UserID(c))
	userIdB, _ := strconv.Atoi(c.PostForm("userIdB"))
	if !models.IsFriend(uint(userIdA), uint(userIdB)) {
		c.JSON(http.StatusForbidden, gin.H{"code": -1, "message": "无权读取该会话"})
		return
	}

	start, _ := strconv.Atoi(c.PostForm("start"))
	end, _ := strconv.Atoi(c.PostForm("end"))
	isRev, _ := strconv.ParseBool(c.PostForm("isRev"))
	res := models.RedisMsg(int64(userIdA), int64(userIdB), int64(start), int64(end), isRev)
	utils.RespOKList(c.Writer, "ok", res)
}

func RedisGroupMsg(c *gin.Context) {
	userIdA := int(middleware.UserID(c))
	userIdB, _ := strconv.Atoi(c.PostForm("userIdB"))
	if !models.IsGroupMember(uint(userIdA), uint(userIdB)) {
		c.JSON(http.StatusForbidden, gin.H{"code": -1, "message": "无权读取该群聊"})
		return
	}
	start, _ := strconv.Atoi(c.PostForm("start"))
	end, _ := strconv.Atoi(c.PostForm("end"))
	isRev, _ := strconv.ParseBool(c.PostForm("isRev"))
	messages := make([]string, 0)
	for _, v := range models.SearchUserByGroupId(uint(userIdB)) {
		messages = append(messages, models.RedisMsg(int64(userIdA), int64(v), int64(start), int64(end), isRev)...)
	}
	utils.RespOKList(c.Writer, "ok", messages)
}

func SearchFriends(c *gin.Context) {
	users := models.SearchFriend(middleware.UserID(c))
	// c.JSON(200, gin.H{
	// 	"code":    0, //  0成功   -1失败
	// 	"message": "查询好友列表成功！",
	// 	"data":    users,
	// })
	utils.RespOKList(c.Writer, users, len(users))
}

func AddFriend(c *gin.Context) {
	targetName := c.Request.FormValue("targetName")
	code, msg := models.AddFriend(middleware.UserID(c), targetName)
	if code == 0 {
		utils.RespOK(c.Writer, code, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

// 新建群
func CreateCommunity(c *gin.Context) {
	name := c.Request.FormValue("name")
	icon := c.Request.FormValue("icon")
	desc := c.Request.FormValue("desc")
	community := models.Community{}
	community.OwnerId = middleware.UserID(c)
	community.Name = name
	community.Img = icon
	community.Desc = desc
	code, msg := models.CreateCommunity(community)
	if code == 0 {
		utils.RespOK(c.Writer, code, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

// 加载群列表
func LoadCommunity(c *gin.Context) {
	data, msg := models.LoadCommunity(middleware.UserID(c))
	if len(data) != 0 {
		utils.RespList(c.Writer, 0, data, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}

// 加入群 userId uint, comId uint
func JoinGroups(c *gin.Context) {
	comId := c.Request.FormValue("comId")

	data, msg := models.JoinGroup(middleware.UserID(c), comId)
	if data == 0 {
		utils.RespOK(c.Writer, data, msg)
	} else {
		utils.RespFail(c.Writer, msg)
	}
}
