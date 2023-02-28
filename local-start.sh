#!/bin/bash
export FUNCTION_TARGET=ReadTimeSeriesFields
export PROJECTID=ce-monitoring
export TABLEID=cpu_usage_time
export DATASETID=earth_engine_metrics
go run cmd/main.go