// Code generated by protoc-gen-zap (etc/proto/protoc-gen-zap). DO NOT EDIT.
//
// source: debug/debug.proto

package debug

import (
	protoextensions "github.com/pachyderm/pachyderm/v2/src/protoextensions"
	zapcore "go.uber.org/zap/zapcore"
)

func (x *ProfileRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("profile", x.Profile)
	enc.AddObject("filter", x.Filter)
	return nil
}

func (x *Profile) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("name", x.Name)
	protoextensions.AddDuration(enc, "duration", x.Duration)
	return nil
}

func (x *Filter) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddBool("pachd", x.GetPachd())
	enc.AddObject("pipeline", x.GetPipeline())
	enc.AddObject("worker", x.GetWorker())
	enc.AddBool("database", x.GetDatabase())
	return nil
}

func (x *Worker) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("pod", x.Pod)
	enc.AddBool("redirected", x.Redirected)
	return nil
}

func (x *BinaryRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("filter", x.Filter)
	return nil
}

func (x *DumpRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("filter", x.Filter)
	enc.AddInt64("limit", x.Limit)
	return nil
}

func (x *SetLogLevelRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("pachyderm", x.GetPachyderm().String())
	enc.AddString("grpc", x.GetGrpc().String())
	protoextensions.AddDuration(enc, "duration", x.Duration)
	enc.AddBool("recurse", x.Recurse)
	return nil
}

func (x *SetLogLevelResponse) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	affected_podsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.AffectedPods {
			enc.AppendString(v)
		}
		return nil
	}
	enc.AddArray("affected_pods", zapcore.ArrayMarshalerFunc(affected_podsArrMarshaller))
	errored_podsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.ErroredPods {
			enc.AppendString(v)
		}
		return nil
	}
	enc.AddArray("errored_pods", zapcore.ArrayMarshalerFunc(errored_podsArrMarshaller))
	return nil
}

func (x *GetDumpV2TemplateRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	filtersArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Filters {
			enc.AppendString(v)
		}
		return nil
	}
	enc.AddArray("filters", zapcore.ArrayMarshalerFunc(filtersArrMarshaller))
	return nil
}

func (x *GetDumpV2TemplateResponse) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("request", x.Request)
	return nil
}

func (x *WorkerDump) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("pod", x.Pod)
	return nil
}

func (x *Pipeline) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("project", x.Project)
	enc.AddString("name", x.Name)
	workersArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Workers {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("workers", zapcore.ArrayMarshalerFunc(workersArrMarshaller))
	return nil
}

func (x *Pod) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("name", x.Name)
	enc.AddString("ip", x.Ip)
	containersArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Containers {
			enc.AppendString(v)
		}
		return nil
	}
	enc.AddArray("containers", zapcore.ArrayMarshalerFunc(containersArrMarshaller))
	return nil
}

func (x *App) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("name", x.Name)
	podsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Pods {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("pods", zapcore.ArrayMarshalerFunc(podsArrMarshaller))
	enc.AddInt64("timeout", x.Timeout)
	enc.AddObject("pipeline", x.Pipeline)
	return nil
}

func (x *System) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddBool("helm", x.Helm)
	enc.AddBool("database", x.Database)
	versionsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Versions {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("versions", zapcore.ArrayMarshalerFunc(versionsArrMarshaller))
	describesArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Describes {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("describes", zapcore.ArrayMarshalerFunc(describesArrMarshaller))
	logsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Logs {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("logs", zapcore.ArrayMarshalerFunc(logsArrMarshaller))
	loki_logsArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.LokiLogs {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("loki_logs", zapcore.ArrayMarshalerFunc(loki_logsArrMarshaller))
	binariesArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Binaries {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("binaries", zapcore.ArrayMarshalerFunc(binariesArrMarshaller))
	profilesArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Profiles {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("profiles", zapcore.ArrayMarshalerFunc(profilesArrMarshaller))
	return nil
}

func (x *DumpV2Request) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("system", x.System)
	pipelinesArrMarshaller := func(enc zapcore.ArrayEncoder) error {
		for _, v := range x.Pipelines {
			enc.AppendObject(v)
		}
		return nil
	}
	enc.AddArray("pipelines", zapcore.ArrayMarshalerFunc(pipelinesArrMarshaller))
	enc.AddBool("input_repos", x.InputRepos)
	enc.AddInt64("timeout", x.Timeout)
	return nil
}

func (x *DumpContent) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	protoextensions.AddBytes(enc, "content", x.Content)
	return nil
}

func (x *DumpProgress) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddString("task", x.Task)
	enc.AddInt64("total", x.Total)
	enc.AddInt64("progress", x.Progress)
	return nil
}

func (x *DumpChunk) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if x == nil {
		return nil
	}
	enc.AddObject("content", x.GetContent())
	enc.AddObject("progress", x.GetProgress())
	return nil
}
