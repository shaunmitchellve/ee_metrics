package function

import (
	"context"
	"log"
	"os"
	"time"
	"strconv"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/iterator"
	"cloud.google.com/go/bigquery"
	"github.com/cloudevents/sdk-go/v2/event"
)

type Item struct {
	Type 				string		`bigquery:"type"`
	Project_ID  string		`bigquery:"project_id"`
	Metric_Kind	string		`bigquery:"metric_kind"`
	Value_Type	string		`bigquery:"value_type"`
	Labels			[]Label		`bigquery:"labels"`
	Start_Time 	time.Time	`bigquery:"start_time"`
	End_Time		time.Time	`bigquery:"end_time"`
	Value				float32		`bigquery:"value"`
	Export_Time	time.Time	`bigquery:"export_time"`
}

type Label struct {
	Key		string	`bigquery:"key"`
	Value	string	`bigquery:"value"`
}

/*
type Point struct {
	Start_Time 	time.Time	`bigquery:"start_time"`
	End_Time		time.Time	`bigquery:"end_time"`
	Value				float32		`bigquery:"value"`
}*/

func init() {
	functions.CloudEvent("ReadTimeSeriesFields", readTimeSeriesFields)
}

func readTimeSeriesFields(ctx context.Context, e event.Event) error {
	var projectId, tableId, datasetId string
	var timeWindow time.Duration
	var aggregation int64

	if envProjectId := os.Getenv("PROJECTID"); envProjectId != "" {
		projectId = envProjectId
	}

	if envTableId := os.Getenv("TABLEID"); envTableId != "" {
		tableId = envTableId
	}

	if envDataSetId := os.Getenv("DATASETID"); envDataSetId != "" {
		datasetId = envDataSetId
	}

	if envTimeWindow := os.Getenv("TIMEWINDOW"); envTimeWindow != "" {
		timeWindow, _ = time.ParseDuration(envTimeWindow)
	}

	if envAggregation := os.Getenv("AGGREGATION"); envAggregation != "" {
		aggregation, _ = strconv.ParseInt(envAggregation, 10, 64)
	}

	client, err := monitoring.NewMetricClient(ctx)

	if err != nil {
		log.Printf("Unable to create new metric client: %v", err)
		return nil // Returning nil tells the function that it shouldn't retry
	}

	bqClient, err := bigquery.NewClient(ctx, projectId)

	if err != nil {
		log.Printf("Unable to create new BigQuery client: %v", err)
		return nil // Returning nil tells the function that it shouldn't retry
	}

	defer bqClient.Close()
	defer client.Close()

	table := bqClient.Dataset(datasetId).Table(tableId)
	startTime := time.Now().UTC().Add(-1 * timeWindow)
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
				Seconds: aggregation,
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
			log.Printf("Error getting next value in metric api response: %v", err)
			return err
		}

		var labels []Label
		for key, value := range resp.GetMetric().GetLabels() {
			labels = append(labels, Label{
				Key: key,
				Value: value,
			})
		}

		/*var points []Point
		for _, value := range resp.GetPoints() {
			points = append(points, Point{
				Start_Time: time.Unix(value.GetInterval().StartTime.Seconds, 0),
				End_Time: time.Unix(value.GetInterval().EndTime.Seconds, 0),
				Value: float32(value.Value.GetDoubleValue()),
			})
		}*/

		row := &Item{
			Type: resp.GetMetric().GetType(),
			Project_ID: resp.GetResource().GetLabels()["project_id"],
			Metric_Kind: resp.MetricKind.String(),
			Value_Type: resp.ValueType.String(),
			Labels: labels,
			//Points: points,
			Start_Time: time.Unix(resp.GetPoints()[0].GetInterval().StartTime.Seconds, 0),
			End_Time: time.Unix(resp.GetPoints()[0].GetInterval().EndTime.Seconds, 0),
			Value: float32(resp.GetPoints()[0].Value.GetDoubleValue()),
			Export_Time: time.Now(),
		}

		rows = append(rows, row)
	}

	inserter := table.Inserter()
	err = inserter.Put(ctx, rows)

	if err != nil {
		if multiErr, ok := err.(bigquery.PutMultiError); ok {
			for _, putErr := range multiErr {
				log.Printf("failed to insert row %d with err: %v \n", putErr.RowIndex, putErr.Error())
			}
		} else {
			log.Printf("Error inserting data: %v", err)

			return err
		}
	}

	log.Printf("EE Metric Collector completd successfully")
	return nil
}
