# Loadbalancer

First attempt at writing a basic loadbalancer. Includes:
- TCP and HTTP proxying
- Prometheus metrics
- Dockerfile

To run the Docker image and attach (must have built locally first):
`docker run -it --rm -p 5353:5353 -p 8090:8090 -p 8091:8091 -v $(pwd)/config.yaml:/opt/loadbalancer/config.yaml loadbalancer`
