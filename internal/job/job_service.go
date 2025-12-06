package job

type JobService struct {
	repo JobRepoInterface
}

func NewJobService(repo JobRepoInterface) *JobService {
	return &JobService{repo: repo}
}

// TODO:
// var _ job.JobRepoInterface = (*JobRepository)(nil)
