package model
type Resource struct{
	MilliCPU int64 `json:"milli_cpu"`
	Memory int64 `json:"memory"`
}
func (r *Resource) LessThan(other Resource) bool {
	return r.MilliCPU <= other.MilliCPU && r.Memory <= other.Memory	
}