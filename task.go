package get

type DownloadTask struct {
	Link string
	Path string
	Err  error
}

func NewDownload(ref, path string) *DownloadTask {
	return &DownloadTask{
		Link: ref,
		Path: path,
	}
}

type Downloads struct {
	List []*DownloadTask
}

func (d *Downloads) Add(ref, path string) {
	d.List = append(d.List, NewDownload(ref, path))
}

func NewDownloads() *Downloads {
	return &Downloads{}
}
