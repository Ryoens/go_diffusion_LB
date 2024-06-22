#!/bin/bash

while true
do
	# after execute vmstat command, save output to variable  
	vmstat_output=$(vmstat 1 2 | tail -n 1)

        # extract id column in cpu_usage from output
	cpu_usage=$(echo "$vmstat_output" | awk '{print $15}')

	# convert cpu_usage to percent type
	cpu_usage_percent=$(expr 100 - $cpu_usage)

        echo "$cpu_usage_percent" > cpu.txt
	# display the result
	echo "CPU Usage: $cpu_usage_percent%"
	sleep 0.5
done

exit 0
