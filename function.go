package function

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
	"cloud.google.com/go/bigquery"
)

type Item struct {
	Type 				string		`bigquery:"type"`
	Project_ID  string		`bigquery:"project_id"`
	Metric_Kind	string		`bigquery:"metric_kind"`
	Value_Type	string		`bigquery:"value_type"`
	Labels			[]Label		`bigquery:"labels"`
	Points			[]Point	`biqquery:"points"`
}

type Label struct {
	Key		string	`bigquery:"key"`
	Value	string	`bigquery:"value"`
}

type Point struct {
	Start_Time 	time.Time	`bigquery:"start_time"`
	End_Time		time.Time	`bigquery:"end_time"`
	Value				float32		`bigquery:"value"`
}

func init() {
	functions.HTTP("ReadTimeSeriesFields", readTimeSeriesFields)
}

func readTimeSeriesFields(w http.ResponseWriter, r *http.Request) {
	var projectId, tableId, datasetId string

	if envProjectId := os.Getenv("PROJECTID"); envProjectId != "" {
		projectId = envProjectId
	}

	if envTableId := os.Getenv("TABLEID"); envTableId != "" {
		tableId = envTableId
	}

	if envDataSetId := os.Getenv("DATASETID"); envDataSetId != "" {
		datasetId = envDataSetId
	}

	ctx := context.Background()
	client, err := monitoring.NewMetricClient(ctx)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	bqClient, err := bigquery.NewClient(ctx, projectId)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	defer bqClient.Close()
	defer client.Close()

	table := bqClient.Dataset(datasetId).Table(tableId)
	startTime := time.Now().UTC().Add(time.Minute * -20)
	endTime := time.Now().UTC()

	req := &monitoringpb.ListTimeSeriesRequest{
		Name:   "projects/" + projectId,
		Filter: `metric.type="earthengine.googleapis.com/project/cpu/usage_time"`,
		Interval: &monitoringpb.TimeInterval{
			StartTime: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
			EndTime: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		},
		Aggregation: &monitoringpb.Aggregation{
			PerSeriesAligner: monitoringpb.Aggregation_ALIGN_SUM,
			AlignmentPeriod: &duration.Duration{
				Seconds: 60,
			},
		},
	}

	it := client.ListTimeSeries(ctx, req)

	rows := []*Item{}

	for {
		resp, err := it.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			break
		}

		var labels []Label
		for key, value := range resp.GetMetric().GetLabels() {
			labels = append(labels, Label{
				Key: key,
				Value: value,
			})
		}

		var points []Point
		for _, value := range resp.GetPoints() {
			points = append(points, Point{
				Start_Time: time.Unix(value.GetInterval().StartTime.Seconds, 0),
				End_Time: time.Unix(value.GetInterval().EndTime.Seconds, 0),
				Value: float32(value.Value.GetDoubleValue()),
			})
		}

		row := &Item{
			Type: resp.GetMetric().GetType(),
			Project_ID: resp.GetResource().GetLabels()["project_id"],
			Metric_Kind: resp.MetricKind.String(),
			Value_Type: resp.ValueType.String(),
			Labels: labels,
			Points: points,
		}

		rows = append(rows, row)
	}

	inserter := table.Inserter()
	err = inserter.Put(ctx, rows)

	if err != nil {
		if multiErr, ok := err.(bigquery.PutMultiError); ok {
			for _, putErr := range multiErr {
				fmt.Printf("failed to insert row %d with err: %v \n", putErr.RowIndex, putErr.Error())
			}
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "success")
}
