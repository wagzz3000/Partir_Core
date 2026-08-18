// Package telemetry - Standard attribute labels
package telemetry

import "go.opentelemetry.io/otel/attribute"

// Common attributes for Partir telemetry
func LabelTicketID(id string) attribute.KeyValue {
	return attribute.String("ticket_id", id)
}

func LabelRunID(id string) attribute.KeyValue {
	return attribute.String("run_id", id)
}

func LabelPlugin(id string) attribute.KeyValue {
	return attribute.String("plugin", id)
}

func LabelJobType(t string) attribute.KeyValue {
	return attribute.String("job_type", t)
}

func LabelGate(g string) attribute.KeyValue {
	return attribute.String("gate", g)
}

func LabelState(s string) attribute.KeyValue {
	return attribute.String("state", s)
}

func LabelResult(r string) attribute.KeyValue {
	return attribute.String("result", r)
}

func LabelDefectClass(c string) attribute.KeyValue {
	return attribute.String("defect_class", c)
}

func LabelAttempt(n int) attribute.KeyValue {
	return attribute.Int("attempt", n)
}

func LabelModel(m string) attribute.KeyValue {
	return attribute.String("model", m)
}
