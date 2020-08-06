package reporter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"emperror.dev/errors"
	"github.com/google/uuid"
	"github.com/meirf/gopart"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	marketplacev1alpha1 "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/apis/marketplace/v1alpha1"
	"github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils"
	loggerf "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/utils/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	additionalLabels = []model.LabelName{"pod", "namespace", "service"}
	logger           = loggerf.NewLogger("reporter")
)

// Goals of the reporter:
// Get current meters to query
//
// Build a report for hourly since last reprot
//
// Query all the meters
//
// Break up reports into manageable chunks
//
// Upload to insights
//
// Update the CR status for each report and queue

type MarketplaceReporter struct {
	api               v1.API
	k8sclient         client.Client
	mktconfig         *marketplacev1alpha1.MarketplaceConfig
	report            *marketplacev1alpha1.MeterReport
	meterDefinitions  []marketplacev1alpha1.MeterDefinition
	prometheusService *corev1.Service
	*Config
}

type ReportName types.NamespacedName

func NewMarketplaceReporter(
	config *Config,
	k8sclient client.Client,
	report *marketplacev1alpha1.MeterReport,
	mktconfig *marketplacev1alpha1.MarketplaceConfig,
	meterDefinitions []marketplacev1alpha1.MeterDefinition,
	prometheusService *corev1.Service,
	apiClient api.Client,
) (*MarketplaceReporter, error) {
	return &MarketplaceReporter{
		api:               v1.NewAPI(apiClient),
		k8sclient:         k8sclient,
		mktconfig:         mktconfig,
		report:            report,
		meterDefinitions:  meterDefinitions,
		Config:            config,
		prometheusService: prometheusService,
	}, nil
}

var ErrNoMeterDefinitionsFound = errors.New("no meterDefinitions found")

func (r *MarketplaceReporter) CollectMetrics(ctxIn context.Context) (map[MetricKey]*MetricBase, error) {
	ctx, cancel := context.WithCancel(ctxIn)
	defer cancel()

	resultsMap := make(map[MetricKey]*MetricBase)
	var resultsMapMutex sync.Mutex

	if len(r.meterDefinitions) == 0 {
		return resultsMap, errors.Wrap(ErrNoMeterDefinitionsFound, "no meterDefs found")
	}

	meterDefsChan := make(chan *marketplacev1alpha1.MeterDefinition, len(r.meterDefinitions))
	promModelsChan := make(chan meterDefPromModel)
	errorsChan := make(chan error)
	queryDone := make(chan bool)
	processDone := make(chan bool)

	defer close(errorsChan)

	logger.Info("starting query")

	go r.query(
		ctx,
		r.report.Spec.StartTime.Time,
		r.report.Spec.EndTime.Time,
		meterDefsChan,
		promModelsChan,
		queryDone,
		errorsChan)

	logger.Info("starting processing")

	go r.process(
		ctx,
		promModelsChan,
		resultsMap,
		&resultsMapMutex,
		r.report,
		processDone,
		errorsChan)

	// send & close data pipe
	for _, meterDef := range r.meterDefinitions {
		meterDefsChan <- &meterDef
	}
	close(meterDefsChan)

	errorList := []error{}

	go func() {
		for err := range errorsChan {
			logger.Error(err, "error occurred processing")
			errorList = append(errorList, err)
		}
	}()

	func() {
		for {
			select {
			case <-queryDone:
				logger.Info("querying done")
				close(promModelsChan)
			case <-processDone:
				logger.Info("processing done")
				return
			}
		}
	}()

	if len(errorList) != 0 {
		err := errors.Combine(errorList...)
		logger.Error(err, "processing errored")
		return nil, err
	}

	return resultsMap, nil
}

type meterDefPromModel struct {
	*marketplacev1alpha1.MeterDefinition
	model.Value
	MetricName string
}

func (r *MarketplaceReporter) query(
	ctx context.Context,
	startTime, endTime time.Time,
	inMeterDefs <-chan *marketplacev1alpha1.MeterDefinition,
	outPromModels chan<- meterDefPromModel,
	done chan bool,
	errorsch chan<- error,
) {
	queryProcess := func(mdef *marketplacev1alpha1.MeterDefinition) {
		for _, workload := range mdef.Spec.Workloads {
			for _, metric := range workload.MetricLabels {
				logger.Info("query", "metric", metric)
				// TODO: use metadata to build a smart roll up
				// Guage = delta
				// Counter = increase
				// Histogram and summary are unsupported
				query := &PromQuery{
					Metric: metric.Label,
					Type:   workload.WorkloadType,
					MeterDef: struct{ Name, Namespace string }{
						Name:      mdef.Name,
						Namespace: mdef.Namespace,
					},
					Labels:        metric.Query,
					Time:          "60m",
					Start:         startTime,
					End:           endTime,
					Step:          time.Hour,
					AggregateFunc: metric.Aggregation,
				}
				logger.Info("output", "query", query.String())

				var val model.Value
				var warnings v1.Warnings

				err := utils.Retry(func() error {
					var err error
					val, warnings, err = r.queryRange(query)

					if err != nil {
						return errors.Wrap(err, "error with query")
					}

					return nil
				}, *r.Retry)

				if warnings != nil {
					logger.Info("warnings %v", warnings)
				}

				if err != nil {
					logger.Error(err, "error encountered")
					errorsch <- err
					return
				}

				outPromModels <- meterDefPromModel{mdef, val, metric.Label}
			}
		}
	}

	wgWait(ctx, "queryProcess", *r.MaxRoutines, done, func() {
		for mdef := range inMeterDefs {
			queryProcess(mdef)
		}
	})
}

