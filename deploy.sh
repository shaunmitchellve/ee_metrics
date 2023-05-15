#!/bin/bash

PROJECT_ID=<GCP-PROJECT-ID>
PUBSUB_TRIGGER=<TRIGGER-NAME>
SA=<SERVICE-ACCOUNT>
DATASETID=<DATASET_ID>
TABLE_ID=<TABLE_ID>



gcloud functions deploy ee-collect-metrics \
--region=us-central1 \
--gen2 \
--project=${PROJECT_ID} \
--runtime=go119 \
--source=. \
--entry-point=ReadTimeSeriesFields \
--trigger-topic=${PUBSUB_TRIGGER} \
--region=us-central1 \
--service-account=${SA} \
--set-env-vars=FUNCTION_TARGET=ReadTimeSeriesFields,PROJECTID=${PROJECT_ID},TABLEID=${TABLE_ID},DATASETID=${DATASET_ID},TIMEWINDOW=30m,AGGREGATION=1800 \
--timeout=90 \
--memory=256MB \
--ingress-settings=internal-only
