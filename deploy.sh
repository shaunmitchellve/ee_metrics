#!/bin/bash

gcloud functions deploy ee-collect-metrics \
--region=us-central1 \
--gen2 \
--runtime=go119 \
--source=. \
--entry-point=ReadTimeSeriesFields \
--trigger-topic=start-ee-metric-collection \
--region=us-central1 \
--service-account=ee-collect-metrics-runner@ce-monitoring.iam.gserviceaccount.com \
--set-env-vars=FUNCTION_TARGET=ReadTimeSeriesFields,PROJECTID=ce-monitoring,TABLEID=cpu_usage_time,DATASETID=earth_engine_metrics,TIMEWINDOW=30m,AGGREGATION=1800 \
--timeout=90 \
--memory=256MB \
--ingress-settings=internal-only
