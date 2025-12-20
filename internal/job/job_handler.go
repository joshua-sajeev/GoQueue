package job

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/middleware"
)

type JobHandler struct {
	service JobServiceInterface
}

func NewJobHandler(s JobServiceInterface) *JobHandler {
	return &JobHandler{service: s}
}

// var _ JobServiceInterface := (*JobHandler )(nil)
var _ JobHandlerInterface = (*JobHandler)(nil)

// Create handles HTTP requests for creating a new job.
// It validates and binds the request body, delegates business logic
// to the JobService, and returns HTTP 201 on successful creation.
func (h *JobHandler) Create(c *gin.Context) {
	var req dto.JobCreateDTO

	if !middleware.Bind(c, &req) {
		c.Abort()
		return
	}

	if err := h.service.CreateJob(c.Request.Context(), &req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	c.JSON(http.StatusCreated, req)
}

// TODO:
func (h *JobHandler) Get(c *gin.Context) {}

// TODO:
func (h *JobHandler) Update(c *gin.Context) {}

// TODO:
func (h *JobHandler) Increment(c *gin.Context) {}

// TODO:
func (h *JobHandler) Save(c *gin.Context) {}

// TODO:
func (h *JobHandler) List(c *gin.Context) {}
