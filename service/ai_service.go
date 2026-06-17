package service

import (
	"errors"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"goChat/ai"
	"goChat/middleware"
)

var (
	aiServiceOnce sync.Once
	aiService     *ai.Service
)

func getAIService() *ai.Service {
	aiServiceOnce.Do(func() {
		aiService = ai.NewServiceFromConfig()
	})
	return aiService
}

func ListAICharacterTemplates(c *gin.Context) {
	templates, err := getAIService().ListTemplates()
	respondAI(c, templates, err, http.StatusOK)
}

func CreateAICharacter(c *gin.Context) {
	var input ai.CharacterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondAI(c, nil, ai.ErrInvalidInput, http.StatusCreated)
		return
	}
	character, err := getAIService().CreateCharacter(middleware.UserID(c), input)
	respondAI(c, character, err, http.StatusCreated)
}

func ListAICharacters(c *gin.Context) {
	characters, err := getAIService().ListCharacters(middleware.UserID(c))
	respondAI(c, characters, err, http.StatusOK)
}

func UpdateAICharacter(c *gin.Context) {
	characterID, ok := pathID(c)
	if !ok {
		return
	}
	var input ai.CharacterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respondAI(c, nil, ai.ErrInvalidInput, http.StatusOK)
		return
	}
	character, err := getAIService().UpdateCharacter(middleware.UserID(c), characterID, input)
	respondAI(c, character, err, http.StatusOK)
}

func DeleteAICharacter(c *gin.Context) {
	characterID, ok := pathID(c)
	if !ok {
		return
	}
	err := getAIService().DeleteCharacter(middleware.UserID(c), characterID)
	respondAI(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

type createConversationRequest struct {
	CharacterID uint   `json:"characterId"`
	Title       string `json:"title"`
}

func CreateAIConversation(c *gin.Context) {
	var input createConversationRequest
	if err := c.ShouldBindJSON(&input); err != nil || input.CharacterID == 0 {
		respondAI(c, nil, ai.ErrInvalidInput, http.StatusCreated)
		return
	}
	conversation, err := getAIService().CreateConversation(middleware.UserID(c), input.CharacterID, input.Title)
	respondAI(c, conversation, err, http.StatusCreated)
}

func ListAIConversations(c *gin.Context) {
	conversations, err := getAIService().ListConversations(middleware.UserID(c))
	respondAI(c, conversations, err, http.StatusOK)
}

func DeleteAIConversation(c *gin.Context) {
	conversationID, ok := pathID(c)
	if !ok {
		return
	}
	err := getAIService().DeleteConversation(middleware.UserID(c), conversationID)
	respondAI(c, gin.H{"deleted": err == nil}, err, http.StatusOK)
}

func ListAIMessages(c *gin.Context) {
	conversationID, ok := pathID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "30"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	messages, err := getAIService().ListMessages(middleware.UserID(c), conversationID, limit, offset)
	respondAI(c, messages, err, http.StatusOK)
}

type sendAIMessageRequest struct {
	Content string `json:"content"`
}

func SendAIMessage(c *gin.Context) {
	conversationID, ok := pathID(c)
	if !ok {
		return
	}
	var input sendAIMessageRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		respondAI(c, nil, ai.ErrInvalidInput, http.StatusOK)
		return
	}
	message, err := getAIService().SendMessage(c.Request.Context(), middleware.UserID(c), conversationID, input.Content)
	respondAI(c, message, err, http.StatusOK)
}

func RetryAIMessage(c *gin.Context) {
	conversationID, ok := pathID(c)
	if !ok {
		return
	}
	messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 64)
	if err != nil || messageID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的消息 ID"})
		return
	}
	message, retryErr := getAIService().RetryMessage(
		c.Request.Context(), middleware.UserID(c), conversationID, uint(messageID),
	)
	respondAI(c, message, retryErr, http.StatusOK)
}

func pathID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "message": "无效的资源 ID"})
		return 0, false
	}
	return uint(value), true
}

func respondAI(c *gin.Context, data interface{}, err error, successStatus int) {
	if err == nil {
		c.JSON(successStatus, gin.H{"code": 0, "data": data})
		return
	}
	status := http.StatusInternalServerError
	message := "AI 服务暂时不可用"
	switch {
	case errors.Is(err, ai.ErrInvalidInput):
		status, message = http.StatusBadRequest, "请求参数不合法"
	case errors.Is(err, ai.ErrNotFound):
		status, message = http.StatusNotFound, "资源不存在"
	case errors.Is(err, ai.ErrConversationBusy):
		status, message = http.StatusConflict, "该会话正在生成回复，请稍后再试"
	case errors.Is(err, ai.ErrQuotaExceeded):
		status, message = http.StatusTooManyRequests, "今日 AI 聊天额度已用完"
	}
	c.JSON(status, gin.H{"code": -1, "message": message})
}
