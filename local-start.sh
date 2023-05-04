#!/bin/bash
export FUNCTION_TARGET=ReadTimeSeriesFields
export PROJECTID=<GCP_PROJECT_ID>
export TABLEID=cpu_usage_time
export DATASETID=earth_engine_metrics
export TIMEWINDOW=30m
export AGGREGATION=1800
go run cmd/main.go
