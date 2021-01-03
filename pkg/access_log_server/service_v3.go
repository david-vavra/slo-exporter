package access_log_server

import (
	"encoding/json"
	"io"

	envoy_service_accesslog_v3 "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/seznam/slo-exporter/pkg/event"
	"github.com/seznam/slo-exporter/pkg/stringmap"
)

type AccessLogServiceV3 struct {
	outChan chan *event.Raw
	logger  logrus.FieldLogger
	envoy_service_accesslog_v3.UnimplementedAccessLogServiceServer
}

func (service_v3 *AccessLogServiceV3) exportLogEntriesV3(msg *envoy_service_accesslog_v3.StreamAccessLogsMessage) []stringmap.StringMap {
	result := []stringmap.StringMap{}

	if logs := msg.GetHttpLogs(); logs != nil {
		for _, l := range logs.LogEntry {
			logEntryJson, err := json.Marshal(l)
			var data interface{}
			json.Unmarshal(logEntryJson, &data)
			if err != nil {
				errorsTotal.WithLabelValues("HTTPLogEntryJsonMarshalling").Inc()
				service_v3.logger.Errorf("Error while marshalling log entry: %v", err)
			}
			metadata, err := unmarshallToMetadata(data, "HTTPLogEntry")
			if err != nil {
				errorsTotal.WithLabelValues("LogEntryProcessing").Inc()
				service_v3.logger.Errorf("Unable to transform log entry to event metadata: %v", err)
				continue
			}
			result = append(result, metadata)

			logEntriesTotal.WithLabelValues("HTTP", "v3").Inc()
		}
	} else if logs := msg.GetTcpLogs(); logs != nil {
		for _, l := range logs.LogEntry {
			logEntryJson, err := json.Marshal(l)
			var data interface{}
			json.Unmarshal(logEntryJson, &data)
			if err != nil {
				errorsTotal.WithLabelValues("TCPLogEntryJsonMarshalling").Inc()
				service_v3.logger.Errorf("Error while marshalling log entry: %v", err)
			}
			metadata, err := unmarshallToMetadata(data, "TCPLogEntry")
			if err != nil {
				errorsTotal.WithLabelValues("LogEntryProcessing").Inc()
				service_v3.logger.Errorf("Unable to transform log entry to event metadata: %v", err)
				continue
			}
			result = append(result, metadata)

			logEntriesTotal.WithLabelValues("TCP", "v3").Inc()
		}
	} else {
		// Unknown access log type
		errorsTotal.WithLabelValues("UnknownLogType").Inc()
		service_v3.logger.Warnf("Unknown log entry type: %s", msg.String())
	}
	return result
}

func (service_v3 *AccessLogServiceV3) StreamAccessLogs(stream envoy_service_accesslog_v3.AccessLogService_StreamAccessLogsServer) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			// TODO verify whether correct
			return nil
		}
		if err != nil {
			errorsTotal.WithLabelValues("ProcessingStream").Inc()
			return err
		}

		for _, singleLogEntryMetadata := range service_v3.exportLogEntriesV3(msg) {
			e := &event.Raw{
				Metadata: singleLogEntryMetadata,
				Quantity: 1,
			}
			service_v3.logger.Debug(e)
			service_v3.outChan <- e
		}

	}
	return nil
}

func (service_v3 *AccessLogServiceV3) Register(server *grpc.Server) {
	envoy_service_accesslog_v3.RegisterAccessLogServiceServer(server, service_v3)
}
