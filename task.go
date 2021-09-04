package get

type DownloadTask struct {
	Link string
	Path string
	Err  error
}

func NewDownloadTask(link, path string) *DownloadTask {
	return &DownloadTask{
		Link: link,
		Path: path,
	}
}

type DownloadTasks struct {
	tasks []*DownloadTask
}

func (d *DownloadTasks) Add(link, path string) {
	for _, t := range d.tasks {
		if t.Link == link && t.Path == path {
			return
		}
	}
	d.tasks = append(d.tasks, NewDownloadTask(link, path))
}

func (d *DownloadTasks) ForEach(f func(t *DownloadTask)) {
	for _, t := range d.tasks {
		f(t)
	}
}

func NewDownloadTasks() *DownloadTasks {
	return &DownloadTasks{}
}
