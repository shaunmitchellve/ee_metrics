# Earth Engine Monitoring - Time Series Data Pull

## Reference Documentation:

- [Filtering and aggregation: manipulating time series](https://cloud.google.com/monitoring/api/v3/aggregation)
- [Earth Engine Monitoring Metric](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-earthengine)
- [Monitoring API project time series](https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.timeSeries/list)

## Setup

1. You need to make sure the following APIs are enabled:

- cloudscheduler.googleapis.com
- bigquery.googleapis.com
- eventarc.googleapis.com
- cloudfunctions.googleapis.com
- run.googleapis.com

This setup assumes it's being run in the same project as your centralized Cloud Monitoring for Earth Engine. Basically, all EE metrics are being written to this project metwrics explorer.

```
gcloud services enable SERVICE
```

2. Create 2 Service Accounts:

We need a Cloud Function SA and a EventArc SA:

```
gcloud iam service-accounts create CLOUD_FUNCTION_SA_NAME
gcloud iam service-accounts create EVENTARC_SA_NAME
```

3. Let's grant the correct permissions:

```
gcloud projects add-iam-policy-binding PROJECT_ID \
--member="serviceAccount:CLOUD_FUNCTION_SA_NAME@PROJECT_ID.iam.gserviceaccount.com" \
--role="roles/monitoring.viewer"

gcloud projects add-iam-policy-binding PROJECT_ID \
--member="serviceAccount:EVENTARC_SA_NAME@PROJECT_ID.iam.gserviceaccount.com" \
--role="roles/run.invoker"
```

4. Now create the BigQuery Dataset and Table (If you already have a BQ dataset, you can skip the dataset creation):

```
bq --location=LOCATION_REGION mk \
--dataset \
PROJECT_ID:DATASET_ID

bq mk \
--table \
PROJECT_ID:DATASET_ID.TABLE_ID \
./bq_schema.json
```

5. More permissions needed now, follow the instructions in these docs [https://cloud.google.com/bigquery/docs/control-access-to-resources-iam#grant_access_to_a_dataset](https://cloud.google.com/bigquery/docs/control-access-to-resources-iam#grant_access_to_a_dataset)

- Grant `READER` on the dataset to the CLOUD_FUNCTION_SA_NAME@PROJECT_ID.iam.gserviceaccount.com service account
- Grant `OWNER` on the table to the CLOUD_FUNCTION_SA_NAME@PROJECT_ID.iam.gserviceaccount.com service account.

6. Let's create the PubSub topic

```
gcloud pubsub topics create TOPIC
```

7. Now, we can edit the `deploy.sh` file with the correct information from above and run the script to deploy the cloud function. Edite the file and adjust the variables at the top of the file.

- PROJECT_ID= PROJECT_ID
- PUBSUB_TOPIC= PUBSUB-TOPIC-NAME
- FUNCTION_SA= CLOUD-FUNCTION-SA-ACCOUNT - Make sure this is fully qualified email address
- TRIGGER_SA= EVENTARC-SA-ACCOUNT - Make sure this is fully qualified email address
- DATASET_ID= BIGQUERY-DATASET-ID
- TABLE_ID= BIGQUERY-TABLE-ID

Execute the shell script.

```
./deploy.sh
```

8. Last we need to create a Cloud SCheduler to run the function

```
gcloud scheduler jobs create pubsub JOB-NAME \
--location=LOCATION_REGION \
--schedule="*/30 * * * *" \
--topic=PUBSUB_TOPIC \
--message-body="Run"
--time-zone="UTC-5"
```

That should be it. I may have missed something so do let me know if something doesn't work properly.

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

