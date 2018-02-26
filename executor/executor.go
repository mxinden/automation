package executor

type Executor interface {
	Execute(ExecutionConfiguration) (ExecutionResult, error)
}
