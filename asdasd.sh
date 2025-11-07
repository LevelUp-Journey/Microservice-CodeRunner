
docker run -d  --name code-runner  --privileged  -p 8084:8084  -p 9084:9084  -v /var/run/docker.sock:/var/run/docker.sock  jonatanfd/code-runner-service-pre
