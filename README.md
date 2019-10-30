To reproduce, run `go run -mod=vendor .`

Vendor is needed because I have added a debug lines to [datadog.go](./vendor/github.com/DataDog/opencensus-go-exporter-datadog/datadog.go#L35:L41)'s ExportView method to show that the MIN/MAX never change for the entre lifetime of the process.

To see the Min/Max fixed, uncomment [worker.go](./vendor/github.com/DataDog/opencensus-go-exporter-datadog/worker.go) in OpenCensus line 236