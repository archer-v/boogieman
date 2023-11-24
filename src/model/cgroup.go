package model

type CGroup struct {
	Name string
	Worker
	Tasks []*Task
}

func (c *CGroup) addTask(task *Task) {
	c.Tasks = append(c.Tasks, task)
}
