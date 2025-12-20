package job

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
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
		if len(c.Errors) > 0 {
			err := c.Errors[0]
			if apiErr, ok := err.Err.(common.APIError); ok {
				c.JSON(apiErr.Status, apiErr)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
		}
		return
	}

	if err := h.service.CreateJob(c.Request.Context(), &req); err != nil {
		if apiErr, ok := err.(common.APIError); ok {
			c.JSON(apiErr.Status, apiErr)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, req)
}

// Get handles HTTP requests for getting a job based on an id.
func (h *JobHandler) Get(c *gin.Context) {

	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, common.APIError{Message: "Invalid ID"})
		return
	}
	resp, err := h.service.GetJobByID(c.Request.Context(), uint(id))

	if err != nil {
		c.JSON(http.StatusNotFound, common.APIError{Message: "Job not found"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// TODO:
func (h *JobHandler) Update(c *gin.Context) {}

// TODO:
func (h *JobHandler) Increment(c *gin.Context) {}

// TODO:
func (h *JobHandler) Save(c *gin.Context) {}

// TODO:
func (h *JobHandler) List(c *gin.Context) {}
