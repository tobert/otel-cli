#!/bin/bash
# an otel-cli log demo

# Generate some sample data to include in the log
user_id=$(( RANDOM % 1000 ))
action="login"

# Send a log message with attributes
../otel-cli log \
	--service  "otel-cli-demo"    \
	--body     "User login successful" \
	--severity INFO               \
	--attrs "user.id=$user_id,action=$action,ip=192.168.1.100"

# Example with different severity levels
../otel-cli log \
	--service  "otel-cli-demo"    \
	--body     "This is a warning message" \
	--severity WARN               \
	--attrs "component=auth,issue=rate_limit"

../otel-cli log \
	--service  "otel-cli-demo"    \
	--body     "Critical error occurred" \
	--severity ERROR              \
	--attrs "component=database,error=connection_timeout"
