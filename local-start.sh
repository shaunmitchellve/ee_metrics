#!/bin/bash
export FUNCTION_TARGET=ReadTimeSeriesFields
export PROJECTID=ce-monitoring
export TABLEID=cpu_usage_time
export DATASETID=earth_engine_metrics
export TIMEWINDOW=30m
export AGGREGATION=1800
go run cmd/main.go