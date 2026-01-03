package job

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joshu-sajeev/goqueue/common"
	"github.com/joshu-sajeev/goqueue/internal/config"
	"github.com/joshu-sajeev/goqueue/internal/dto"
	"github.com/joshu-sajeev/goqueue/middleware"
	"gorm.io/datatypes"
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

// Get handles HTTP requests to fetch a job by its ID.
// It validates the job ID, calls the JobService, and returns
// HTTP 200 with the job data on success or an appropriate error code.
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

// Update handles HTTP requests to update the status of a job.
// It validates the job ID and request body, delegates the update to the JobService,
// and returns HTTP 204 on success.
func (h *JobHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id < 1 {
		c.Error(common.Errf(http.StatusBadRequest, "invalid ID"))
		return
	}

	var body struct {
		Status config.JobStatus `json:"status" validate:"required"`
	}
	if !middleware.Bind(c, &body) {
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), uint(id), body.Status); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Increment handles HTTP requests to increment the attempt counter of a job.
// It validates the job ID, calls the JobService, and returns HTTP 204 on success.
func (h *JobHandler) Increment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id < 1 {
		c.Error(common.Errf(http.StatusBadRequest, "invalid ID"))
		return
	}

	if err := h.service.IncrementAttempts(c.Request.Context(), uint(id)); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Save handles HTTP requests to save the result and error message for a job.
// It validates the job ID and request body, calls the JobService, and returns HTTP 204 on success.
func (h *JobHandler) Save(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id < 1 {
		c.Error(common.Errf(http.StatusBadRequest, "invalid ID"))
		return
	}

	var body struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if !middleware.Bind(c, &body) {
		return
	}

	// Cast json.RawMessage to datatypes.JSON
	if err := h.service.SaveResult(c.Request.Context(), uint(id), datatypes.JSON(body.Result), body.Error); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// List handles HTTP requests to retrieve all jobs for a given queue.
// It validates the queue query parameter, fetches jobs via JobService,
// and returns them as JSON with HTTP 200.
func (h *JobHandler) List(c *gin.Context) {
	queue := c.Query("queue")
	if queue == "" {
		c.JSON(http.StatusBadRequest, common.APIError{Message: "queue parameter is required"})
		return
	}

	jobs, err := h.service.ListJobs(c.Request.Context(), queue)
	if err != nil {
		c.JSON(http.StatusInternalServerError, common.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, jobs)
}
