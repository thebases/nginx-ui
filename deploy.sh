cd ./dist
ssh root@217.15.164.105  "service nginx-ui stop"
scp -r nginx-ui root@217.15.164.105:/opt/
ssh root@217.15.164.105  "service nginx-ui start && journalctl -f -u nginx-ui"
cd ..