func (r *MarketplaceReporter) process(
	ctx context.Context,
	inPromModels <-chan meterDefPromModel,
	results map[MetricKey]*MetricBase,
	mutex sync.Locker,
	report *marketplacev1alpha1.MeterReport,
	done chan bool,
	errorsch chan error,
) {
	syncProcess := func(
		name string,
		mdef *marketplacev1alpha1.MeterDefinition,
		report *marketplacev1alpha1.MeterReport,
		m model.Value,
	) {
		//# do the work
		switch m.Type() {
		case model.ValMatrix:
			matrixVals := m.(model.Matrix)

			for _, matrix := range matrixVals {
				logger.Debug("adding metric", "metric", matrix.Metric)

				for _, pair := range matrix.Values {
					func() {
						key := MetricKey{
							ReportPeriodStart: report.Spec.StartTime.Format(time.RFC3339),
							ReportPeriodEnd:   report.Spec.EndTime.Format(time.RFC3339),
							IntervalStart:     pair.Timestamp.Time().Format(time.RFC3339),
							IntervalEnd:       pair.Timestamp.Add(time.Hour).Time().Format(time.RFC3339),
							MeterDomain:       mdef.Spec.Group,
							MeterKind:         mdef.Spec.Kind,
						}

						mutex.Lock()
						defer mutex.Unlock()

						base, ok := results[key]

						if !ok {
							base = &MetricBase{
								Key: key,
							}
						}

						logger.Debug("adding pair", "metric", matrix.Metric, "pair", pair)
						metricPairs := []interface{}{name, pair.Value.String()}

						err := base.AddAdditionalLabels(getKeysFromMetric(matrix.Metric, additionalLabels)...)

						if err != nil {
							errorsch <- errors.Wrap(err, "failed adding additional labels")
							return
						}

						err = base.AddMetrics(metricPairs...)

						if err != nil {
							errorsch <- errors.Wrap(err, "failed adding metrics")
							return
						}

						results[key] = base
					}()
				}
			}
		case model.ValString:
		case model.ValVector:
		case model.ValScalar:
		case model.ValNone:
			errorsch <- errors.Errorf("can't process model type=%s", m.Type())
		}
	}

	wgWait(ctx, "syncProcess", *r.MaxRoutines, done, func() {
		for pmodel := range inPromModels {
			syncProcess(pmodel.MetricName, pmodel.MeterDefinition, report, pmodel.Value)
		}
	})
}

func (r *MarketplaceReporter) WriteReport(
	source uuid.UUID,
	metrics map[MetricKey]*MetricBase) ([]string, error) {
	metadata := NewReportMetadata(source, ReportSourceMetadata{
		RhmAccountID: r.mktconfig.Spec.RhmAccountID,
		RhmClusterID: r.mktconfig.Spec.ClusterUUID,
	})

	var partitionSize = *r.MetricsPerFile

	metricsArr := make([]*MetricBase, 0, len(metrics))

	filedir := filepath.Join(r.Config.OutputDirectory, source.String())
	err := os.Mkdir(filedir, 0755)

	if err != nil {
		return []string{}, errors.Wrap(err, "error creating directory")
	}

	for _, v := range metrics {
		metricsArr = append(metricsArr, v)
	}

	filenames := []string{}

	for idxRange := range gopart.Partition(len(metricsArr), partitionSize) {
		metricReport := NewReport()
		metadata.AddMetricsReport(metricReport)

		err := metricReport.AddMetrics(metricsArr[idxRange.Low:idxRange.High]...)

		if err != nil {
			return filenames, err
		}

		metadata.UpdateMetricsReport(metricReport)

		marshallBytes, err := json.Marshal(metricReport)
		logger.Debug(string(marshallBytes))
		if err != nil {
			logger.Error(err, "failed to marshal metrics report", "report", metricReport)
			return nil, err
		}
		filename := filepath.Join(
			filedir,
			fmt.Sprintf("%s.json", metricReport.ReportSliceID.String()))

		err = ioutil.WriteFile(
			filename,
			marshallBytes,
			0600)

		if err != nil {
			logger.Error(err, "failed to write file", "file", filename)
			return nil, errors.Wrap(err, "failed to write file")
		}

		filenames = append(filenames, filename)
	}

	marshallBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.Error(err, "failed to marshal report metadata", "metadata", metadata)
		return nil, err
	}

	filename := filepath.Join(filedir, "metadata.json")
	err = ioutil.WriteFile(filename, marshallBytes, 0600)
	if err != nil {
		logger.Error(err, "failed to write file", "file", filename)
		return nil, err
	}

	filenames = append(filenames, filename)

	return filenames, nil
}

func getKeysFromMetric(metric model.Metric, labels []model.LabelName) []interface{} {
	allLabels := make([]interface{}, 0, len(labels)*2)
	for _, label := range labels {
		if val, ok := metric[label]; ok {
			allLabels = append(allLabels, string(label), string(val))
		}
	}
	return allLabels
}

func wgWait(ctx context.Context, processName string, maxRoutines int, done chan bool, waitFunc func()) {
	var wg sync.WaitGroup
	for w := 1; w <= maxRoutines; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			waitFunc()
		}()
	}

	wait := make(chan bool)
	defer close(wait)

	go func() {
		wg.Wait()
		wait <- true
	}()

	select {
	case <-ctx.Done():
		logger.Info("canceling wg", "name", processName)
	case <-wait:
		logger.Info("wg is done", "name", processName)
	}

	done <- true
}