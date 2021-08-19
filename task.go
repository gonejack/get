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
	List []*DownloadTask
}

func (d *DownloadTasks) Add(link, path string) {
	d.List = append(d.List, NewDownloadTask(link, path))
}

func (d *DownloadTasks) ForEach(f func(t *DownloadTask)) {
	for _, t := range d.List {
		f(t)
	}
}

func NewDownloadTasks() *DownloadTasks {
	return &DownloadTasks{}
}
