// Package types mirrors @medusajs/workflow-engine-redis/src/types.
package types

// RedisWorkflowsOptions mirrors RedisWorkflowsOptions (the `redis` module
// option).
type RedisWorkflowsOptions struct {
	URL                  string
	RedisURL             string
	Options              map[string]any
	RedisOptions         map[string]any
	QueueName            string
	JobQueueName         string
	QueueOptions         map[string]any
	WorkerOptions        map[string]any
	MainQueueOptions     map[string]any
	MainWorkerOptions    map[string]any
	JobQueueOptions      map[string]any
	JobWorkerOptions     map[string]any
	CleanerQueueOptions  map[string]any
	CleanerWorkerOptions map[string]any
	PubSubURL            string
	PubSubOptions        map[string]any
}

// OptionsFromMap reads the module options ({ redis: { ... } }).
func OptionsFromMap(m map[string]any) RedisWorkflowsOptions {
	var o RedisWorkflowsOptions
	r, _ := m["redis"].(map[string]any)
	if r == nil {
		return o
	}
	str := func(k string) string { s, _ := r[k].(string); return s }
	obj := func(k string) map[string]any { v, _ := r[k].(map[string]any); return v }
	o.URL = str("url")
	o.RedisURL = str("redisUrl")
	o.Options = obj("options")
	o.RedisOptions = obj("redisOptions")
	o.QueueName = str("queueName")
	o.JobQueueName = str("jobQueueName")
	o.QueueOptions = obj("queueOptions")
	o.WorkerOptions = obj("workerOptions")
	o.MainQueueOptions = obj("mainQueueOptions")
	o.MainWorkerOptions = obj("mainWorkerOptions")
	o.JobQueueOptions = obj("jobQueueOptions")
	o.JobWorkerOptions = obj("jobWorkerOptions")
	o.CleanerQueueOptions = obj("cleanerQueueOptions")
	o.CleanerWorkerOptions = obj("cleanerWorkerOptions")
	if ps := obj("pubsub"); ps != nil {
		o.PubSubURL, _ = ps["url"].(string)
		o.PubSubOptions, _ = ps["options"].(map[string]any)
	}
	return o
}

// Merge mirrors { ...shared, ...perQueue }.
func Merge(shared, perQueue map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range shared {
		out[k] = v
	}
	for k, v := range perQueue {
		out[k] = v
	}
	return out
}
