package generator

import "notequizai/internal/models"

type Task struct {
	Index int
	Topic models.Topic
	Type  models.QuestionType
	Count int
}

type Result struct {
	Task      Task
	Questions []models.Question
	Err       error
}

func buildTasks(topics []models.Topic, specs []models.QuestionSpec) []Task {
	tasks := make([]Task, 0, len(topics)*len(specs))
	for _, topic := range topics {
		for _, spec := range specs {
			tasks = append(tasks, Task{
				Index: len(tasks),
				Topic: topic,
				Type:  spec.Type,
				Count: spec.Count,
			})
		}
	}
	return tasks
}
