# Earth Engine Monitoring - Time Series Data Pull

## Reference Documentation:

- [Filtering and aggregation: manipulating time series](https://cloud.google.com/monitoring/api/v3/aggregation)
- [Earth Engine Monitoring Metric](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-earthengine)
- [Monitoring API project time series](https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.timeSeries/list)

## Metrics Overview

Google Clouds Monitoring system is a time series based database that stores different types of metrics. For the purposes of this document the metric that is being used is the earthengine.googleapis.com/project/cpu/usage_time. This metric is saved as a double which represents the number of EECUseconds (Earth Engine Compute Units). The metric is DELTA metric which represents the number of EECUs that were used since the last record was saved. For example:

```
[2023-01-01 12:00:00.00] - 0.12 = 12 seconds of EECU, since this is the first entry then the previous entry will have been 0 seconds
[2023-01-01 12:01:00.00] - 1.34 = 1 min, 34 seconds of EECU that were used within the last minute.
```

These values are not cumulative but independent usage for each time period.

Also, Google Cloud metrics are saved for a total of 6 weeks before they are deleted.

## Goal / Outcome

As of the authoring of this document, Google Earth Engine does not export its metrics with labels as part of the billing process into BigQuery. Partners require a way to report on the total EECUs being consumed for each workload type. This Earth Engine workload type metadata can then be used to track / calculate the total (sum) of EECUs consumed for a particular Earth Engine job.

## Grouping / Alignment Period

Since there is too much raw data to properly analyze, we are required to group the data and align the data into a time period. The grouping will be the metadata “workload type”, for obvious reasons mentioned above.

The alignment period needs to be small enough as to be able to provide any reasonable rollup in a detailed report and at the same time be able to be aggregated into a billing cycle. Since we don’t know the exact level of details and billing cycle it’s recommended to keep the alignment period at a lower level, in this case I’m recommending a 60 seconds / 1 min alignment period. With using an alignment period we need to choose an aligner that will return a single value for that time period, since we want to know the total number of EECUs used in that time period we will be using the SUM aligner.

## Reducers

A reducer in time series data is used to combine multiple time series down into a single output. At this time, we are using an alignment period to group our data by a specific time period and we do not wish to combine the different time series (which are separated by labels) into a single output number so no reducer is required.

## Conclusion

The API call / data pull for Earth Engine usage_time metric will be combined into 30 minute aggregations (Initial proposal was 1 minute, however further investigation determined that level of granularity was not required). No further reduction is required as we wish to keep each time series for workload_tag. The output from the API will look something like:

```
     "metric": {
       "labels": {
         "client_type": "ee-py/0.1.321 python/3.8.10",
         "workload_tag": "some-type-of-work",
         "compute_type": "online"
       },
       "type": "earthengine.googleapis.com/project/cpu/usage_time"
     },
     "resource": {
       "type": "earthengine.googleapis.com/Project",
       "labels": {
         "project_id": "ce-datasets",
         "location": "us-central1"
       }
     },
     "metricKind": "DELTA",
     "valueType": "DOUBLE",
     "points": [
       {
         "interval": {
           "startTime": "2023-02-20T11:00:00Z",
           "endTime": "2023-02-20T11:01:00Z"
         },
         "value": {
           "doubleValue": 0.0256482421
         }
       }
     ]
   }
```

