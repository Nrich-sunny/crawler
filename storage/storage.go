package storage

type DataCell struct {
	Data map[string]interface{}
}

func (d *DataCell) GetTableName() string {
	return d.Data["Task"].(string)
}

func (d *DataCell) GetTaskName() string {
	return d.Data["Task"].(string)
}

// Storage 数据存储的接口
type Storage interface {
	Save(datas ...*DataCell) error
}
