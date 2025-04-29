package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"backend/model"
	"backend/service"
)

// PollController 处理投票相关API请求
type PollController struct {
	pollService service.PollService
}

// NewPollController 创建投票控制器
func NewPollController(pollService service.PollService) *PollController {
	return &PollController{
		pollService: pollService,
	}
}

// RegisterRoutes 注册API路由
func (c *PollController) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		// 投票活动管理
		polls := api.Group("/polls")
		{
			polls.POST("", c.CreatePoll)
			polls.GET("", c.ListPolls)
			polls.GET("/:id", c.GetPoll)
			polls.PUT("/:id", c.UpdatePoll)
			polls.DELETE("/:id", c.DeletePoll)

			// 投票操作
			polls.POST("/:id/vote", c.Vote)
			polls.GET("/:id/statistics", c.GetStatistics)
		}
	}
}

// CreatePoll 创建投票活动
// @Summary 创建新投票活动
// @Description 创建新的投票活动及选项
// @Tags polls
// @Accept json
// @Produce json
// @Param poll body model.PollRequest true "投票活动信息"
// @Success 201 {object} model.Poll
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/polls [post]
func (c *PollController) CreatePoll(ctx *gin.Context) {
	var req model.PollRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request: " + err.Error()})
		return
	}

	// 解析时间
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid start time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid end time format"})
		return
	}

	// 创建投票活动
	poll := &model.Poll{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Status:      model.PollStatusDraft,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 创建选项
	options := make([]model.Option, len(req.Options))
	for i, content := range req.Options {
		options[i] = model.Option{
			Content:   content,
			Count:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	poll.Options = options

	// 调用服务创建投票
	createdPoll, err := c.pollService.CreatePoll(ctx, poll)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create poll: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, createdPoll)
}

// GetPoll 获取投票活动详情
// @Summary 获取投票活动详情
// @Description 获取指定ID的投票活动详情
// @Tags polls
// @Produce json
// @Param id path int true "投票活动ID"
// @Success 200 {object} model.Poll
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/polls/{id} [get]
func (c *PollController) GetPoll(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid poll ID"})
		return
	}

	poll, err := c.pollService.GetPollByID(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Error: "Poll not found"})
		return
	}

	ctx.JSON(http.StatusOK, poll)
}

// ListPolls 获取投票活动列表
// @Summary 获取投票活动列表
// @Description 分页获取投票活动列表
// @Tags polls
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页数量，默认10"
// @Success 200 {array} model.Poll
// @Failure 500 {object} ErrorResponse
// @Router /api/polls [get]
func (c *PollController) ListPolls(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	polls, err := c.pollService.ListPolls(ctx, offset, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list polls: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, polls)
}

// Vote 提交投票
// @Summary 提交投票
// @Description 为指定投票活动的选项投票
// @Tags polls
// @Accept json
// @Produce json
// @Param id path int true "投票活动ID"
// @Param vote body model.VoteRequest true "投票信息"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/polls/{id}/vote [post]
func (c *PollController) Vote(ctx *gin.Context) {
	pollID, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid poll ID"})
		return
	}

	var req model.VoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request: " + err.Error()})
		return
	}

	// 确保请求中的投票活动ID与路径参数一致
	req.PollID = pollID

	// 获取用户ID（IP或会话ID）
	userID := ctx.ClientIP()
	if userID == "" {
		userID = ctx.GetHeader("X-Forwarded-For")
	}
	req.UserID = userID

	// 提交投票
	err = c.pollService.SubmitVote(ctx, &req)
	if err != nil {
		switch err {
		case service.ErrPollNotFound:
			ctx.JSON(http.StatusNotFound, ErrorResponse{Error: "Poll not found"})
		case service.ErrOptionNotFound:
			ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Option not found"})
		case service.ErrPollClosed:
			ctx.JSON(http.StatusForbidden, ErrorResponse{Error: "Poll is closed"})
		case service.ErrAlreadyVoted:
			ctx.JSON(http.StatusForbidden, ErrorResponse{Error: "You have already voted"})
		default:
			ctx.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to submit vote: " + err.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Vote submitted successfully"})
}

// GetStatistics 获取投票统计结果
// @Summary 获取投票统计结果
// @Description 获取指定投票活动的统计结果
// @Tags polls
// @Produce json
// @Param id path int true "投票活动ID"
// @Success 200 {object} model.PollStatistics
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/polls/{id}/statistics [get]
func (c *PollController) GetStatistics(ctx *gin.Context) {
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid poll ID"})
		return
	}

	stats, err := c.pollService.GetPollStatistics(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, ErrorResponse{Error: "Poll not found"})
		return
	}

	ctx.JSON(http.StatusOK, stats)
}

// UpdatePoll 更新投票活动
func (c *PollController) UpdatePoll(ctx *gin.Context) {
	// 省略实现
	ctx.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Not implemented"})
}

// DeletePoll 删除投票活动
func (c *PollController) DeletePoll(ctx *gin.Context) {
	// 省略实现
	ctx.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Not implemented"})
}

// ErrorResponse API错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse API成功响应
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
