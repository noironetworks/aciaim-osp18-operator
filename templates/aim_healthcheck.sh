#!/bin/sh

aim_aid_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-aid | awk -F ' ' '{print $2}')
aim_event_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-event | awk -F ' ' '{print $2}')
aim_rpc_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-rpc | awk -F ' ' '{print $2}')

if [ "$aim_aid_status" != "RUNNING" ] || [ "$aim_event_status" != "RUNNING" ] || [ "$aim_rpc_status" != "RUNNING" ]; then
   echo "fail"
   exit 1
fi
