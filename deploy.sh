#!/bin/bash

PROJECT_ID=<PROJECT_ID>
PUBSUB_TOPIC=<PUBSUB-TOPIC-NAME>
FUNCTION_SA=<CLOUD-FUNCTION-SA-ACCOUNT>
TRIGGER_SA=<EVENTARC-SA-ACCOUNT>
DATASET_ID=<BIGQUERY-DATASET-ID>
TABLE_ID=<BIGQUERY-TABLE-ID>

gcloud functions deploy ee-collect-metrics \
--region=us-central1 \
--gen2 \
--project=${PROJECT_ID} \
--trigger-topic=${PUBSUB_TOPIC} \
--trigger-location=us-central1 \
--trigger-service-account=${TRIGGER_SA} \
--runtime=go119 \
--source=. \
--entry-point=ReadTimeSeriesFields \
--region=us-central1 \
--service-account=${FUNCTION_SA} \
--set-env-vars=FUNCTION_TARGET=ReadTimeSeriesFields,PROJECTID=${PROJECT_ID},TABLEID=${TABLE_ID},DATASETID=${DATASET_ID},TIMEWINDOW=30m,AGGREGATION=1800 \
--timeout=90 \
--memory=256MB \
--ingress-settings=internal-only
