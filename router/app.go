package router

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"goChat/docs"
	"goChat/middleware"
	"goChat/service"
)

func Router() *gin.Engine {
	r := gin.Default()

	//swagger
	docs.SwaggerInfo.BasePath = ""
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	//静态资源
	r.Static("/asset", "asset/")
	r.StaticFile("/favicon.ico", "asset/images/favicon.ico")
	//	r.StaticFS()
	r.LoadHTMLGlob("views/**/*")

	//index
	r.GET("/", service.GetIndex)
	r.GET("/index", service.GetIndex)
	r.GET("/toRegister", service.ToRegister)
	r.GET("/toChat", middleware.AuthRequired(), service.ToChat)
	r.GET("/chat", middleware.AuthRequired(), service.Chat)

	//public user endpoints
	r.POST("/user/createUser", service.CreateUser)
	r.POST("/user/findUserByNameAndPwd", service.FindUserByNameAndPwd)

	authorized := r.Group("/")
	authorized.Use(middleware.AuthRequired())
	{
		authorized.POST("/searchFriends", service.SearchFriends)
		authorized.POST("/user/getUserList", service.GetUserList)
		authorized.POST("/user/deleteUser", service.DeleteUser)
		authorized.POST("/user/updateUser", service.UpdateUser)
		authorized.POST("/user/find", service.FindByID)
		authorized.GET("/user/sendUserMsg", service.SendUserMsg)
		authorized.POST("/contact/addfriend", service.AddFriend)
		authorized.POST("/attach/upload", service.Upload)
		authorized.POST("/contact/createCommunity", service.CreateCommunity)
		authorized.POST("/contact/loadcommunity", service.LoadCommunity)
		authorized.POST("/contact/joinGroup", service.JoinGroups)
		authorized.POST("/user/redisMsg", service.RedisMsg)
		authorized.POST("/user/RedisGroupMsg", service.RedisGroupMsg)

		authorized.GET("/api/ai/templates", service.ListAICharacterTemplates)
		authorized.POST("/api/ai/characters", service.CreateAICharacter)
		authorized.GET("/api/ai/characters", service.ListAICharacters)
		authorized.PUT("/api/ai/characters/:id", service.UpdateAICharacter)
		authorized.DELETE("/api/ai/characters/:id", service.DeleteAICharacter)
		authorized.POST("/api/ai/conversations", service.CreateAIConversation)
		authorized.GET("/api/ai/conversations", service.ListAIConversations)
		authorized.DELETE("/api/ai/conversations/:id", service.DeleteAIConversation)
		authorized.GET("/api/ai/conversations/:id/messages", service.ListAIMessages)
		authorized.POST("/api/ai/conversations/:id/messages", service.SendAIMessage)
		authorized.POST("/api/ai/conversations/:id/messages/:messageId/retry", service.RetryAIMessage)
	}

	return r
}
