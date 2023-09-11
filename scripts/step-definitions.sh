#!/usr/bin/env bash

for i in StepResult Step Reject Resolve  ; do
	oak go definitions --recurse --name "$i"  pkg/steps/step.go pkg/helpers --with-body
done

cat pkg/helpers/result.go

for i in ExtractJSONStep; do
	oak go definitions --recurse --name "$i"  pkg/ --with-body
done

for i in Bind; do
	oak go definitions --recurse --name "$i"  pkg/steps/step.go pkg/helpers
done


