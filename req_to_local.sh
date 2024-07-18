#!/bin/bash

#server="127.0.0.1:8000" # to localhost
server="10.10.0.2:8000" # to container

count=0
start_time=`date "+%Y-%m-%d %H:%M:%S"`

while true
do
  
  if [ $count -eq 270000 ]; then
    end_time=`date "+%Y-%m-%d %H:%M:%S"`
    echo "finish" $start_time $end_time
    break
  else
    curl -X GET -m 300 -s -o /dev/null "$server" &
    curl -X GET -m 300 -s -o /dev/null "$server" &
    curl -X GET -m 300 -s -o /dev/null "$server" &
    curl -X GET -m 300 -s -o /dev/null "$server" &
    curl -X GET -m 300 -s -o /dev/null "$server" &
    
    count=$(expr $count + 5)
    echo $count
    
    #echo "+++"
    sleep 0.001
  fi

done
